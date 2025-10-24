package usecase

import (
	"context"
	"encoding/json"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/event"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/policy"
	"github.com/secmon-lab/warren/pkg/domain/types"
	policySvc "github.com/secmon-lab/warren/pkg/service/policy"
)

// AlertPipelineResult represents the result of processing a single alert through the pipeline
type AlertPipelineResult struct {
	Alert        *alert.Alert
	EnrichResult policy.EnrichResults
	CommitResult *policy.CommitPolicyResult
}

// ProcessAlertPipeline processes an alert through the complete pipeline.
// This is a pure function without side effects (no DB save, no Slack notification).
//
// Pipeline stages:
// 1. Alert Policy Evaluation - transforms raw data into Alert objects
// 2. Tag Conversion - converts tag names to tag IDs
// 3. Metadata Generation - fills missing titles/descriptions using LLM
// 4. Enrich Policy Evaluation - executes enrichment tasks (query/agent)
// 5. Commit Policy Evaluation - applies final metadata and determines publish type
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

	// Step 1: Evaluate alert policy
	alerts, err := policyService.EvaluateAlertPolicy(ctx, schema, alertData)
	if err != nil {
		notifier.NotifyError(ctx, &event.ErrorEvent{
			Error:   err,
			Message: "Alert policy evaluation failed",
		})
		return nil, goerr.Wrap(err, "failed to evaluate alert policy")
	}

	// Notify alert policy result
	notifier.NotifyAlertPolicyResult(ctx, &event.AlertPolicyResultEvent{
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
		if len(processedAlert.Metadata.Tags) > 0 && uc.tagService != nil {
			tags, err := uc.tagService.ConvertNamesToTags(ctx, processedAlert.Metadata.Tags)
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

		// Step 3: Evaluate commit policy for this alert
		commitResult, err := policyService.EvaluateCommitPolicy(ctx, processedAlert, enrichResults)
		if err != nil {
			notifier.NotifyError(ctx, &event.ErrorEvent{
				Error:   err,
				Message: "Commit policy evaluation failed",
			})
			return nil, goerr.Wrap(err, "failed to evaluate commit policy")
		}

		// Notify commit policy result
		notifier.NotifyCommitPolicyResult(ctx, &event.CommitPolicyResultEvent{
			Result: commitResult,
		})

		// Step 4: Apply commit policy result to alert
		commitResult.ApplyTo(processedAlert)

		results = append(results, &AlertPipelineResult{
			Alert:        processedAlert,
			EnrichResult: enrichResults,
			CommitResult: commitResult,
		})
	}

	return results, nil
}

// executeTasks executes all enrichment tasks and returns results
func (uc *UseCases) executeTasks(ctx context.Context, alert *alert.Alert, policyResult *policy.EnrichPolicyResult, notifier interfaces.Notifier) (policy.EnrichResults, error) {
	results := make(policy.EnrichResults)

	// Execute query tasks
	for _, task := range policyResult.Query {
		result, err := uc.executeQueryTask(ctx, alert, task, notifier)
		if err != nil {
			notifier.NotifyError(ctx, &event.ErrorEvent{
				TaskID:  task.ID,
				Error:   err,
				Message: "Query task execution failed",
			})
			return nil, goerr.Wrap(err, "failed to execute query task",
				goerr.V("task_id", task.ID))
		}

		results[task.ID] = result
	}

	// Execute agent tasks
	for _, task := range policyResult.Agent {
		result, err := uc.executeAgentTask(ctx, alert, task, notifier)
		if err != nil {
			notifier.NotifyError(ctx, &event.ErrorEvent{
				TaskID:  task.ID,
				Error:   err,
				Message: "Agent task execution failed",
			})
			return nil, goerr.Wrap(err, "failed to execute agent task",
				goerr.V("task_id", task.ID))
		}

		results[task.ID] = result
	}

	return results, nil
}

// executeQueryTask executes a query-type task using GenerateContent
func (uc *UseCases) executeQueryTask(ctx context.Context, alert *alert.Alert, task policy.QueryTask, notifier interfaces.Notifier) (any, error) {
	promptText, err := uc.resolvePrompt(ctx, &task.EnrichTask)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to resolve prompt")
	}

	// Notify prompt
	notifier.NotifyEnrichTaskPrompt(ctx, &event.EnrichTaskPromptEvent{
		TaskID:     task.ID,
		TaskType:   "query",
		PromptText: promptText,
	})

	// Prepare system prompt with alert data
	systemPrompt, err := uc.buildAlertSystemPrompt(alert)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to build alert system prompt")
	}

	var options []gollem.SessionOption
	if task.EnrichTask.Format == types.GenAIContentFormatJSON {
		options = append(options, gollem.WithSessionContentType(gollem.ContentTypeJSON))
	}

	session, err := uc.llmClient.NewSession(ctx, options...)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create LLM session")
	}

	// Generate content with system prompt and user prompt
	response, err := session.GenerateContent(ctx,
		gollem.Text(systemPrompt),
		gollem.Text(promptText),
	)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate content")
	}

	result, err := uc.parseResponse(response, task.EnrichTask.Format)
	if err != nil {
		return nil, err
	}

	// Notify response
	notifier.NotifyEnrichTaskResponse(ctx, &event.EnrichTaskResponseEvent{
		TaskID:   task.ID,
		TaskType: "query",
		Response: result,
	})

	return result, nil
}

// executeAgentTask executes an agent-type task using gollem.Agent
func (uc *UseCases) executeAgentTask(ctx context.Context, alert *alert.Alert, task policy.AgentTask, notifier interfaces.Notifier) (any, error) {
	promptText, err := uc.resolvePrompt(ctx, &task.EnrichTask)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to resolve prompt")
	}

	// Notify prompt
	notifier.NotifyEnrichTaskPrompt(ctx, &event.EnrichTaskPromptEvent{
		TaskID:     task.ID,
		TaskType:   "agent",
		PromptText: promptText,
	})

	// Prepare system prompt with alert data
	systemPrompt, err := uc.buildAlertSystemPrompt(alert)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to build alert system prompt")
	}

	// Create agent options
	var options []gollem.Option

	// Add system prompt with alert data
	options = append(options, gollem.WithSystemPrompt(systemPrompt))

	// Add tools if available
	if len(uc.tools) > 0 {
		options = append(options, gollem.WithToolSets(uc.tools...))
	}
	if task.Format == types.GenAIContentFormatJSON {
		options = append(options, gollem.WithContentType(gollem.ContentTypeJSON))
	}

	// Create agent
	agent := gollem.New(uc.llmClient, options...)

	// Combine system prompt with user prompt for agent
	combinedPrompt := systemPrompt + "\n\n" + promptText

	// Run agent with combined prompt
	result, err := agent.Execute(ctx, gollem.Text(combinedPrompt))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to run agent")
	}

	// Extract response text
	responseText := result.String()

	// Parse response based on format
	var parsedResult any
	if task.Format == types.GenAIContentFormatJSON {
		var parsed any
		if err := json.Unmarshal([]byte(responseText), &parsed); err != nil {
			parsedResult = responseText // Return raw text if JSON parsing fails
		} else {
			parsedResult = parsed
		}
	} else {
		parsedResult = responseText
	}

	// Notify response
	notifier.NotifyEnrichTaskResponse(ctx, &event.EnrichTaskResponseEvent{
		TaskID:   task.ID,
		TaskType: "agent",
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
