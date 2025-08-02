package main

import (
	"context"
	"fmt"
	"log"

	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service/tag"
	"github.com/secmon-lab/warren/pkg/usecase"
)

func main() {
	ctx := context.Background()
	
	// Create in-memory repository
	repo := repository.NewMemory()
	
	// Create tag service and usecase
	tagSvc := tag.New(repo)
	tagUC := usecase.NewTagUseCase(tagSvc)
	
	// Test case-insensitive tag creation and retrieval
	fmt.Println("Testing case-insensitive tag operations...")
	
	// Create tag with mixed case
	createdTag, err := tagUC.CreateTag(ctx, "Security-Alert")
	if err != nil {
		log.Fatal("Failed to create tag:", err)
	}
	fmt.Printf("Created tag: %s with color: %s\n", createdTag.Name, createdTag.Color)
	
	// Try to get the tag with different casing
	retrievedTag, err := tagSvc.GetTag(ctx, "security-alert")
	if err != nil {
		log.Fatal("Failed to get tag:", err)
	}
	if retrievedTag == nil {
		log.Fatal("Tag not found - case sensitivity issue!")
	}
	fmt.Printf("Retrieved tag: %s with color: %s\n", retrievedTag.Name, retrievedTag.Color)
	
	// Verify they are the same
	if createdTag.Name != retrievedTag.Name || createdTag.Color != retrievedTag.Color {
		log.Fatal("Tag mismatch!")
	}
	
	fmt.Println("✅ Case-insensitive tag operations work correctly!")
}