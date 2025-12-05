package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/knowledge"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

type slugMeta struct {
	state types.KnowledgeState
}

type knowledgeStore struct {
	mu         sync.RWMutex
	slugStates map[string]map[string]*slugMeta                       // [topic][slug] -> slugMeta
	commits    map[string]map[string]map[string]*knowledge.Knowledge // [topic][slug][commitID] -> Knowledge
}

func newKnowledgeStore() *knowledgeStore {
	return &knowledgeStore{
		slugStates: make(map[string]map[string]*slugMeta),
		commits:    make(map[string]map[string]map[string]*knowledge.Knowledge),
	}
}

func (s *knowledgeStore) getLatestCommit(topic, slug string) *knowledge.Knowledge {
	commits, ok := s.commits[topic][slug]
	if !ok || len(commits) == 0 {
		return nil
	}

	// Find the latest by UpdatedAt
	var latest *knowledge.Knowledge
	for _, k := range commits {
		if latest == nil || k.UpdatedAt.After(latest.UpdatedAt) {
			latest = k
		}
	}
	return latest
}

func (r *Memory) GetKnowledges(ctx context.Context, topic types.KnowledgeTopic) ([]*knowledge.Knowledge, error) {
	r.knowledge.mu.RLock()
	defer r.knowledge.mu.RUnlock()

	topicStr := topic.String()
	slugs, ok := r.knowledge.slugStates[topicStr]
	if !ok {
		return nil, nil
	}

	var result []*knowledge.Knowledge
	for slug, meta := range slugs {
		if meta.state.IsActive() {
			if k := r.knowledge.getLatestCommit(topicStr, slug); k != nil {
				result = append(result, k)
			}
		}
	}

	// Sort by slug for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].Slug < result[j].Slug
	})

	return result, nil
}

func (r *Memory) GetKnowledge(ctx context.Context, topic types.KnowledgeTopic, slug types.KnowledgeSlug) (*knowledge.Knowledge, error) {
	r.knowledge.mu.RLock()
	defer r.knowledge.mu.RUnlock()

	topicStr := topic.String()
	slugStr := slug.String()

	meta, ok := r.knowledge.slugStates[topicStr][slugStr]
	if !ok || !meta.state.IsActive() {
		return nil, nil
	}

	return r.knowledge.getLatestCommit(topicStr, slugStr), nil
}

func (r *Memory) GetKnowledgeByCommit(ctx context.Context, topic types.KnowledgeTopic, slug types.KnowledgeSlug, commitID string) (*knowledge.Knowledge, error) {
	r.knowledge.mu.RLock()
	defer r.knowledge.mu.RUnlock()

	topicStr := topic.String()
	slugStr := slug.String()

	if commits, ok := r.knowledge.commits[topicStr][slugStr]; ok {
		if k, ok := commits[commitID]; ok {
			return k, nil
		}
	}

	return nil, nil
}

func (r *Memory) ListKnowledgeSlugs(ctx context.Context, topic types.KnowledgeTopic) ([]*knowledge.SlugInfo, error) {
	r.knowledge.mu.RLock()
	defer r.knowledge.mu.RUnlock()

	topicStr := topic.String()
	slugs, ok := r.knowledge.slugStates[topicStr]
	if !ok {
		return nil, nil
	}

	var result []*knowledge.SlugInfo
	for slugStr, meta := range slugs {
		if meta.state.IsActive() {
			if k := r.knowledge.getLatestCommit(topicStr, slugStr); k != nil {
				result = append(result, &knowledge.SlugInfo{
					Slug: k.Slug,
					Name: k.Name,
				})
			}
		}
	}

	// Sort by slug for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].Slug < result[j].Slug
	})

	return result, nil
}

func (r *Memory) PutKnowledge(ctx context.Context, k *knowledge.Knowledge) error {
	if err := k.Validate(); err != nil {
		return goerr.Wrap(err, "invalid knowledge")
	}

	r.knowledge.mu.Lock()
	defer r.knowledge.mu.Unlock()

	topicStr := k.Topic.String()
	slugStr := k.Slug.String()

	// Initialize topic if needed
	if r.knowledge.slugStates[topicStr] == nil {
		r.knowledge.slugStates[topicStr] = make(map[string]*slugMeta)
	}
	if r.knowledge.commits[topicStr] == nil {
		r.knowledge.commits[topicStr] = make(map[string]map[string]*knowledge.Knowledge)
	}

	// Set/update slug state to active
	r.knowledge.slugStates[topicStr][slugStr] = &slugMeta{
		state: types.KnowledgeStateActive,
	}

	// Initialize slug commits if needed
	if r.knowledge.commits[topicStr][slugStr] == nil {
		r.knowledge.commits[topicStr][slugStr] = make(map[string]*knowledge.Knowledge)
	}

	// Store the commit
	// Make a copy to avoid external mutations
	kCopy := *k
	r.knowledge.commits[topicStr][slugStr][k.CommitID] = &kCopy

	return nil
}

func (r *Memory) ArchiveKnowledge(ctx context.Context, topic types.KnowledgeTopic, slug types.KnowledgeSlug) error {
	r.knowledge.mu.Lock()
	defer r.knowledge.mu.Unlock()

	topicStr := topic.String()
	slugStr := slug.String()

	meta, ok := r.knowledge.slugStates[topicStr][slugStr]
	if !ok {
		// Slug doesn't exist, nothing to archive
		return nil
	}

	meta.state = types.KnowledgeStateArchived
	return nil
}

func (r *Memory) CalculateKnowledgeSize(ctx context.Context, topic types.KnowledgeTopic) (int, error) {
	r.knowledge.mu.RLock()
	defer r.knowledge.mu.RUnlock()

	topicStr := topic.String()
	slugs, ok := r.knowledge.slugStates[topicStr]
	if !ok {
		return 0, nil
	}

	totalSize := 0
	for slug, meta := range slugs {
		if meta.state.IsActive() {
			if k := r.knowledge.getLatestCommit(topicStr, slug); k != nil {
				totalSize += k.Size()
			}
		}
	}

	return totalSize, nil
}

func (r *Memory) ListKnowledgeTopics(ctx context.Context) ([]*knowledge.TopicSummary, error) {
	r.knowledge.mu.RLock()
	defer r.knowledge.mu.RUnlock()

	var result []*knowledge.TopicSummary
	for topicStr, slugs := range r.knowledge.slugStates {
		count := 0
		for slug, meta := range slugs {
			if meta.state.IsActive() {
				if k := r.knowledge.getLatestCommit(topicStr, slug); k != nil {
					count++
				}
			}
		}

		if count > 0 {
			result = append(result, &knowledge.TopicSummary{
				Topic: types.KnowledgeTopic(topicStr),
				Count: count,
			})
		}
	}

	// Sort by topic for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].Topic < result[j].Topic
	})

	return result, nil
}
