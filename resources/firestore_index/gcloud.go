package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// ParseCollectionFromName extracts collection name from index name
// Example: "projects/ubie-mizutani-sandbox/databases/warren-test/collectionGroups/lists/indexes/CICAgJjFqZMK"
func ParseCollectionFromName(name string) string {
	parts := strings.Split(name, "/")
	for i, part := range parts {
		if part == "collectionGroups" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// GetExistingIndexes retrieves existing Firestore indexes using gcloud CLI
func GetExistingIndexes(projectID, databaseID string) ([]Index, error) {
	// #nosec G204
	cmd := exec.Command("gcloud", "firestore", "indexes", "composite", "list",
		"--project="+projectID,
		"--database="+databaseID,
		"--format=json")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute gcloud command: %w", err)
	}

	var indexes []Index
	if err := json.Unmarshal(output, &indexes); err != nil {
		return nil, fmt.Errorf("failed to parse indexes: %w", err)
	}

	// Set CollectionGroup from name if not set
	for i := range indexes {
		if indexes[i].CollectionGroup == "" {
			indexes[i].CollectionGroup = ParseCollectionFromName(indexes[i].Name)
		}
	}

	return indexes, nil
}

// CreateIndex creates a new Firestore index using gcloud CLI
func CreateIndex(projectID, databaseID string, config IndexConfig) error {
	fmt.Printf("Creating index for %s with fields %v...\n", config.CollectionGroup, GetFieldPaths(config.Fields))

	args := []string{
		"firestore", "indexes", "composite", "create",
		"--project=" + projectID,
		"--database=" + databaseID,
		"--collection-group=" + config.CollectionGroup,
		"--query-scope=COLLECTION",
	}

	// Add field configs
	for _, field := range config.Fields {
		var fieldConfig string
		if field.VectorConfig != nil {
			// Vector field
			vectorConfig, err := json.Marshal(field.VectorConfig)
			if err != nil {
				return fmt.Errorf("failed to marshal vector config: %w", err)
			}
			fieldConfig = fmt.Sprintf("vector-config=%s,field-path=%s", string(vectorConfig), field.FieldPath)
		} else {
			// Regular field
			fieldConfig = fmt.Sprintf("order=%s,field-path=%s", field.Order, field.FieldPath)
		}
		args = append(args, "--field-config="+fieldConfig)
	}

	// #nosec G204
	cmd := exec.Command("gcloud", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create index: %w, output: %s", err, string(output))
	}

	return nil
}

// IndexExists checks if an index with the required configuration already exists
func IndexExists(existingIndexes []Index, required IndexConfig) bool {
	for _, existing := range existingIndexes {
		if existing.CollectionGroup != required.CollectionGroup {
			continue
		}

		newFields := append(required.Fields, IndexField{
			FieldPath: "__name__",
			Order:     "*",
		})
		if FieldsMatch(existing.Fields, newFields) {
			return true
		}
	}
	return false
}

// FieldsMatch compares two sets of index fields
func FieldsMatch(existing, new []IndexField) bool {
	if len(existing) != len(new) {
		return false
	}

	// Create maps of field paths for comparison
	existingPaths := make(map[string]string)
	for _, f := range existing {
		if f.FieldPath == "__name__" {
			existingPaths[f.FieldPath] = "*"
		} else {
			existingPaths[f.FieldPath] = f.Order
		}
	}

	// Check if all new fields exist in existing index
	matchedPaths := make(map[string]bool)
	for _, f := range new {
		if existingPaths[f.FieldPath] != f.Order {
			return false
		}
		matchedPaths[f.FieldPath] = true
	}

	for _, f := range existing {
		if !matchedPaths[f.FieldPath] {
			return false
		}
	}

	return true
}

// GetFieldPaths extracts field paths from a slice of IndexFields
func GetFieldPaths(fields []IndexField) []string {
	paths := make([]string, len(fields))
	for i, f := range fields {
		paths[i] = f.FieldPath
	}
	return paths
}
