package bluebell_test

import (
	"strings"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/usecase/chat/bluebell"
)

func TestGenerateSystemPrompt_WithTicket(t *testing.T) {
	ctx := setupTestContext(t)

	result, err := bluebell.ExportGenerateSystemPrompt(ctx, bluebell.SystemPromptData{
		Context: bluebell.ContextData{
			Ticket: `{"id": "T-123", "status": "open"}`,
			Alert: bluebell.AlertData{
				Data:  `{"schema": "test.alert", "data": {"ip": "1.2.3.4"}}`,
				Count: 3,
			},
		},
		Tools: bluebell.ToolsData{
			Description: "- `bigquery` — Query BigQuery",
		},
		ResolvedIntent: "Investigate potential lateral movement from 1.2.3.4",
		Lang:           "English",
		Requester: bluebell.Requester{
			ID: "U12345",
		},
	})
	gt.NoError(t, err)
	gt.True(t, strings.Contains(result, "T-123"))
	gt.True(t, strings.Contains(result, "1 of 3"))
	gt.True(t, strings.Contains(result, "Investigation Directive"))
	gt.True(t, strings.Contains(result, "lateral movement"))
	gt.True(t, strings.Contains(result, "bigquery"))
	gt.True(t, strings.Contains(result, "U12345"))
}

func TestGenerateSystemPrompt_Ticketless(t *testing.T) {
	ctx := setupTestContext(t)

	result, err := bluebell.ExportGenerateSystemPrompt(ctx, bluebell.SystemPromptData{
		Context: bluebell.ContextData{
			// No Ticket — ticketless mode
		},
		Tools: bluebell.ToolsData{
			Description: "- `virustotal` — Check indicators",
		},
		Lang: "Japanese",
	})
	gt.NoError(t, err)
	gt.True(t, !strings.Contains(result, "Ticket Information"))
	gt.True(t, strings.Contains(result, "virustotal"))
	gt.True(t, strings.Contains(result, "Japanese"))
}

func TestGenerateSystemPrompt_EmptyResolvedIntent(t *testing.T) {
	ctx := setupTestContext(t)

	result, err := bluebell.ExportGenerateSystemPrompt(ctx, bluebell.SystemPromptData{
		Context: bluebell.ContextData{
			Ticket: `{"id": "T-456"}`,
			Alert:  bluebell.AlertData{Data: `{}`, Count: 1},
		},
		Tools: bluebell.ToolsData{
			Description: "(no tools available)",
		},
		ResolvedIntent: "", // empty
		Lang:           "English",
	})
	gt.NoError(t, err)
	gt.True(t, !strings.Contains(result, "Investigation Directive"))
}

func TestGenerateSystemPrompt_WithResolvedIntent(t *testing.T) {
	ctx := setupTestContext(t)

	result, err := bluebell.ExportGenerateSystemPrompt(ctx, bluebell.SystemPromptData{
		Context: bluebell.ContextData{
			Ticket: `{"id": "T-789"}`,
			Alert:  bluebell.AlertData{Data: `{}`, Count: 1},
		},
		Tools: bluebell.ToolsData{
			Description: "(no tools available)",
		},
		ResolvedIntent: "This is an infrastructure issue, not a security threat.",
		Lang:           "English",
	})
	gt.NoError(t, err)
	gt.True(t, strings.Contains(result, "Investigation Directive"))
	gt.True(t, strings.Contains(result, "infrastructure issue"))
}

func TestGenerateSystemPrompt_AllFieldsPopulated(t *testing.T) {
	ctx := setupTestContext(t)

	result, err := bluebell.ExportGenerateSystemPrompt(ctx, bluebell.SystemPromptData{
		Context: bluebell.ContextData{
			Ticket: `{"id": "T-FULL"}`,
			Alert:  bluebell.AlertData{Data: `{"full": true}`, Count: 5},
		},
		Tools: bluebell.ToolsData{
			Description: "- `tool1` — Tool 1\n- `tool2` — Tool 2",
		},
		ResolvedIntent: "Full investigation directive.",
		Lang:           "English",
		Requester:      bluebell.Requester{ID: "UFULL"},
	})
	gt.NoError(t, err)
	gt.True(t, strings.Contains(result, "T-FULL"))
	gt.True(t, strings.Contains(result, "1 of 5"))
	gt.True(t, strings.Contains(result, "tool1"))
	gt.True(t, strings.Contains(result, "tool2"))
	gt.True(t, strings.Contains(result, "Investigation Directive"))
	gt.True(t, strings.Contains(result, "Full investigation directive."))
	gt.True(t, strings.Contains(result, "UFULL"))
	gt.True(t, strings.Contains(result, "English"))
}
