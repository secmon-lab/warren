package firestore

import (
	"context"
	"sort"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/knowledge"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	collectionKnowledges    = "knowledges"
	collectionKnowledgeLogs = "logs"
	collectionKnowledgeTags = "knowledge_tags"
)

func (r *Firestore) ListAllKnowledges(ctx context.Context) ([]*knowledge.Knowledge, error) {
	iter := r.db.Collection(collectionKnowledges).Documents(ctx)
	defer iter.Stop()

	var result []*knowledge.Knowledge
	for {
		doc, err := iter.Next()
		if err != nil {
			break
		}
		var k knowledge.Knowledge
		if err := doc.DataTo(&k); err != nil {
			return nil, goerr.Wrap(err, "failed to decode knowledge")
		}
		result = append(result, &k)
	}
	return result, nil
}

func (r *Firestore) GetKnowledge(ctx context.Context, id types.KnowledgeID) (*knowledge.Knowledge, error) {
	doc, err := r.db.Collection(collectionKnowledges).Doc(id.String()).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, goerr.Wrap(err, "failed to get knowledge", goerr.V("id", id))
	}

	var k knowledge.Knowledge
	if err := doc.DataTo(&k); err != nil {
		return nil, goerr.Wrap(err, "failed to decode knowledge", goerr.V("id", id))
	}
	return &k, nil
}

func (r *Firestore) PutKnowledge(ctx context.Context, k *knowledge.Knowledge) error {
	_, err := r.db.Collection(collectionKnowledges).Doc(k.ID.String()).Set(ctx, k)
	if err != nil {
		return goerr.Wrap(err, "failed to put knowledge", goerr.V("id", k.ID))
	}
	return nil
}

func (r *Firestore) DeleteKnowledge(ctx context.Context, id types.KnowledgeID) error {
	_, err := r.db.Collection(collectionKnowledges).Doc(id.String()).Delete(ctx)
	if err != nil {
		return goerr.Wrap(err, "failed to delete knowledge", goerr.V("id", id))
	}
	return nil
}

func (r *Firestore) ListKnowledgesByCategoryAndTags(ctx context.Context, category types.KnowledgeCategory, tagIDs []types.KnowledgeTagID) ([]*knowledge.Knowledge, error) {
	if len(tagIDs) == 0 {
		return nil, goerr.New("at least one tag ID is required for knowledge search")
	}

	// Firestore array-contains only supports a single value, so we query with the first tag
	// and filter the rest in-app
	firstTag := tagIDs[0].String()

	iter := r.db.Collection(collectionKnowledges).
		Where("category", "==", string(category)).
		Where("tags", "array-contains", firstTag).
		Documents(ctx)
	defer iter.Stop()

	additionalTags := make(map[string]bool, len(tagIDs)-1)
	for _, id := range tagIDs[1:] {
		additionalTags[id.String()] = true
	}

	var result []*knowledge.Knowledge
	for {
		doc, err := iter.Next()
		if err != nil {
			if status.Code(err) == codes.OK {
				break
			}
			// Check for iterator done
			break
		}

		var k knowledge.Knowledge
		if err := doc.DataTo(&k); err != nil {
			return nil, goerr.Wrap(err, "failed to decode knowledge")
		}

		// If multiple tags requested, verify the knowledge has all of them
		if len(additionalTags) > 0 {
			tagSet := make(map[string]bool, len(k.Tags))
			for _, t := range k.Tags {
				tagSet[t.String()] = true
			}
			allMatch := true
			for addTag := range additionalTags {
				if !tagSet[addTag] {
					allMatch = false
					break
				}
			}
			if !allMatch {
				continue
			}
		}

		result = append(result, &k)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})

	return result, nil
}

// Knowledge log methods

func (r *Firestore) GetKnowledgeLog(ctx context.Context, knowledgeID types.KnowledgeID, logID types.KnowledgeLogID) (*knowledge.KnowledgeLog, error) {
	doc, err := r.db.Collection(collectionKnowledges).Doc(knowledgeID.String()).
		Collection(collectionKnowledgeLogs).Doc(logID.String()).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, goerr.Wrap(err, "failed to get knowledge log", goerr.V("knowledgeID", knowledgeID), goerr.V("logID", logID))
	}

	var l knowledge.KnowledgeLog
	if err := doc.DataTo(&l); err != nil {
		return nil, goerr.Wrap(err, "failed to decode knowledge log")
	}
	return &l, nil
}

func (r *Firestore) ListKnowledgeLogs(ctx context.Context, knowledgeID types.KnowledgeID) ([]*knowledge.KnowledgeLog, error) {
	iter := r.db.Collection(collectionKnowledges).Doc(knowledgeID.String()).
		Collection(collectionKnowledgeLogs).
		OrderBy("created_at", firestore.Desc).
		Documents(ctx)
	defer iter.Stop()

	var result []*knowledge.KnowledgeLog
	for {
		doc, err := iter.Next()
		if err != nil {
			break
		}

		var l knowledge.KnowledgeLog
		if err := doc.DataTo(&l); err != nil {
			return nil, goerr.Wrap(err, "failed to decode knowledge log")
		}
		result = append(result, &l)
	}

	return result, nil
}

func (r *Firestore) PutKnowledgeLog(ctx context.Context, log *knowledge.KnowledgeLog) error {
	_, err := r.db.Collection(collectionKnowledges).Doc(log.KnowledgeID.String()).
		Collection(collectionKnowledgeLogs).Doc(log.ID.String()).Set(ctx, log)
	if err != nil {
		return goerr.Wrap(err, "failed to put knowledge log", goerr.V("knowledgeID", log.KnowledgeID), goerr.V("logID", log.ID))
	}
	return nil
}

// Knowledge tag methods

func (r *Firestore) GetKnowledgeTag(ctx context.Context, id types.KnowledgeTagID) (*knowledge.KnowledgeTag, error) {
	doc, err := r.db.Collection(collectionKnowledgeTags).Doc(id.String()).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, goerr.Wrap(err, "failed to get knowledge tag", goerr.V("id", id))
	}

	var t knowledge.KnowledgeTag
	if err := doc.DataTo(&t); err != nil {
		return nil, goerr.Wrap(err, "failed to decode knowledge tag", goerr.V("id", id))
	}
	return &t, nil
}

func (r *Firestore) ListKnowledgeTags(ctx context.Context) ([]*knowledge.KnowledgeTag, error) {
	iter := r.db.Collection(collectionKnowledgeTags).Documents(ctx)
	defer iter.Stop()

	var result []*knowledge.KnowledgeTag
	for {
		doc, err := iter.Next()
		if err != nil {
			break
		}

		var t knowledge.KnowledgeTag
		if err := doc.DataTo(&t); err != nil {
			return nil, goerr.Wrap(err, "failed to decode knowledge tag")
		}
		result = append(result, &t)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result, nil
}

func (r *Firestore) PutKnowledgeTag(ctx context.Context, tag *knowledge.KnowledgeTag) error {
	_, err := r.db.Collection(collectionKnowledgeTags).Doc(tag.ID.String()).Set(ctx, tag)
	if err != nil {
		return goerr.Wrap(err, "failed to put knowledge tag", goerr.V("id", tag.ID))
	}
	return nil
}

func (r *Firestore) DeleteKnowledgeTag(ctx context.Context, id types.KnowledgeTagID) error {
	_, err := r.db.Collection(collectionKnowledgeTags).Doc(id.String()).Delete(ctx)
	if err != nil {
		return goerr.Wrap(err, "failed to delete knowledge tag", goerr.V("id", id))
	}
	return nil
}
