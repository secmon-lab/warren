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

func (n *ConsoleNotifier) NotifyIngestPolicyResult(ctx context.Context, ev *event.IngestPolicyResultEvent) {
	printIngestPolicyResult(ev)
}

func (n *ConsoleNotifier) NotifyEnrichPolicyResult(ctx context.Context, ev *event.EnrichPolicyResultEvent) {
	printEnrichPolicyResult(ev)
}

func (n *ConsoleNotifier) NotifyTriagePolicyResult(ctx context.Context, ev *event.TriagePolicyResultEvent) {
	printTriagePolicyResult(ev)
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

func printIngestPolicyResult(e *event.IngestPolicyResultEvent) {
	blue := color.New(color.FgBlue, color.Bold)
	white := color.New(color.FgWhite)

	_, _ = blue.Println("Ingest Policy Result:")
	fmt.Printf("  Schema: %s\n", e.Schema)
	fmt.Printf("  Alerts: %d\n\n", len(e.Alerts))

	// Print full alert details
	for i, alert := range e.Alerts {
		_, _ = blue.Printf("Alert #%d:\n", i+1)
		alertJSON, err := json.MarshalIndent(alert, "  ", "  ")
		if err != nil {
			_, _ = white.Printf("  %v\n", alert)
		} else {
			_, _ = white.Printf("  %s\n", string(alertJSON))
		}
		fmt.Println()
	}
}

func printEnrichPolicyResult(e *event.EnrichPolicyResultEvent) {
	blue := color.New(color.FgBlue, color.Bold)
	yellow := color.New(color.FgYellow)

	_, _ = blue.Print("Enrich Policy Result: ")
	fmt.Printf("Tasks=%d\n", e.TaskCount)

	if e.Policy != nil {
		if len(e.Policy.Prompts) > 0 {
			promptIDs := make([]string, 0, len(e.Policy.Prompts))
			for _, task := range e.Policy.Prompts {
				promptIDs = append(promptIDs, task.ID)
			}
			_, _ = yellow.Printf("  Prompt tasks: ")
			fmt.Printf("%s\n", strings.Join(promptIDs, ", "))
		}
	}
}

func printTriagePolicyResult(e *event.TriagePolicyResultEvent) {
	blue := color.New(color.FgBlue, color.Bold)
	green := color.New(color.FgGreen)

	_, _ = blue.Print("Triage Policy Result: ")
	fmt.Printf("Publish=%s\n", e.Result.Publish)

	if e.Result.Title != "" {
		_, _ = green.Printf("  Title: %s\n", e.Result.Title)
	}
	if e.Result.Description != "" {
		_, _ = green.Printf("  Description: %s\n", e.Result.Description)
	}
	if e.Result.Channel != "" {
		_, _ = green.Printf("  Channel: %s\n", e.Result.Channel)
	}
	if len(e.Result.Attr) > 0 {
		_, _ = green.Printf("  Attributes: %d\n", len(e.Result.Attr))
	}
}

func printEnrichTaskPrompt(e *event.EnrichTaskPromptEvent) {
	cyan := color.New(color.FgCyan, color.Bold)
	gray := color.New(color.FgHiBlack)

	_, _ = cyan.Printf("Task Prompt [%s]: ", e.TaskID)
	fmt.Printf("%d chars\n", len(e.PromptText))
	_, _ = gray.Printf("  %s\n", e.PromptText)
}

func printEnrichTaskResponse(e *event.EnrichTaskResponseEvent) {
	green := color.New(color.FgGreen, color.Bold)
	white := color.New(color.FgWhite)

	_, _ = green.Printf("Task Response [%s]:\n", e.TaskID)

	// Format response based on type
	switch v := e.Response.(type) {
	case string:
		_, _ = white.Printf("  %s\n", v)

	case map[string]any, []any:
		jsonBytes, err := json.MarshalIndent(v, "  ", "  ")
		if err != nil {
			_, _ = white.Printf("  %v\n", v)
		} else {
			_, _ = white.Printf("  %s\n", string(jsonBytes))
		}

	default:
		_, _ = white.Printf("  %v\n", v)
	}
}

func printError(e *event.ErrorEvent) {
	red := color.New(color.FgRed, color.Bold)

	if e.TaskID != "" {
		_, _ = red.Printf("Error [%s]: %s\n", e.TaskID, e.Message)
	} else {
		_, _ = red.Printf("Error: %s\n", e.Message)
	}

	if e.Error != nil {
		gray := color.New(color.FgHiBlack)
		_, _ = gray.Printf("  %v\n", e.Error)
	}
}
