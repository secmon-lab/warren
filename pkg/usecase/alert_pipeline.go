package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/event"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/policy"
	"github.com/secmon-lab/warren/pkg/domain/types"
	policySvc "github.com/secmon-lab/warren/pkg/service/policy"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

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

		enrichResults := make(policy.EnrichResults)
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

	results := make(policy.EnrichResults)

	// Execute query tasks
	for _, task := range policyResult.Query {
		logging.From(ctx).Info("executing query task",
			"task_id", task.ID,
			"format", task.Format,
			"has_inline", task.Inline != "",
			"has_prompt", task.Prompt != "")

		result, err := uc.executeQueryTask(ctx, alert, task, notifier)
		if err != nil {
			notifier.NotifyError(ctx, &event.ErrorEvent{
				TaskID:  task.ID,
				Error:   err,
				Message: "Query task execution failed",
			})
			logging.From(ctx).Error("query task failed",
				"task_id", task.ID,
				"error", err)
			return nil, goerr.Wrap(err, "failed to execute query task",
				goerr.V("task_id", task.ID))
		}

		logging.From(ctx).Info("query task completed",
			"task_id", task.ID,
			"result_type", fmt.Sprintf("%T", result))
		results[task.ID] = result
	}

	// Execute agent tasks
	for _, task := range policyResult.Agent {
		logging.From(ctx).Info("executing agent task",
			"task_id", task.ID,
			"format", task.Format,
			"has_inline", task.Inline != "",
			"has_prompt", task.Prompt != "",
			"tools_count", len(uc.tools))

		result, err := uc.executeAgentTask(ctx, alert, task, notifier)
		if err != nil {
			notifier.NotifyError(ctx, &event.ErrorEvent{
				TaskID:  task.ID,
				Error:   err,
				Message: "Agent task execution failed",
			})
			logging.From(ctx).Error("agent task failed",
				"task_id", task.ID,
				"error", err)
			return nil, goerr.Wrap(err, "failed to execute agent task",
				goerr.V("task_id", task.ID))
		}

		logging.From(ctx).Info("agent task completed",
			"task_id", task.ID,
			"result_type", fmt.Sprintf("%T", result))
		results[task.ID] = result
	}

	return results, nil
}

// executeQueryTask executes a query-type task using GenerateContent
func (uc *UseCases) executeQueryTask(ctx context.Context, alert *alert.Alert, task policy.QueryTask, notifier interfaces.Notifier) (any, error) {
	logger := logging.From(ctx)

	promptText, err := uc.resolvePrompt(ctx, &task.EnrichTask)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to resolve prompt")
	}

	logger.Debug("resolved query task prompt",
		"task_id", task.ID,
		"prompt_length", len(promptText))

	// Notify prompt
	notifier.NotifyEnrichTaskPrompt(ctx, &event.EnrichTaskPromptEvent{
		TaskID:     task.ID,
		TaskType:   policy.TaskTypeQuery,
		PromptText: promptText,
	})

	// Prepare system prompt with alert data
	systemPrompt, err := uc.buildAlertSystemPrompt(alert)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to build alert system prompt")
	}

	logger.Debug("built alert system prompt",
		"task_id", task.ID,
		"system_prompt_length", len(systemPrompt))

	var options []gollem.SessionOption
	if task.Format == types.GenAIContentFormatJSON {
		options = append(options, gollem.WithSessionContentType(gollem.ContentTypeJSON))
		logger.Debug("using JSON content type", "task_id", task.ID)
	}

	session, err := uc.llmClient.NewSession(ctx, options...)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create LLM session")
	}

	logger.Debug("calling LLM GenerateContent", "task_id", task.ID)

	// Generate content with system prompt and user prompt
	response, err := session.GenerateContent(ctx,
		gollem.Text(systemPrompt),
		gollem.Text(promptText),
	)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate content")
	}

	logger.Debug("received LLM response",
		"task_id", task.ID,
		"response_texts", response.Texts)

	result, err := uc.parseResponse(response, task.Format)
	if err != nil {
		return nil, err
	}

	logger.Debug("parsed query task response",
		"task_id", task.ID,
		"result_type", fmt.Sprintf("%T", result))

	// Notify response
	notifier.NotifyEnrichTaskResponse(ctx, &event.EnrichTaskResponseEvent{
		TaskID:   task.ID,
		TaskType: policy.TaskTypeQuery,
		Response: result,
	})

	return result, nil
}

// executeAgentTask executes an agent-type task using gollem.Agent
func (uc *UseCases) executeAgentTask(ctx context.Context, alert *alert.Alert, task policy.AgentTask, notifier interfaces.Notifier) (any, error) {
	logger := logging.From(ctx)

	promptText, err := uc.resolvePrompt(ctx, &task.EnrichTask)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to resolve prompt")
	}

	logger.Debug("resolved agent task prompt",
		"task_id", task.ID,
		"prompt_length", len(promptText))

	// Notify prompt
	notifier.NotifyEnrichTaskPrompt(ctx, &event.EnrichTaskPromptEvent{
		TaskID:     task.ID,
		TaskType:   policy.TaskTypeAgent,
		PromptText: promptText,
	})

	// Prepare system prompt with alert data
	systemPrompt, err := uc.buildAlertSystemPrompt(alert)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to build alert system prompt")
	}

	logger.Debug("built alert system prompt for agent",
		"task_id", task.ID,
		"system_prompt_length", len(systemPrompt))

	// Create agent options
	var options []gollem.Option

	// Add system prompt with alert data
	options = append(options, gollem.WithSystemPrompt(systemPrompt))

	// Add tools if available
	if len(uc.tools) > 0 {
		options = append(options, gollem.WithToolSets(uc.tools...))
		logger.Debug("agent has tools available",
			"task_id", task.ID,
			"tools_count", len(uc.tools))
	}
	if task.Format == types.GenAIContentFormatJSON {
		options = append(options, gollem.WithContentType(gollem.ContentTypeJSON))
		logger.Debug("using JSON content type for agent", "task_id", task.ID)
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
		"response_length", len(responseText))

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

	logger.Debug("parsed agent task response",
		"task_id", task.ID,
		"result_type", fmt.Sprintf("%T", parsedResult))

	// Notify response
	notifier.NotifyEnrichTaskResponse(ctx, &event.EnrichTaskResponseEvent{
		TaskID:   task.ID,
		TaskType: policy.TaskTypeAgent,
		Response: parsedResult,
	})

	return parsedResult, nil
}

// resolvePrompt resolves the prompt text from either inline or file
// Note: Alert data is NOT included in the prompt - it's passed via system prompt
func (uc *UseCases) resolvePrompt(ctx context.Context, task *policy.EnrichTask) (string, error) {
	if task.Inline != "" {
		// Inline prompt - use as is
		return task.Inline, nil
	}

	if task.Prompt != "" {
		// File path - read the prompt file directly without template rendering
		if uc.promptService == nil {
			return "", goerr.New("prompt service not configured: task requires prompt file but --prompt-dir was not specified",
				goerr.V("task_id", task.ID),
				goerr.V("prompt_file", task.Prompt))
		}
		// Read prompt file without alert data (alert is in system prompt)
		return uc.promptService.ReadPromptFile(ctx, task.Prompt)
	}

	return "", goerr.New("no prompt content specified",
		goerr.V("task_id", task.ID))
}

// buildAlertSystemPrompt creates a system prompt containing alert data
func (uc *UseCases) buildAlertSystemPrompt(alert *alert.Alert) (string, error) {
	alertJSON, err := json.MarshalIndent(alert, "", "  ")
	if err != nil {
		return "", goerr.Wrap(err, "failed to marshal alert to JSON")
	}

	systemPrompt := "You are analyzing a security alert. Here is the alert information:\n\n"
	systemPrompt += "```json\n"
	systemPrompt += string(alertJSON)
	systemPrompt += "\n```\n\n"
	systemPrompt += "Use this alert information to respond to the user's request. Do not include the alert data in your response unless specifically asked."

	return systemPrompt, nil
}

// parseResponse parses the LLM response based on format
func (uc *UseCases) parseResponse(response *gollem.Response, format types.GenAIContentFormat) (any, error) {
	var responseText string
	if len(response.Texts) > 0 {
		responseText = response.Texts[0]
	}

	if format == types.GenAIContentFormatJSON {
		var parsed any
		if err := json.Unmarshal([]byte(responseText), &parsed); err != nil {
			return responseText, nil // Return raw text if JSON parsing fails
		}
		return parsed, nil
	}

	return responseText, nil
}
