package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/secmon-lab/warren/pkg/domain/model/policy"
)

// LoadTestFiles exports loadTestFiles for testing
func LoadTestFiles(basePath string) (*policy.TestData, error) {
	return loadTestFiles(basePath)
}

// loadTestFiles loads test data from JSON/JSONL files in the specified directory
func loadTestFiles(basePath string) (*policy.TestData, error) {
	data := &policy.TestData{
		Data: make(map[string]map[string]any),
	}

	// Walk through the directory structure
	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Skip non-JSON files
		if !strings.HasSuffix(path, ".json") && !strings.HasSuffix(path, ".jsonl") {
			return nil
		}

		// Get relative path from basePath
		relPath, err := filepath.Rel(basePath, path)
		if err != nil {
			return err
		}

		// Determine schema (top-level directory)
		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) == 0 {
			return nil
		}
		schema := parts[0]

		// Initialize schema map if needed
		if data.Data[schema] == nil {
			data.Data[schema] = make(map[string]any)
		}

		// Get path relative to schema directory (everything after schema/)
		schemaRelPath := strings.Join(parts[1:], "/")

		// Read and parse file
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if strings.HasSuffix(path, ".jsonl") {
			// Handle JSONL files with multiple objects
			lines := strings.Split(strings.TrimSpace(string(content)), "\n")
			for i, line := range lines {
				if strings.TrimSpace(line) == "" {
					continue
				}
				var obj any
				if err := json.Unmarshal([]byte(line), &obj); err != nil {
					return fmt.Errorf("failed to parse JSON line %d in %s: %w", i+1, path, err)
				}
				key := schemaRelPath
				if i > 0 {
					// Add suffix for subsequent objects
					key = strings.TrimSuffix(schemaRelPath, ".jsonl") + fmt.Sprintf("_obj%d.jsonl", i+1)
				}
				data.Data[schema][key] = obj
			}
		} else {
			// Handle regular JSON files
			var obj any
			if err := json.Unmarshal(content, &obj); err != nil {
				return fmt.Errorf("failed to parse JSON in %s: %w", path, err)
			}
			data.Data[schema][schemaRelPath] = obj
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Check if any data was loaded
	if len(data.Data) == 0 {
		return nil, fmt.Errorf("no test data found in %s", basePath)
	}

	return data, nil
}
