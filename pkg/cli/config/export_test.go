package config

import "github.com/secmon-lab/warren/pkg/domain/model/policy"

// LoadTestFiles exports loadTestFiles for testing
func LoadTestFiles(basePath string) (*policy.TestData, error) {
	return loadTestFiles(basePath)
}
