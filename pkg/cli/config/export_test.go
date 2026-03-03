package config

import "github.com/secmon-lab/warren/pkg/domain/model/policy"

// LoadTestFiles exports loadTestFiles for testing
func LoadTestFiles(basePath string) (*policy.TestData, error) {
	return loadTestFiles(basePath)
}

// NewUserSystemPromptWithPath creates a UserSystemPrompt with a given file path for testing.
func NewUserSystemPromptWithPath(path string) *UserSystemPrompt {
	return &UserSystemPrompt{filePath: path}
}
