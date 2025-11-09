package firestore

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/tag"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Tag management methods

func (r *Firestore) RemoveTagFromAllAlerts(ctx context.Context, name string) error {
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

func (r *Firestore) RemoveTagFromAllTickets(ctx context.Context, name string) error {
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

func (r *Firestore) GetTagByID(ctx context.Context, tagID string) (*tag.Tag, error) {
	doc, err := r.db.Collection(collectionTags).Doc(tagID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, goerr.Wrap(err, "failed to get tag by ID", goerr.V("tagID", tagID))
	}

	var tagData tag.Tag
	if err := doc.DataTo(&tagData); err != nil {
		return nil, goerr.Wrap(err, "failed to decode tag data", goerr.V("tagID", tagID))
	}

	return &tagData, nil
}

func (r *Firestore) GetTagsByIDs(ctx context.Context, tagIDs []string) ([]*tag.Tag, error) {
	if len(tagIDs) == 0 {
		return []*tag.Tag{}, nil
	}

	// Convert tag IDs to document references
	refs := make([]*firestore.DocumentRef, len(tagIDs))
	for i, tagID := range tagIDs {
		refs[i] = r.db.Collection(collectionTags).Doc(tagID)
	}

	// Batch get documents
	docs, err := r.db.GetAll(ctx, refs)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to batch get tags", goerr.V("tagIDs", tagIDs))
	}

	// Convert to tag structs
	tags := make([]*tag.Tag, 0, len(docs))
	for i, doc := range docs {
		if !doc.Exists() {
			// Skip non-existent tags (they may have been deleted)
			continue
		}

		var tagData tag.Tag
		if err := doc.DataTo(&tagData); err != nil {
			return nil, goerr.Wrap(err, "failed to decode tag data", goerr.V("tagID", tagIDs[i]))
		}
		tags = append(tags, &tagData)
	}

	return tags, nil
}

func (r *Firestore) CreateTagWithID(ctx context.Context, tag *tag.Tag) error {
	if tag.ID == "" {
		return goerr.New("tag ID is required")
	}

	// Check if tag already exists
	existing, err := r.GetTagByID(ctx, tag.ID)
	if err != nil {
		return goerr.Wrap(err, "failed to check existing tag")
	}
	if existing != nil {
		return goerr.New("tag ID already exists", goerr.V("tagID", tag.ID))
	}

	// Set timestamps
	now := clock.Now(ctx)
	tag.CreatedAt = now
	tag.UpdatedAt = now

	// Create the tag document
	_, err = r.db.Collection(collectionTags).Doc(tag.ID).Set(ctx, tag)
	if err != nil {
		return goerr.Wrap(err, "failed to create tag", goerr.V("tagID", tag.ID))
	}

	return nil
}

func (r *Firestore) UpdateTag(ctx context.Context, tag *tag.Tag) error {
	if tag.ID == "" {
		return goerr.New("tag ID is required")
	}

	// Set update timestamp
	tag.UpdatedAt = clock.Now(ctx)

	// Update the tag document
	_, err := r.db.Collection(collectionTags).Doc(tag.ID).Set(ctx, tag)
	if err != nil {
		return goerr.Wrap(err, "failed to update tag", goerr.V("tagID", tag.ID))
	}

	return nil
}

func (r *Firestore) DeleteTagByID(ctx context.Context, tagID string) error {
	// Delete the tag document
	_, err := r.db.Collection(collectionTags).Doc(tagID).Delete(ctx)
	if err != nil {
		return goerr.Wrap(err, "failed to delete tag", goerr.V("tagID", tagID))
	}

	return nil
}

func (r *Firestore) RemoveTagIDFromAllAlerts(ctx context.Context, tagID string) error {
	// Get all alerts and filter in memory (more reliable than nested map queries)
	iter := r.db.Collection(collectionAlerts).Documents(ctx)
	defer iter.Stop()

	bw := r.db.BulkWriter(ctx)
	var jobs []*firestore.BulkWriterJob

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return goerr.Wrap(err, "failed to iterate alerts")
		}

		// Parse the alert document to check if it has the tag
		var alertData map[string]interface{}
		if err := doc.DataTo(&alertData); err != nil {
			return goerr.Wrap(err, "failed to parse alert document", goerr.V("alertID", doc.Ref.ID))
		}

		// Check if this alert has the tag ID
		if tagIDs, ok := alertData["TagIDs"].(map[string]interface{}); ok {
			if _, hasTag := tagIDs[tagID]; hasTag {
				// Remove the tag ID from the document
				job, err := bw.Update(doc.Ref, []firestore.Update{
					{Path: "TagIDs." + tagID, Value: firestore.Delete},
				})
				if err != nil {
					return goerr.Wrap(err, "failed to update alert", goerr.V("alertID", doc.Ref.ID))
				}
				jobs = append(jobs, job)
			}
		}
	}

	bw.End()

	// Wait for all jobs to complete
	for _, job := range jobs {
		if _, err := job.Results(); err != nil {
			return goerr.Wrap(err, "failed to commit bulk writer job")
		}
	}

	return nil
}

func (r *Firestore) RemoveTagIDFromAllTickets(ctx context.Context, tagID string) error {
	// Get all tickets and filter in memory (more reliable than nested map queries)
	iter := r.db.Collection(collectionTickets).Documents(ctx)
	defer iter.Stop()

	bw := r.db.BulkWriter(ctx)
	var jobs []*firestore.BulkWriterJob

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return goerr.Wrap(err, "failed to iterate tickets")
		}

		// Parse the ticket document to check if it has the tag
		var ticketData map[string]interface{}
		if err := doc.DataTo(&ticketData); err != nil {
			return goerr.Wrap(err, "failed to parse ticket document", goerr.V("ticketID", doc.Ref.ID))
		}

		// Check if this ticket has the tag ID
		if tagIDs, ok := ticketData["TagIDs"].(map[string]interface{}); ok {
			if _, hasTag := tagIDs[tagID]; hasTag {
				// Remove the tag ID from the document
				job, err := bw.Update(doc.Ref, []firestore.Update{
					{Path: "TagIDs." + tagID, Value: firestore.Delete},
				})
				if err != nil {
					return goerr.Wrap(err, "failed to update ticket", goerr.V("ticketID", doc.Ref.ID))
				}
				jobs = append(jobs, job)
			}
		}
	}

	bw.End()

	// Wait for all jobs to complete
	for _, job := range jobs {
		if _, err := job.Results(); err != nil {
			return goerr.Wrap(err, "failed to commit bulk writer job")
		}
	}

	return nil
}

func (r *Firestore) GetTagByName(ctx context.Context, name string) (*tag.Tag, error) {
	iter := r.db.Collection(collectionTags).Where("Name", "==", name).Limit(1).Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, nil
	}
	if err != nil {
		return nil, goerr.Wrap(err, "failed to query tag by name", goerr.V("name", name))
	}

	var tagData tag.Tag
	if err := doc.DataTo(&tagData); err != nil {
		return nil, goerr.Wrap(err, "failed to decode tag data", goerr.V("name", name))
	}

	return &tagData, nil
}

func (r *Firestore) IsTagNameExists(ctx context.Context, name string) (bool, error) {
	iter := r.db.Collection(collectionTags).Where("Name", "==", name).Limit(1).Documents(ctx)
	defer iter.Stop()

	_, err := iter.Next()
	if err == iterator.Done {
		return false, nil
	}
	if err != nil {
		return false, goerr.Wrap(err, "failed to check tag name existence", goerr.V("name", name))
	}

	return true, nil
}

// GetOrCreateTagByName atomically gets an existing tag or creates a new one
func (r *Firestore) GetOrCreateTagByName(ctx context.Context, name, description, color, createdBy string) (*tag.Tag, error) {
	// First, try to get existing tag by name
	existingTag, err := r.GetTagByName(ctx, name)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to check for existing tag", goerr.V("name", name))
	}
	if existingTag != nil {
		return existingTag, nil
	}

	// Tag doesn't exist, create it
	// Generate unique ID with collision retry
	var tagID string
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		tagID = tag.NewID()
		existing, err := r.GetTagByID(ctx, tagID)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to check tag ID collision")
		}
		if existing == nil {
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

	// Use Firestore transaction to handle concurrent creation attempts
	err = r.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		// Check again if tag exists (within transaction)
		iter := r.db.Collection(collectionTags).Where("name", "==", name).Limit(1).Documents(ctx)
		defer iter.Stop()

		doc, err := iter.Next()
		if err != iterator.Done {
			if err != nil {
				return goerr.Wrap(err, "failed to check tag existence in transaction")
			}
			// Tag already exists, read it
			var existingData tag.Tag
			if err := doc.DataTo(&existingData); err != nil {
				return goerr.Wrap(err, "failed to parse existing tag data")
			}
			*newTag = existingData // Update newTag with existing data
			return nil
		}

		// Tag still doesn't exist, create it
		docRef := r.db.Collection(collectionTags).Doc(tagID)
		return tx.Set(docRef, newTag)
	})

	if err != nil {
		return nil, goerr.Wrap(err, "failed to create tag", goerr.V("name", name))
	}

	return newTag, nil
}

func (r *Firestore) ListAllTags(ctx context.Context) ([]*tag.Tag, error) {
	iter := r.db.Collection(collectionTags).Documents(ctx)
	defer iter.Stop()

	var tags []*tag.Tag
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(err, "failed to iterate tags")
		}

		var tagData tag.Tag
		if err := doc.DataTo(&tagData); err != nil {
			return nil, goerr.Wrap(err, "failed to decode tag data", goerr.V("docID", doc.Ref.ID))
		}
		tags = append(tags, &tagData)
	}

	return tags, nil
}

func (r *Firestore) PutTag(ctx context.Context, tag *tag.Tag) error {
	return r.UpdateTag(ctx, tag)
}

func (r *Firestore) GetTag(ctx context.Context, tagID string) (*tag.Tag, error) {
	return r.GetTagByID(ctx, tagID)
}

func (r *Firestore) GetTags(ctx context.Context) ([]*tag.Tag, error) {
	return r.ListAllTags(ctx)
}

func (r *Firestore) DeleteTag(ctx context.Context, tagID string) error {
	return r.DeleteTagByID(ctx, tagID)
}

func (r *Firestore) AddTicketToTag(ctx context.Context, tagID string, ticketID types.TicketID) error {
	ticketDoc := r.db.Collection(collectionTickets).Doc(ticketID.String())
	_, err := ticketDoc.Update(ctx, []firestore.Update{
		{
			Path:  "TagIDs." + tagID,
			Value: true,
		},
	})
	if err != nil {
		return goerr.Wrap(err, "failed to add ticket to tag", goerr.V("tag_id", tagID), goerr.V("ticket_id", ticketID))
	}
	return nil
}

func (r *Firestore) RemoveTicketFromTag(ctx context.Context, tagID string, ticketID types.TicketID) error {
	ticketDoc := r.db.Collection(collectionTickets).Doc(ticketID.String())
	_, err := ticketDoc.Update(ctx, []firestore.Update{
		{
			Path:  "TagIDs." + tagID,
			Value: firestore.Delete,
		},
	})
	if err != nil {
		return goerr.Wrap(err, "failed to remove ticket from tag", goerr.V("tag_id", tagID), goerr.V("ticket_id", ticketID))
	}
	return nil
}

func (r *Firestore) GetTicketsByTagName(ctx context.Context, tagName string) ([]types.TicketID, error) {
	// First get the tag to find its ID
	tag, err := r.GetTagByName(ctx, tagName)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get tag by name", goerr.V("tag_name", tagName))
	}
	if tag == nil {
		return []types.TicketID{}, nil
	}

	// Query tickets that have this tag ID
	iter := r.db.Collection(collectionTickets).Documents(ctx)
	defer iter.Stop()

	var ticketIDs []types.TicketID
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(err, "failed to iterate tickets")
		}

		var ticketData map[string]interface{}
		if err := doc.DataTo(&ticketData); err != nil {
			return nil, goerr.Wrap(err, "failed to parse ticket document", goerr.V("ticketID", doc.Ref.ID))
		}

		if tagIDs, ok := ticketData["TagIDs"].(map[string]interface{}); ok {
			if _, hasTag := tagIDs[tag.ID]; hasTag {
				ticketIDs = append(ticketIDs, types.TicketID(doc.Ref.ID))
			}
		}
	}

	return ticketIDs, nil
}

func (r *Firestore) GetTicketsByTagNames(ctx context.Context, tagNames []string) ([]types.TicketID, error) {
	if len(tagNames) == 0 {
		return []types.TicketID{}, nil
	}

	// Get all tag IDs for the given names
	var tagIDs []string
	for _, tagName := range tagNames {
		tag, err := r.GetTagByName(ctx, tagName)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to get tag by name", goerr.V("tag_name", tagName))
		}
		if tag != nil {
			tagIDs = append(tagIDs, tag.ID)
		}
	}

	if len(tagIDs) == 0 {
		return []types.TicketID{}, nil
	}

	// Query tickets that have all these tag IDs
	iter := r.db.Collection(collectionTickets).Documents(ctx)
	defer iter.Stop()

	var ticketIDs []types.TicketID
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(err, "failed to iterate tickets")
		}

		var ticketData map[string]interface{}
		if err := doc.DataTo(&ticketData); err != nil {
			return nil, goerr.Wrap(err, "failed to parse ticket document", goerr.V("ticketID", doc.Ref.ID))
		}

		if ticketTagIDs, ok := ticketData["TagIDs"].(map[string]interface{}); ok {
			// Check if ticket has all required tags
			hasAllTags := true
			for _, tagID := range tagIDs {
				if _, hasTag := ticketTagIDs[tagID]; !hasTag {
					hasAllTags = false
					break
				}
			}
			if hasAllTags {
				ticketIDs = append(ticketIDs, types.TicketID(doc.Ref.ID))
			}
		}
	}

	return ticketIDs, nil
}

func (r *Firestore) GetTagsByTicketID(ctx context.Context, ticketID types.TicketID) ([]*tag.Tag, error) {
	doc, err := r.db.Collection(collectionTickets).Doc(ticketID.String()).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return []*tag.Tag{}, nil
		}
		return nil, goerr.Wrap(err, "failed to get ticket", goerr.V("ticket_id", ticketID))
	}

	var ticketData map[string]interface{}
	if err := doc.DataTo(&ticketData); err != nil {
		return nil, goerr.Wrap(err, "failed to parse ticket document", goerr.V("ticket_id", ticketID))
	}

	tagIDsMap, ok := ticketData["TagIDs"].(map[string]interface{})
	if !ok || len(tagIDsMap) == 0 {
		return []*tag.Tag{}, nil
	}

	var tagIDs []string
	for tagID := range tagIDsMap {
		tagIDs = append(tagIDs, tagID)
	}

	return r.GetTagsByIDs(ctx, tagIDs)
}

func (r *Firestore) DeleteTagsByTicketID(ctx context.Context, ticketID types.TicketID) error {
	ticketDoc := r.db.Collection(collectionTickets).Doc(ticketID.String())
	_, err := ticketDoc.Update(ctx, []firestore.Update{
		{
			Path:  "TagIDs",
			Value: firestore.Delete,
		},
	})
	if err != nil {
		return goerr.Wrap(err, "failed to delete tags from ticket", goerr.V("ticket_id", ticketID))
	}
	return nil
}

func (r *Firestore) CountTags(ctx context.Context) (int, error) {
	result, err := r.db.Collection(collectionTags).NewAggregationQuery().WithCount("total").Get(ctx)
	if err != nil {
		return 0, goerr.Wrap(err, "failed to count tags")
	}

	return extractCountFromAggregationResult(result, "total")
}

func (r *Firestore) QueryTags(ctx context.Context, query string) ([]*tag.Tag, error) {
	// For now, this is a placeholder implementation
	// In a real implementation, this would use full-text search or similar
	return r.ListAllTags(ctx)
}
