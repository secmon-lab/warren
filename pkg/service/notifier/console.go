package notifier

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/secmon-lab/warren/pkg/domain/event"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
)

// ConsoleNotifier is a console-based event notifier that outputs
// alert pipeline events to the console with color formatting.
// Useful for CLI mode and debugging.
type ConsoleNotifier struct{}

// NewConsoleNotifier creates a new console notifier
func NewConsoleNotifier() interfaces.Notifier {
	return &ConsoleNotifier{}
}

func (n *ConsoleNotifier) NotifyAlertPolicyResult(ctx context.Context, ev *event.AlertPolicyResultEvent) {
	printAlertPolicyResult(ev)
}

func (n *ConsoleNotifier) NotifyEnrichPolicyResult(ctx context.Context, ev *event.EnrichPolicyResultEvent) {
	printEnrichPolicyResult(ev)
}

func (n *ConsoleNotifier) NotifyCommitPolicyResult(ctx context.Context, ev *event.CommitPolicyResultEvent) {
	printCommitPolicyResult(ev)
}

func (n *ConsoleNotifier) NotifyEnrichTaskPrompt(ctx context.Context, ev *event.EnrichTaskPromptEvent) {
	printEnrichTaskPrompt(ev)
}

func (n *ConsoleNotifier) NotifyEnrichTaskResponse(ctx context.Context, ev *event.EnrichTaskResponseEvent) {
	printEnrichTaskResponse(ev)
}

func (n *ConsoleNotifier) NotifyError(ctx context.Context, ev *event.ErrorEvent) {
	printError(ev)
}

func printAlertPolicyResult(e *event.AlertPolicyResultEvent) {
	blue := color.New(color.FgBlue, color.Bold)
	white := color.New(color.FgWhite)

	blue.Println("Alert Policy Result:")
	fmt.Printf("  Schema: %s\n", e.Schema)
	fmt.Printf("  Alerts: %d\n\n", len(e.Alerts))

	// Print full alert details
	for i, alert := range e.Alerts {
		blue.Printf("Alert #%d:\n", i+1)
		alertJSON, err := json.MarshalIndent(alert, "  ", "  ")
		if err != nil {
			white.Printf("  %v\n", alert)
		} else {
			white.Printf("  %s\n", string(alertJSON))
		}
		fmt.Println()
	}
}

func printEnrichPolicyResult(e *event.EnrichPolicyResultEvent) {
	blue := color.New(color.FgBlue, color.Bold)
	yellow := color.New(color.FgYellow)

	blue.Print("Enrich Policy Result: ")
	fmt.Printf("Tasks=%d\n", e.TaskCount)

	if e.Policy != nil {
		if len(e.Policy.Query) > 0 {
			queryIDs := make([]string, 0, len(e.Policy.Query))
			for _, task := range e.Policy.Query {
				queryIDs = append(queryIDs, task.ID)
			}
			yellow.Printf("  Query tasks: ")
			fmt.Printf("%s\n", strings.Join(queryIDs, ", "))
		}
		if len(e.Policy.Agent) > 0 {
			agentIDs := make([]string, 0, len(e.Policy.Agent))
			for _, task := range e.Policy.Agent {
				agentIDs = append(agentIDs, task.ID)
			}
			yellow.Printf("  Agent tasks: ")
			fmt.Printf("%s\n", strings.Join(agentIDs, ", "))
		}
	}
}

func printCommitPolicyResult(e *event.CommitPolicyResultEvent) {
	blue := color.New(color.FgBlue, color.Bold)
	green := color.New(color.FgGreen)

	blue.Print("Commit Policy Result: ")
	fmt.Printf("Publish=%s\n", e.Result.Publish)

	if e.Result.Title != "" {
		green.Printf("  Title: %s\n", e.Result.Title)
	}
	if e.Result.Description != "" {
		green.Printf("  Description: %s\n", e.Result.Description)
	}
	if e.Result.Channel != "" {
		green.Printf("  Channel: %s\n", e.Result.Channel)
	}
	if len(e.Result.Attr) > 0 {
		green.Printf("  Attributes: %d\n", len(e.Result.Attr))
	}
}

func printEnrichTaskPrompt(e *event.EnrichTaskPromptEvent) {
	cyan := color.New(color.FgCyan, color.Bold)
	gray := color.New(color.FgHiBlack)

	cyan.Printf("Task Prompt [%s] (%s): ", e.TaskID, e.TaskType)
	fmt.Printf("%d chars\n", len(e.PromptText))
	gray.Printf("  %s\n", e.PromptText)
}

func printEnrichTaskResponse(e *event.EnrichTaskResponseEvent) {
	green := color.New(color.FgGreen, color.Bold)
	white := color.New(color.FgWhite)

	green.Printf("Task Response [%s] (%s):\n", e.TaskID, e.TaskType)

	// Format response based on type
	switch v := e.Response.(type) {
	case string:
		white.Printf("  %s\n", v)

	case map[string]any, []any:
		jsonBytes, err := json.MarshalIndent(v, "  ", "  ")
		if err != nil {
			white.Printf("  %v\n", v)
		} else {
			white.Printf("  %s\n", string(jsonBytes))
		}

	default:
		white.Printf("  %v\n", v)
	}
}

func printError(e *event.ErrorEvent) {
	red := color.New(color.FgRed, color.Bold)

	if e.TaskID != "" {
		red.Printf("Error [%s]: %s\n", e.TaskID, e.Message)
	} else {
		red.Printf("Error: %s\n", e.Message)
	}

	if e.Error != nil {
		gray := color.New(color.FgHiBlack)
		gray.Printf("  %v\n", e.Error)
	}
}
