package bigquery

import (
	"github.com/m-mizutani/gollem"
)

// NewToolSetForTest creates a toolSet instance for testing with direct configuration
func NewToolSetForTest(config *Config, projectID string, impersonateServiceAccount string) *toolSet {
	return &toolSet{
		config: config,
		tool: &internalTool{
			config:                    config,
			projectID:                 projectID,
			impersonateServiceAccount: impersonateServiceAccount,
		},
	}
}

// ExportNewInternalTool creates a new internalTool instance for testing
func ExportNewInternalTool(config *Config, projectID string, impersonateServiceAccount string) gollem.ToolSet {
	return &internalTool{
		config:                    config,
		projectID:                 projectID,
		impersonateServiceAccount: impersonateServiceAccount,
	}
}

// ToolSpec is exported for testing
type ToolSpec = gollem.ToolSpec

// ExportedBuildSystemPrompt is exported for testing
func ExportedBuildSystemPrompt(config *Config) (string, error) {
	return buildSystemPrompt(config)
}

// ExportedBuildPromptHint is exported for testing
func ExportedBuildPromptHint(config *Config) (string, error) {
	return buildPromptHint(config)
}
