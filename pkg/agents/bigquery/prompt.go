package bigquery

import (
	"context"
	"strings"

	"github.com/secmon-lab/warren/pkg/domain/model/memory"
)

// buildSystemPromptWithMemories builds system prompt with KPT-formatted memories
func (a *Agent) buildSystemPromptWithMemories(ctx context.Context, memories []*memory.AgentMemory) (string, error) {
	var builder strings.Builder

	// Base prompt - include table information
	if len(a.config.Tables) > 0 {
		builder.WriteString("# Available BigQuery Tables\n\n")
		for _, table := range a.config.Tables {
			builder.WriteString("- ")
			builder.WriteString(table.ProjectID)
			builder.WriteString(".")
			builder.WriteString(table.DatasetID)
			builder.WriteString(".")
			builder.WriteString(table.TableID)
			if table.Description != "" {
				builder.WriteString(": ")
				builder.WriteString(table.Description)
			}
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}

	// Add KPT-formatted memories
	if len(memories) > 0 {
		builder.WriteString("# Past Execution Experiences\n\n")
		builder.WriteString("You have access to past execution experiences in KPT (Keep/Problem/Try) format:\n\n")

		for i, mem := range memories {
			builder.WriteString("## Experience ")
			builder.WriteString(string(rune('A' + i)))
			builder.WriteString("\n\n")

			builder.WriteString("**Query**: ")
			builder.WriteString(mem.TaskQuery)
			builder.WriteString("\n\n")

			// K: Keep - Success patterns
			if mem.SuccessDescription != "" {
				builder.WriteString("**Keep (What worked well)**:\n")
				builder.WriteString(mem.SuccessDescription)
				builder.WriteString("\n\n")
			}

			// P: Problem - Issues encountered
			if len(mem.Problems) > 0 {
				builder.WriteString("**Problem (Issues encountered)**:\n")
				for _, problem := range mem.Problems {
					builder.WriteString("- ")
					builder.WriteString(problem)
					builder.WriteString("\n")
				}
				builder.WriteString("\n")
			}

			// T: Try - Improvements to try
			if len(mem.Improvements) > 0 {
				builder.WriteString("**Try (Improvements to apply)**:\n")
				for _, improvement := range mem.Improvements {
					builder.WriteString("- ")
					builder.WriteString(improvement)
					builder.WriteString("\n")
				}
				builder.WriteString("\n")
			}

			builder.WriteString("---\n\n")
		}

		builder.WriteString("Use these experiences to inform your approach to the current task.\n\n")
	}

	return builder.String(), nil
}
