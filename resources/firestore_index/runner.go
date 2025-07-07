package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

// runCreateIndexes executes the main logic for creating Firestore indexes
func runCreateIndexes(ctx context.Context, projectID, databaseID string, dryrun bool) error {
	fmt.Println("Creating Firestore indexes...")
	fmt.Println("----------------------------------------")
	fmt.Printf("Project ID: %s\n", projectID)
	fmt.Printf("Database ID: %s\n", databaseID)
	fmt.Printf("Dry run: %t\n", dryrun)
	fmt.Println("----------------------------------------")

	// Define all required indexes
	requiredIndexes := DefineRequiredIndexes()

	// Get existing indexes
	existingIndexes, err := GetExistingIndexes(projectID, databaseID)
	if err != nil {
		return fmt.Errorf("failed to get existing indexes: %w", err)
	}

	if dryrun {
		checkRequiredIndexes(existingIndexes, requiredIndexes)
		return nil
	}

	// Check and create missing indexes in parallel
	return createMissingIndexes(projectID, databaseID, existingIndexes, requiredIndexes)
}

// checkRequiredIndexes performs a dry-run check of required indexes
func checkRequiredIndexes(existingIndexes []Index, requiredIndexes []IndexConfig) {
	fmt.Println("Checking required indexes...")
	fmt.Println("----------------------------------------")

	for _, required := range requiredIndexes {
		if IndexExists(existingIndexes, required) {
			fmt.Printf("✅ Index for %s with fields %v exists\n",
				required.CollectionGroup, GetFieldPaths(required.Fields))
		} else {
			fmt.Printf("❌ Index for %s with fields %v is missing\n",
				required.CollectionGroup, GetFieldPaths(required.Fields))
			fmt.Printf("  Configuration:\n")
			for _, field := range required.Fields {
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
	fmt.Println("Note: Run without --dry-run to create missing indexes")
}

// createMissingIndexes creates missing indexes in parallel
func createMissingIndexes(projectID, databaseID string, existingIndexes []Index, requiredIndexes []IndexConfig) error {
	var wg sync.WaitGroup
	var mu sync.Mutex // for synchronized output
	errors := make(chan error, len(requiredIndexes))

	// First, show what will be created
	fmt.Println("The following indexes will be created:")
	fmt.Println("----------------------------------------")
	for _, required := range requiredIndexes {
		if !IndexExists(existingIndexes, required) {
			fmt.Printf("Creating index for %s:\n", required.CollectionGroup)
			for _, field := range required.Fields {
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
		if !IndexExists(existingIndexes, required) {
			wg.Add(1)
			go func(config IndexConfig) {
				defer wg.Done()
				if err := CreateIndex(projectID, databaseID, config); err != nil {
					mu.Lock()
					log.Printf("Failed to create index for %s: %v", config.CollectionGroup, err)
					mu.Unlock()
					errors <- err
				}
			}(required)
		} else {
			mu.Lock()
			fmt.Printf("Index for %s with fields %v already exists.\n",
				required.CollectionGroup, GetFieldPaths(required.Fields))
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
