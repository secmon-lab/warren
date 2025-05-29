package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type indexField struct {
	FieldPath    string                 `json:"fieldPath"`
	Order        string                 `json:"order,omitempty"`
	VectorConfig map[string]interface{} `json:"vectorConfig,omitempty"`
}

type index struct {
	Name            string       `json:"name"`
	CollectionGroup string       `json:"collectionGroup"`
	Fields          []indexField `json:"fields"`
	QueryScope      string       `json:"queryScope"`
}

type indexConfig struct {
	collectionGroup string
	fields          []indexField
}

// parseCollectionFromName extracts collection name from index name
// Example: "projects/ubie-mizutani-sandbox/databases/warren-test/collectionGroups/lists/indexes/CICAgJjFqZMK"
func parseCollectionFromName(name string) string {
	parts := strings.Split(name, "/")
	for i, part := range parts {
		if part == "collectionGroups" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func main() {
	projectID := flag.String("project", "", "GCP project ID")
	databaseID := flag.String("database", "", "Firestore database ID")
	dryrun := flag.Bool("dryrun", false, "Check required indexes without creating them")
	flag.Parse()

	if *projectID == "" || *databaseID == "" {
		fmt.Println("Usage: create_index -project=<project_id> -database=<database_id> [-dryrun]")
		os.Exit(1)
	}

	// Define all required indexes
	requiredIndexes := defineRequiredIndexes()

	// Get existing indexes
	existingIndexes, err := getExistingIndexes(*projectID, *databaseID)
	if err != nil {
		log.Fatalf("Failed to get existing indexes: %v", err)
	}

	if *dryrun {
		checkRequiredIndexes(existingIndexes, requiredIndexes)
		return
	}

	// Check and create missing indexes in parallel
	if err := createMissingIndexes(*projectID, *databaseID, existingIndexes, requiredIndexes); err != nil {
		os.Exit(1)
	}
}

func defineRequiredIndexes() []indexConfig {
	collections := []string{"alerts", "tickets", "lists"}
	var requiredIndexes []indexConfig

	for _, collection := range collections {
		// Single-field Embedding index
		requiredIndexes = append(requiredIndexes, indexConfig{
			collectionGroup: collection,
			fields: []indexField{
				{
					FieldPath:    "Embedding",
					VectorConfig: map[string]interface{}{"dimension": 256, "flat": map[string]interface{}{}},
				},
			},
		})

		// Embedding + CreatedAt index
		requiredIndexes = append(requiredIndexes, indexConfig{
			collectionGroup: collection,
			fields: []indexField{
				{
					FieldPath: "CreatedAt",
					Order:     "DESCENDING",
				},
				{
					FieldPath:    "Embedding",
					VectorConfig: map[string]interface{}{"dimension": 256, "flat": map[string]interface{}{}},
				},
			},
		})

		// Status + CreatedAt index only for 'tickets'
		if collection == "tickets" {
			requiredIndexes = append(requiredIndexes, indexConfig{
				collectionGroup: collection,
				fields: []indexField{
					{
						FieldPath: "Status",
						Order:     "ASCENDING",
					},
					{
						FieldPath: "CreatedAt",
						Order:     "DESCENDING",
					},
				},
			})
		}
	}

	return requiredIndexes
}

func checkRequiredIndexes(existingIndexes []index, requiredIndexes []indexConfig) {
	fmt.Println("Checking required indexes...")
	fmt.Println("----------------------------------------")

	for _, required := range requiredIndexes {
		if indexExists(existingIndexes, required) {
			fmt.Printf("✅ Index for %s with fields %v exists\n",
				required.collectionGroup, getFieldPaths(required.fields))
		} else {
			fmt.Printf("❌ Index for %s with fields %v is missing\n",
				required.collectionGroup, getFieldPaths(required.fields))
			fmt.Printf("  Configuration:\n")
			for _, field := range required.fields {
				if field.VectorConfig != nil {
					vectorConfig, _ := json.Marshal(field.VectorConfig)
					fmt.Printf("    - %s (vector config: %s)\n", field.FieldPath, string(vectorConfig))
				} else {
					fmt.Printf("    - %s (order: %s)\n", field.FieldPath, field.Order)
				}
			}
		}
	}

	fmt.Println("----------------------------------------")
	fmt.Println("Note: Run without -dryrun to create missing indexes")
}

func createMissingIndexes(projectID, databaseID string, existingIndexes []index, requiredIndexes []indexConfig) error {
	var wg sync.WaitGroup
	var mu sync.Mutex // for synchronized output
	errors := make(chan error, len(requiredIndexes))

	// First, show what will be created
	fmt.Println("The following indexes will be created:")
	fmt.Println("----------------------------------------")
	for _, required := range requiredIndexes {
		if !indexExists(existingIndexes, required) {
			fmt.Printf("Creating index for %s:\n", required.collectionGroup)
			for _, field := range required.fields {
				if field.VectorConfig != nil {
					vectorConfig, _ := json.Marshal(field.VectorConfig)
					fmt.Printf("  - %s (vector config: %s)\n", field.FieldPath, string(vectorConfig))
				} else {
					fmt.Printf("  - %s (order: %s)\n", field.FieldPath, field.Order)
				}
			}
		}
	}
	fmt.Println("----------------------------------------")

	// Then create the indexes
	for _, required := range requiredIndexes {
		if !indexExists(existingIndexes, required) {
			wg.Add(1)
			go func(config indexConfig) {
				defer wg.Done()
				if err := createIndex(projectID, databaseID, config); err != nil {
					mu.Lock()
					log.Printf("Failed to create index for %s: %v", config.collectionGroup, err)
					mu.Unlock()
					errors <- err
				}
			}(required)
		} else {
			mu.Lock()
			fmt.Printf("Index for %s with fields %v already exists.\n",
				required.collectionGroup, getFieldPaths(required.fields))
			mu.Unlock()
		}
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errors)

	// Check if any errors occurred
	hasErrors := false
	for err := range errors {
		if err != nil {
			hasErrors = true
		}
	}

	if hasErrors {
		return fmt.Errorf("one or more indexes failed to create")
	}
	return nil
}

func getExistingIndexes(projectID, databaseID string) ([]index, error) {
	cmd := exec.Command("gcloud", "firestore", "indexes", "composite", "list",
		"--project="+projectID,
		"--database="+databaseID,
		"--format=json")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute gcloud command: %w", err)
	}

	var indexes []index
	if err := json.Unmarshal(output, &indexes); err != nil {
		return nil, fmt.Errorf("failed to parse indexes: %w", err)
	}

	// Set CollectionGroup from name if not set
	for i := range indexes {
		if indexes[i].CollectionGroup == "" {
			indexes[i].CollectionGroup = parseCollectionFromName(indexes[i].Name)
		}
	}

	return indexes, nil
}

func createIndex(projectID, databaseID string, config indexConfig) error {
	fmt.Printf("Creating index for %s with fields %v...\n", config.collectionGroup, getFieldPaths(config.fields))

	args := []string{
		"firestore", "indexes", "composite", "create",
		"--project=" + projectID,
		"--database=" + databaseID,
		"--collection-group=" + config.collectionGroup,
		"--query-scope=COLLECTION",
	}

	// Add field configs
	for _, field := range config.fields {
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

	cmd := exec.Command("gcloud", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create index: %w, output: %s", err, string(output))
	}

	return nil
}

func indexExists(existingIndexes []index, required indexConfig) bool {
	for _, existing := range existingIndexes {
		if existing.CollectionGroup != required.collectionGroup {
			continue
		}

		newFields := append(required.fields, indexField{
			FieldPath: "__name__",
			Order:     "*",
		})
		if fieldsMatch(existing.Fields, newFields) {
			return true
		}
	}
	return false
}

func fieldsMatch(existing, new []indexField) bool {
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

func getFieldPaths(fields []indexField) []string {
	paths := make([]string, len(fields))
	for i, f := range fields {
		paths[i] = f.FieldPath
	}
	return paths
}
