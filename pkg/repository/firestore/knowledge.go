package firestore

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/knowledge"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	collectionTopics     = "topics"
	collectionKnowledges = "knowledges"
	collectionCommits    = "commits"
)

type slugDoc struct {
	State string
}

type commitDoc struct {
	Slug      string
	Name      string
	Content   string
	CommitID  string
	Author    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (c *commitDoc) toKnowledge(topic types.KnowledgeTopic, state types.KnowledgeState) *knowledge.Knowledge {
	return &knowledge.Knowledge{
		Slug:      types.KnowledgeSlug(c.Slug),
		Name:      c.Name,
		Topic:     topic,
		Content:   c.Content,
		CommitID:  c.CommitID,
		Author:    types.UserID(c.Author),
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		State:     state,
	}
}

func (r *Firestore) topicDoc(topic types.KnowledgeTopic) *firestore.DocumentRef {
	return r.db.Collection(collectionTopics).Doc(topic.String())
}

func (r *Firestore) slugDoc(topic types.KnowledgeTopic, slug types.KnowledgeSlug) *firestore.DocumentRef {
	return r.topicDoc(topic).Collection(collectionKnowledges).Doc(slug.String())
}

func (r *Firestore) commitDoc(topic types.KnowledgeTopic, slug types.KnowledgeSlug, commitID string) *firestore.DocumentRef {
	return r.slugDoc(topic, slug).Collection(collectionCommits).Doc(commitID)
}

func (r *Firestore) getSlugState(ctx context.Context, topic types.KnowledgeTopic, slug types.KnowledgeSlug) (types.KnowledgeState, error) {
	doc, err := r.slugDoc(topic, slug).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", nil
		}
		return "", r.eb.Wrap(err, "failed to get slug document", goerr.V("topic", topic), goerr.V("slug", slug))
	}

	var data slugDoc
	if err := doc.DataTo(&data); err != nil {
		return "", r.eb.Wrap(err, "failed to parse slug document", goerr.V("topic", topic), goerr.V("slug", slug))
	}

	return types.KnowledgeState(data.State), nil
}

func (r *Firestore) getLatestCommit(ctx context.Context, topic types.KnowledgeTopic, slug types.KnowledgeSlug, state types.KnowledgeState) (*knowledge.Knowledge, error) {
	iter := r.slugDoc(topic, slug).Collection(collectionCommits).
		OrderBy("UpdatedAt", firestore.Desc).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, nil
	}
	if err != nil {
		return nil, r.eb.Wrap(err, "failed to get latest commit", goerr.V("topic", topic), goerr.V("slug", slug))
	}

	var commit commitDoc
	if err := doc.DataTo(&commit); err != nil {
		return nil, r.eb.Wrap(err, "failed to parse commit", goerr.V("topic", topic), goerr.V("slug", slug))
	}

	return commit.toKnowledge(topic, state), nil
}

func (r *Firestore) GetKnowledges(ctx context.Context, topic types.KnowledgeTopic) ([]*knowledge.Knowledge, error) {
	iter := r.topicDoc(topic).Collection(collectionKnowledges).Documents(ctx)
	defer iter.Stop()

	var result []*knowledge.Knowledge
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, r.eb.Wrap(err, "failed to iterate knowledges", goerr.V("topic", topic))
		}

		var slug slugDoc
		if err := doc.DataTo(&slug); err != nil {
			return nil, r.eb.Wrap(err, "failed to parse slug document", goerr.V("topic", topic), goerr.V("slug", doc.Ref.ID))
		}

		state := types.KnowledgeState(slug.State)
		if !state.IsActive() {
			continue
		}

		k, err := r.getLatestCommit(ctx, topic, types.KnowledgeSlug(doc.Ref.ID), state)
		if err != nil {
			return nil, err
		}
		if k != nil {
			result = append(result, k)
		}
	}

	return result, nil
}

func (r *Firestore) GetKnowledge(ctx context.Context, topic types.KnowledgeTopic, slug types.KnowledgeSlug) (*knowledge.Knowledge, error) {
	state, err := r.getSlugState(ctx, topic, slug)
	if err != nil {
		return nil, err
	}
	if state == "" || !state.IsActive() {
		return nil, nil
	}

	return r.getLatestCommit(ctx, topic, slug, state)
}

func (r *Firestore) GetKnowledgeByCommit(ctx context.Context, topic types.KnowledgeTopic, slug types.KnowledgeSlug, commitID string) (*knowledge.Knowledge, error) {
	state, err := r.getSlugState(ctx, topic, slug)
	if err != nil {
		return nil, err
	}

	doc, err := r.commitDoc(topic, slug, commitID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, r.eb.Wrap(err, "failed to get commit", goerr.V("topic", topic), goerr.V("slug", slug), goerr.V("commit_id", commitID))
	}

	var commit commitDoc
	if err := doc.DataTo(&commit); err != nil {
		return nil, r.eb.Wrap(err, "failed to parse commit", goerr.V("topic", topic), goerr.V("slug", slug), goerr.V("commit_id", commitID))
	}

	// Use current state or default to active if state is empty (backward compatibility)
	if state == "" {
		state = types.KnowledgeStateActive
	}

	return commit.toKnowledge(topic, state), nil
}

func (r *Firestore) ListKnowledgeSlugs(ctx context.Context, topic types.KnowledgeTopic) ([]*knowledge.SlugInfo, error) {
	iter := r.topicDoc(topic).Collection(collectionKnowledges).Documents(ctx)
	defer iter.Stop()

	var result []*knowledge.SlugInfo
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, r.eb.Wrap(err, "failed to iterate knowledges", goerr.V("topic", topic))
		}

		var slug slugDoc
		if err := doc.DataTo(&slug); err != nil {
			return nil, r.eb.Wrap(err, "failed to parse slug document", goerr.V("topic", topic), goerr.V("slug", doc.Ref.ID))
		}

		state := types.KnowledgeState(slug.State)
		if !state.IsActive() {
			continue
		}

		// Get latest commit to get the name
		k, err := r.getLatestCommit(ctx, topic, types.KnowledgeSlug(doc.Ref.ID), state)
		if err != nil {
			return nil, err
		}
		if k != nil {
			result = append(result, &knowledge.SlugInfo{
				Slug: k.Slug,
				Name: k.Name,
			})
		}
	}

	return result, nil
}

func (r *Firestore) PutKnowledge(ctx context.Context, k *knowledge.Knowledge) error {
	if err := k.Validate(); err != nil {
		return goerr.Wrap(err, "invalid knowledge")
	}

	bw := r.db.BulkWriter(ctx)
	var jobs []*firestore.BulkWriterJob

	// Set/update slug state to active
	slugRef := r.slugDoc(k.Topic, k.Slug)
	job1, err := bw.Set(slugRef, slugDoc{
		State: types.KnowledgeStateActive.String(),
	}, firestore.Merge([]firestore.FieldPath{{"State"}}...))
	if err != nil {
		return r.eb.Wrap(err, "failed to set slug state", goerr.V("topic", k.Topic), goerr.V("slug", k.Slug))
	}
	jobs = append(jobs, job1)

	// Store the commit
	commitRef := r.commitDoc(k.Topic, k.Slug, k.CommitID)
	job2, err := bw.Set(commitRef, commitDoc{
		Slug:      k.Slug.String(),
		Name:      k.Name,
		Content:   k.Content,
		CommitID:  k.CommitID,
		Author:    k.Author.String(),
		CreatedAt: k.CreatedAt,
		UpdatedAt: k.UpdatedAt,
	})
	if err != nil {
		return r.eb.Wrap(err, "failed to set commit", goerr.V("topic", k.Topic), goerr.V("slug", k.Slug))
	}
	jobs = append(jobs, job2)

	bw.End()

	// Wait for all jobs to complete
	for _, job := range jobs {
		if _, err := job.Results(); err != nil {
			return r.eb.Wrap(err, "failed to commit knowledge", goerr.V("topic", k.Topic), goerr.V("slug", k.Slug))
		}
	}

	return nil
}

func (r *Firestore) ArchiveKnowledge(ctx context.Context, topic types.KnowledgeTopic, slug types.KnowledgeSlug) error {
	_, err := r.slugDoc(topic, slug).Set(ctx, slugDoc{
		State: types.KnowledgeStateArchived.String(),
	}, firestore.Merge([]firestore.FieldPath{{"State"}}...))
	if err != nil {
		return r.eb.Wrap(err, "failed to archive knowledge", goerr.V("topic", topic), goerr.V("slug", slug))
	}

	return nil
}

func (r *Firestore) CalculateKnowledgeSize(ctx context.Context, topic types.KnowledgeTopic) (int, error) {
	knowledges, err := r.GetKnowledges(ctx, topic)
	if err != nil {
		return 0, err
	}

	totalSize := 0
	for _, k := range knowledges {
		totalSize += k.Size()
	}

	return totalSize, nil
}

func (r *Firestore) ListKnowledgeTopics(ctx context.Context) ([]*knowledge.TopicSummary, error) {
	iter := r.db.Collection(collectionTopics).Documents(ctx)
	defer iter.Stop()

	var result []*knowledge.TopicSummary
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, r.eb.Wrap(err, "failed to iterate topics")
		}

		topic := types.KnowledgeTopic(doc.Ref.ID)

		// Count active knowledges in this topic
		knowledges, err := r.GetKnowledges(ctx, topic)
		if err != nil {
			return nil, err
		}

		if len(knowledges) > 0 {
			result = append(result, &knowledge.TopicSummary{
				Topic: topic,
				Count: len(knowledges),
			})
		}
	}

	return result, nil
}
