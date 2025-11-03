package bigquery

import (
	"context"
	_ "embed"
	"strings"

	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

//go:embed prompt/base.md
var basePrompt string

// buildSystemPromptWithMemories builds system prompt with KPT-formatted memories
func (a *Agent) buildSystemPromptWithMemories(ctx context.Context, memories []*memory.AgentMemory) (string, error) {
	log := logging.From(ctx)
	log.Debug("Building system prompt with memories", "memory_count", len(memories), "table_count", len(a.config.Tables))

	var builder strings.Builder

	// Base prompt with guidelines
	log.Debug("Adding base prompt", "base_prompt_length", len(basePrompt))
	builder.WriteString(basePrompt)
	builder.WriteString("\n\n")

	// Available tables
	if len(a.config.Tables) > 0 {
		log.Debug("Adding available tables section", "table_count", len(a.config.Tables))
		builder.WriteString("## Available BigQuery Tables\n\n")
		builder.WriteString("You have access to the following BigQuery tables:\n\n")
		for i, table := range a.config.Tables {
			builder.WriteString("- `")
			builder.WriteString(table.ProjectID)
			builder.WriteString(".")
			builder.WriteString(table.DatasetID)
			builder.WriteString(".")
			builder.WriteString(table.TableID)
			builder.WriteString("`")
			if table.Description != "" {
				builder.WriteString(": ")
				builder.WriteString(table.Description)
			}
			builder.WriteString("\n")
			log.Debug("Added table to prompt",
				"index", i,
				"table", table.ProjectID+"."+table.DatasetID+"."+table.TableID)
		}
		builder.WriteString("\n")
	} else {
		log.Debug("No tables to add to prompt")
	}

	// Add KPT-formatted memories
	if len(memories) > 0 {
		log.Debug("Adding past execution experiences", "experience_count", len(memories))
		builder.WriteString("# Past Execution Experiences\n\n")
		builder.WriteString("You have access to past execution experiences in KPT (Keep/Problem/Try) format:\n\n")

		for i, mem := range memories {
			log.Debug("Adding experience to prompt",
				"index", i,
				"memory_id", mem.ID,
				"task_query", mem.TaskQuery,
				"has_success", mem.SuccessDescription != "",
				"problem_count", len(mem.Problems),
				"improvement_count", len(mem.Improvements))

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
	} else {
		log.Debug("No memories to add to prompt")
	}

	prompt := builder.String()
	log.Debug("System prompt built successfully", "total_length", len(prompt))

	return prompt, nil
}
