package usecase

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/event"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/knowledge"
	"github.com/secmon-lab/warren/pkg/domain/model/policy"
	"github.com/secmon-lab/warren/pkg/domain/model/prompt"
	"github.com/secmon-lab/warren/pkg/domain/types"
	policySvc "github.com/secmon-lab/warren/pkg/service/policy"
	knowledgeTool "github.com/secmon-lab/warren/pkg/tool/knowledge"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

//go:embed prompt/alert_system_prompt.md
var alertSystemPromptTemplate string

// AlertPipelineResult represents the result of processing a single alert through the pipeline
type AlertPipelineResult struct {
	Alert        *alert.Alert
	EnrichResult policy.EnrichResults
	TriageResult *policy.TriagePolicyResult
}

// ProcessAlertPipeline processes an alert through the complete pipeline.
// This is a pure function without side effects (no DB save, no Slack notification).
//
// Pipeline stages:
// 1. Ingest Policy Evaluation - transforms raw data into Alert objects
// 2. Tag Conversion - converts tag names to tag IDs
// 3. Metadata Generation - fills missing titles/descriptions using LLM
// 4. Enrich Policy Evaluation - executes enrichment tasks (query/agent)
// 5. Triage Policy Evaluation - applies final metadata and determines publish type
//
// All pipeline events are emitted through the notifier for real-time monitoring.
// The notifier receives type-safe events for each stage of processing.
func (uc *UseCases) ProcessAlertPipeline(
	ctx context.Context,
	schema types.AlertSchema,
	alertData any,
	notifier interfaces.Notifier,
) ([]*AlertPipelineResult, error) {
	// Create policy service
	policyService := policySvc.NewWithStrictMode(uc.policyClient, uc.strictAlert)

	// Step 1: Evaluate ingest policy
	alerts, err := policyService.EvaluateIngestPolicy(ctx, schema, alertData)
	if err != nil {
		notifier.NotifyError(ctx, &event.ErrorEvent{
			Error:   err,
			Message: "Ingest policy evaluation failed",
		})
		return nil, goerr.Wrap(err, "failed to evaluate ingest policy")
	}

	// Notify ingest policy result
	notifier.NotifyIngestPolicyResult(ctx, &event.IngestPolicyResultEvent{
		Schema: schema,
		Alerts: alerts,
	})

	if len(alerts) == 0 {
		return []*AlertPipelineResult{}, nil
	}

	// Process each alert through enrich and commit policies
	var results []*AlertPipelineResult
	for _, processedAlert := range alerts {
		// Step 1.4: Set default topic from schema if not already set
		if processedAlert.Topic == "" {
			processedAlert.Topic = types.KnowledgeTopic(schema)
		}

		// Step 1.5: Convert metadata tags (names) to tag IDs
		if len(processedAlert.Tags) > 0 && uc.tagService != nil {
			tags, err := uc.tagService.ConvertNamesToTags(ctx, processedAlert.Tags)
			if err != nil {
				notifier.NotifyError(ctx, &event.ErrorEvent{
					Error:   err,
					Message: "Failed to convert tag names to IDs",
				})
				return nil, goerr.Wrap(err, "failed to convert tag names")
			}
			if processedAlert.TagIDs == nil {
				processedAlert.TagIDs = make(map[string]bool)
			}
			for _, tagID := range tags {
				processedAlert.TagIDs[tagID] = true
			}
		}

		// Step 1.6: Fill metadata (generate title/description if missing)
		if uc.llmClient != nil {
			if err := processedAlert.FillMetadata(ctx, uc.llmClient); err != nil {
				notifier.NotifyError(ctx, &event.ErrorEvent{
					Error:   err,
					Message: "Failed to generate alert metadata",
				})
				return nil, goerr.Wrap(err, "failed to fill alert metadata")
			}
		}

		// Step 2: Evaluate and execute enrich policy for this alert
		enrichPolicy, err := policyService.EvaluateEnrichPolicy(ctx, processedAlert)

		if err != nil {
			notifier.NotifyError(ctx, &event.ErrorEvent{
				Error:   err,
				Message: "Enrich policy evaluation failed",
			})
			return nil, goerr.Wrap(err, "failed to evaluate enrich policy")
		}

		taskCount := enrichPolicy.TaskCount()

		// Notify enrich policy result
		notifier.NotifyEnrichPolicyResult(ctx, &event.EnrichPolicyResultEvent{
			TaskCount: taskCount,
			Policy:    enrichPolicy,
		})

		var enrichResults policy.EnrichResults
		if taskCount > 0 {
			enrichResults, err = uc.executeTasks(ctx, processedAlert, enrichPolicy, notifier)
			if err != nil {
				return nil, goerr.Wrap(err, "failed to execute enrich tasks")
			}
		}

		// Step 3: Evaluate triage policy for this alert
		triageResult, err := policyService.EvaluateTriagePolicy(ctx, processedAlert, enrichResults)
		if err != nil {
			notifier.NotifyError(ctx, &event.ErrorEvent{
				Error:   err,
				Message: "Triage policy evaluation failed",
			})
			return nil, goerr.Wrap(err, "failed to evaluate triage policy")
		}

		// Notify triage policy result
		notifier.NotifyTriagePolicyResult(ctx, &event.TriagePolicyResultEvent{
			Result: triageResult,
		})

		// Step 4: Apply triage policy result to alert
		triageResult.ApplyTo(processedAlert)

		results = append(results, &AlertPipelineResult{
			Alert:        processedAlert,
			EnrichResult: enrichResults,
			TriageResult: triageResult,
		})
	}

	return results, nil
}

// executeTasks executes all enrichment tasks and returns results
func (uc *UseCases) executeTasks(ctx context.Context, alert *alert.Alert, policyResult *policy.EnrichPolicyResult, notifier interfaces.Notifier) (policy.EnrichResults, error) {
	// Check if LLM client is configured when tasks are present
	if uc.llmClient == nil && policyResult.TaskCount() > 0 {
		return nil, goerr.New("LLM client is not configured, but enrich policy contains tasks")
	}

	var results policy.EnrichResults

	// Execute all prompts
	for _, task := range policyResult.Prompts {
		logging.From(ctx).Info("executing prompt task",
			"task_id", task.ID,
			"format", task.Format,
			"has_inline", task.Inline != "",
			"has_template", task.Template != "",
			"has_params", len(task.Params) > 0,
			"tools_count", len(uc.tools))

		// Resolve prompt text
		promptText, err := uc.resolvePrompt(ctx, &task, alert)
		if err != nil {
			notifier.NotifyError(ctx, &event.ErrorEvent{
				TaskID:  task.ID,
				Error:   err,
				Message: "Failed to resolve prompt",
			})
			return nil, goerr.Wrap(err, "failed to resolve prompt",
				goerr.V("task_id", task.ID))
		}

		logging.From(ctx).Debug("resolved prompt",
			"task_id", task.ID,
			"prompt_text", promptText)

		// Execute task (unified execution)
		result, err := uc.executePromptTask(ctx, alert, task, promptText, notifier)
		if err != nil {
			notifier.NotifyError(ctx, &event.ErrorEvent{
				TaskID:  task.ID,
				Error:   err,
				Message: "Prompt task execution failed",
			})
			logging.From(ctx).Error("prompt task failed",
				"task_id", task.ID,
				"error", err)
			return nil, goerr.Wrap(err, "failed to execute prompt task",
				goerr.V("task_id", task.ID))
		}

		logging.From(ctx).Info("prompt task completed",
			"task_id", task.ID,
			"result_type", fmt.Sprintf("%T", result))

		// Add to results array
		results = append(results, policy.EnrichResult{
			ID:     task.ID,
			Prompt: promptText,
			Result: result,
		})
	}

	return results, nil
}

// resolvePrompt resolves the prompt text from either inline or template file
func (uc *UseCases) resolvePrompt(ctx context.Context, task *policy.EnrichTask, alert *alert.Alert) (string, error) {
	if task.Inline != "" {
		// Inline prompt - use as is
		return task.Inline, nil
	}

	if task.Template != "" {
		// Template file
		if uc.promptService == nil {
			return "", goerr.New("prompt service not configured: task requires template file but --prompt-dir was not specified",
				goerr.V("task_id", task.ID),
				goerr.V("template_file", task.Template))
		}

		// Use GeneratePromptWithParams regardless of whether params exist
		// If params is nil or empty, it will just use Alert data
		return uc.promptService.GeneratePromptWithParams(ctx, task.Template, alert, task.Params)
	}

	return "", goerr.New("no prompt content specified",
		goerr.V("task_id", task.ID))
}

// executePromptTask executes a prompt task using gollem.Agent
func (uc *UseCases) executePromptTask(ctx context.Context, alert *alert.Alert, task policy.EnrichTask, promptText string, notifier interfaces.Notifier) (any, error) {
	logger := logging.From(ctx)

	// Notify prompt
	notifier.NotifyEnrichTaskPrompt(ctx, &event.EnrichTaskPromptEvent{
		TaskID:     task.ID,
		PromptText: promptText,
	})

	// Prepare system prompt with alert data
	systemPrompt, err := uc.buildAlertSystemPrompt(ctx, alert)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to build alert system prompt")
	}

	logger.Debug("built alert system prompt",
		"task_id", task.ID,
		"system_prompt_length", len(systemPrompt))

	// Create agent options
	var options []gollem.Option

	// Add system prompt with alert data
	options = append(options, gollem.WithSystemPrompt(systemPrompt))

	// Add tools if available
	if len(uc.tools) > 0 {
		// Set topic for knowledge tool if present
		for _, tool := range uc.tools {
			if kt, ok := tool.(*knowledgeTool.Knowledge); ok {
				kt.SetTopic(alert.Topic)
				defer kt.SetTopic("") // Reset after use
				logger.Debug("set topic for knowledge tool", "topic", alert.Topic)
				break
			}
		}

		options = append(options, gollem.WithToolSets(uc.tools...))
		logger.Debug("agent has tools available",
			"task_id", task.ID,
			"tools_count", len(uc.tools))
	}
	if task.Format == types.GenAIContentFormatJSON {
		options = append(options, gollem.WithContentType(gollem.ContentTypeJSON))
		logger.Debug("using JSON content type", "task_id", task.ID)
	}

	// Add sub-agents if available
	if len(uc.subAgents) > 0 {
		gollemSubAgents := make([]*gollem.SubAgent, len(uc.subAgents))
		for i, sa := range uc.subAgents {
			gollemSubAgents[i] = sa.Inner()
		}
		options = append(options, gollem.WithSubAgents(gollemSubAgents...))
		logger.Debug("agent has sub-agents available",
			"task_id", task.ID,
			"sub_agents_count", len(gollemSubAgents))
	}

	// Create agent
	agent := gollem.New(uc.llmClient, options...)

	// Combine system prompt with user prompt for agent
	combinedPrompt := systemPrompt + "\n\n" + promptText

	logger.Debug("executing agent",
		"task_id", task.ID,
		"combined_prompt_length", len(combinedPrompt))

	// Run agent with combined prompt
	result, err := agent.Execute(ctx, gollem.Text(combinedPrompt))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to run agent")
	}

	// Extract response text
	responseText := result.String()

	logger.Debug("received agent response",
		"task_id", task.ID,
		"response", responseText)

	// Parse response based on format
	var parsedResult any
	if task.Format == types.GenAIContentFormatJSON {
		var parsed any
		if err := json.Unmarshal([]byte(responseText), &parsed); err != nil {
			logger.Debug("failed to parse JSON response, using raw text",
				"task_id", task.ID,
				"error", err)
			parsedResult = responseText // Return raw text if JSON parsing fails
		} else {
			logger.Debug("successfully parsed JSON response",
				"task_id", task.ID)
			parsedResult = parsed
		}
	} else {
		parsedResult = responseText
	}

	logger.Debug("parsed task response",
		"task_id", task.ID,
		"result_type", fmt.Sprintf("%T", parsedResult))

	// Notify response
	notifier.NotifyEnrichTaskResponse(ctx, &event.EnrichTaskResponseEvent{
		TaskID:   task.ID,
		Response: parsedResult,
	})

	return parsedResult, nil
}

// buildAlertSystemPrompt creates a system prompt containing alert data and relevant domain knowledge
func (uc *UseCases) buildAlertSystemPrompt(ctx context.Context, alert *alert.Alert) (string, error) {
	// Get knowledges for the topic
	knowledges := []*knowledge.Knowledge{} // Initialize as empty slice, not nil
	if alert.Topic != "" {
		var err error
		retrieved, err := uc.repository.GetKnowledges(ctx, alert.Topic)
		if err != nil {
			logging.From(ctx).Warn("failed to get knowledges", "error", err, "topic", alert.Topic)
			// Continue with empty knowledges
		} else if retrieved != nil {
			knowledges = retrieved
		}
	}

	// Marshal alert to JSON for template
	alertJSON, err := json.MarshalIndent(alert, "", "  ")
	if err != nil {
		return "", goerr.Wrap(err, "failed to marshal alert to JSON")
	}

	// Generate prompt from template
	systemPrompt, err := prompt.GenerateWithStruct(ctx, alertSystemPromptTemplate, map[string]any{
		"alert_json": "```json\n" + string(alertJSON) + "\n```",
		"topic":      alert.Topic,
		"knowledges": knowledges,
	})
	if err != nil {
		return "", goerr.Wrap(err, "failed to generate alert system prompt")
	}

	return systemPrompt, nil
}
