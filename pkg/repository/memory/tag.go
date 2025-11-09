package memory

import (
	"context"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/tag"
)

// Tag management methods

func (r *Memory) RemoveTagFromAllAlerts(ctx context.Context, name string) error {
	// First, look up the tag by name to get its ID
	tag, err := r.GetTagByName(ctx, name)
	if err != nil {
		return goerr.Wrap(err, "failed to get tag by name")
	}
	if tag == nil {
		// Tag doesn't exist, nothing to remove
		return nil
	}

	// Use the new ID-based removal method
	return r.RemoveTagIDFromAllAlerts(ctx, tag.ID)
}

func (r *Memory) RemoveTagFromAllTickets(ctx context.Context, name string) error {
	// First, look up the tag by name to get its ID
	tag, err := r.GetTagByName(ctx, name)
	if err != nil {
		return goerr.Wrap(err, "failed to get tag by name")
	}
	if tag == nil {
		// Tag doesn't exist, nothing to remove
		return nil
	}

	// Use the new ID-based removal method
	return r.RemoveTagIDFromAllTickets(ctx, tag.ID)
}

// New ID-based tag management methods

func (r *Memory) GetTagByID(ctx context.Context, tagID string) (*tag.Tag, error) {
	r.tagMu.RLock()
	defer r.tagMu.RUnlock()

	if tagData, exists := r.tagsV2[tagID]; exists {
		// Return a copy to prevent external modification
		tagCopy := *tagData
		return &tagCopy, nil
	}

	return nil, nil
}

func (r *Memory) GetTagsByIDs(ctx context.Context, tagIDs []string) ([]*tag.Tag, error) {
	r.tagMu.RLock()
	defer r.tagMu.RUnlock()

	tags := make([]*tag.Tag, 0, len(tagIDs))
	for _, tagID := range tagIDs {
		if tagData, exists := r.tagsV2[tagID]; exists {
			// Return a copy to prevent external modification
			tagCopy := *tagData
			tags = append(tags, &tagCopy)
		}
	}

	return tags, nil
}

func (r *Memory) CreateTagWithID(ctx context.Context, tag *tag.Tag) error {
	r.tagMu.Lock()
	defer r.tagMu.Unlock()

	if tag.ID == "" {
		return goerr.New("tag ID is required")
	}

	if _, exists := r.tagsV2[tag.ID]; exists {
		return goerr.New("tag ID already exists", goerr.V("tagID", tag.ID))
	}

	// Set timestamps if not already set
	now := time.Now()
	tagCopy := *tag
	if tagCopy.CreatedAt.IsZero() {
		tagCopy.CreatedAt = now
	}
	if tagCopy.UpdatedAt.IsZero() {
		tagCopy.UpdatedAt = now
	}

	r.tagsV2[tag.ID] = &tagCopy

	return nil
}

func (r *Memory) UpdateTag(ctx context.Context, tag *tag.Tag) error {
	r.tagMu.Lock()
	defer r.tagMu.Unlock()

	if tag.ID == "" {
		return goerr.New("tag ID is required")
	}

	// Set UpdatedAt timestamp
	tagCopy := *tag
	tagCopy.UpdatedAt = time.Now()
	r.tagsV2[tag.ID] = &tagCopy

	return nil
}

func (r *Memory) DeleteTagByID(ctx context.Context, tagID string) error {
	r.tagMu.Lock()
	defer r.tagMu.Unlock()

	delete(r.tagsV2, tagID)
	return nil
}

func (r *Memory) RemoveTagIDFromAllAlerts(ctx context.Context, tagID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Iterate through all alerts and remove the tag ID
	for _, alert := range r.alerts {
		if alert.TagIDs != nil {
			// Remove tagID from map
			delete(alert.TagIDs, tagID)
		}
	}

	return nil
}

func (r *Memory) RemoveTagIDFromAllTickets(ctx context.Context, tagID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Iterate through all tickets and remove the tag ID
	for _, ticket := range r.tickets {
		if ticket.TagIDs != nil {
			// Remove tagID from map
			delete(ticket.TagIDs, tagID)
		}
	}

	return nil
}

func (r *Memory) GetTagByName(ctx context.Context, name string) (*tag.Tag, error) {
	r.tagMu.RLock()
	defer r.tagMu.RUnlock()

	for _, tagData := range r.tagsV2 {
		if tagData.Name == name {
			// Return a copy to prevent external modification
			tagCopy := *tagData
			return &tagCopy, nil
		}
	}

	return nil, nil
}

func (r *Memory) IsTagNameExists(ctx context.Context, name string) (bool, error) {
	r.tagMu.RLock()
	defer r.tagMu.RUnlock()

	for _, tagData := range r.tagsV2 {
		if tagData.Name == name {
			return true, nil
		}
	}

	return false, nil
}

// GetOrCreateTagByName atomically gets an existing tag or creates a new one
func (r *Memory) GetOrCreateTagByName(ctx context.Context, name, description, color, createdBy string) (*tag.Tag, error) {
	r.tagMu.Lock()
	defer r.tagMu.Unlock()

	// First check if tag already exists by name
	for _, tagData := range r.tagsV2 {
		if tagData.Name == name {
			return tagData, nil
		}
	}

	// Tag doesn't exist, create it
	// Generate unique ID with collision retry
	var tagID string
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		tagID = tag.NewID()
		if _, exists := r.tagsV2[tagID]; !exists {
			break // No collision
		}
		if i == maxRetries-1 {
			return nil, goerr.New("failed to generate unique tag ID after retries")
		}
	}

	// Use provided color or generate one
	if color == "" {
		color = tag.GenerateColor(name)
	}

	// Create the new tag
	now := time.Now()
	newTag := &tag.Tag{
		ID:          tagID,
		Name:        name,
		Description: description,
		Color:       color,
		CreatedBy:   createdBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Store the tag
	r.tagsV2[tagID] = newTag

	return newTag, nil
}

func (r *Memory) ListAllTags(ctx context.Context) ([]*tag.Tag, error) {
	r.tagMu.RLock()
	defer r.tagMu.RUnlock()

	tags := make([]*tag.Tag, 0, len(r.tagsV2))
	for _, tagData := range r.tagsV2 {
		// Return a copy to prevent external modification
		tagCopy := *tagData
		tags = append(tags, &tagCopy)
	}

	return tags, nil
}
