package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/secmon-lab/warren/pkg/domain/model/knowledge"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

type knowledgeStore struct {
	mu         sync.RWMutex
	knowledges map[types.KnowledgeID]*knowledge.Knowledge
	logs       map[types.KnowledgeID]map[types.KnowledgeLogID]*knowledge.KnowledgeLog
	tags       map[types.KnowledgeTagID]*knowledge.KnowledgeTag
}

func newKnowledgeStore() *knowledgeStore {
	return &knowledgeStore{
		knowledges: make(map[types.KnowledgeID]*knowledge.Knowledge),
		logs:       make(map[types.KnowledgeID]map[types.KnowledgeLogID]*knowledge.KnowledgeLog),
		tags:       make(map[types.KnowledgeTagID]*knowledge.KnowledgeTag),
	}
}

func (r *Memory) ListAllKnowledges(_ context.Context) ([]*knowledge.Knowledge, error) {
	r.knowledge.mu.RLock()
	defer r.knowledge.mu.RUnlock()

	var result []*knowledge.Knowledge
	for _, k := range r.knowledge.knowledges {
		cp := *k
		cp.Tags = append([]types.KnowledgeTagID(nil), k.Tags...)
		result = append(result, &cp)
	}
	return result, nil
}

func (r *Memory) GetKnowledge(_ context.Context, id types.KnowledgeID) (*knowledge.Knowledge, error) {
	r.knowledge.mu.RLock()
	defer r.knowledge.mu.RUnlock()

	k, ok := r.knowledge.knowledges[id]
	if !ok {
		return nil, nil
	}
	cp := *k
	cp.Tags = append([]types.KnowledgeTagID(nil), k.Tags...)
	return &cp, nil
}

func (r *Memory) PutKnowledge(_ context.Context, k *knowledge.Knowledge) error {
	r.knowledge.mu.Lock()
	defer r.knowledge.mu.Unlock()

	cp := *k
	cp.Tags = append([]types.KnowledgeTagID(nil), k.Tags...)
	r.knowledge.knowledges[k.ID] = &cp
	return nil
}

func (r *Memory) DeleteKnowledge(_ context.Context, id types.KnowledgeID) error {
	r.knowledge.mu.Lock()
	defer r.knowledge.mu.Unlock()

	delete(r.knowledge.knowledges, id)
	// Note: logs are NOT deleted (orphan logs for audit trail)
	return nil
}

func (r *Memory) ListKnowledgesByCategoryAndTags(_ context.Context, category types.KnowledgeCategory, tagIDs []types.KnowledgeTagID) ([]*knowledge.Knowledge, error) {
	r.knowledge.mu.RLock()
	defer r.knowledge.mu.RUnlock()

	tagSet := make(map[types.KnowledgeTagID]bool, len(tagIDs))
	for _, id := range tagIDs {
		tagSet[id] = true
	}

	var result []*knowledge.Knowledge
	for _, k := range r.knowledge.knowledges {
		if k.Category != category {
			continue
		}

		// Check if knowledge has at least one matching tag
		matched := false
		for _, t := range k.Tags {
			if tagSet[t] {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}

		cp := *k
		cp.Tags = append([]types.KnowledgeTagID(nil), k.Tags...)
		result = append(result, &cp)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})

	return result, nil
}

// Knowledge log methods

func (r *Memory) GetKnowledgeLog(_ context.Context, knowledgeID types.KnowledgeID, logID types.KnowledgeLogID) (*knowledge.KnowledgeLog, error) {
	r.knowledge.mu.RLock()
	defer r.knowledge.mu.RUnlock()

	logs, ok := r.knowledge.logs[knowledgeID]
	if !ok {
		return nil, nil
	}
	l, ok := logs[logID]
	if !ok {
		return nil, nil
	}
	cp := *l
	return &cp, nil
}

func (r *Memory) ListKnowledgeLogs(_ context.Context, knowledgeID types.KnowledgeID) ([]*knowledge.KnowledgeLog, error) {
	r.knowledge.mu.RLock()
	defer r.knowledge.mu.RUnlock()

	logs, ok := r.knowledge.logs[knowledgeID]
	if !ok {
		return nil, nil
	}

	var result []*knowledge.KnowledgeLog
	for _, l := range logs {
		cp := *l
		result = append(result, &cp)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	return result, nil
}

func (r *Memory) PutKnowledgeLog(_ context.Context, log *knowledge.KnowledgeLog) error {
	r.knowledge.mu.Lock()
	defer r.knowledge.mu.Unlock()

	if r.knowledge.logs[log.KnowledgeID] == nil {
		r.knowledge.logs[log.KnowledgeID] = make(map[types.KnowledgeLogID]*knowledge.KnowledgeLog)
	}

	cp := *log
	r.knowledge.logs[log.KnowledgeID][log.ID] = &cp
	return nil
}

// Knowledge tag methods

func (r *Memory) GetKnowledgeTag(_ context.Context, id types.KnowledgeTagID) (*knowledge.KnowledgeTag, error) {
	r.knowledge.mu.RLock()
	defer r.knowledge.mu.RUnlock()

	t, ok := r.knowledge.tags[id]
	if !ok {
		return nil, nil
	}
	cp := *t
	return &cp, nil
}

func (r *Memory) ListKnowledgeTags(_ context.Context) ([]*knowledge.KnowledgeTag, error) {
	r.knowledge.mu.RLock()
	defer r.knowledge.mu.RUnlock()

	var result []*knowledge.KnowledgeTag
	for _, t := range r.knowledge.tags {
		cp := *t
		result = append(result, &cp)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result, nil
}

func (r *Memory) PutKnowledgeTag(_ context.Context, tag *knowledge.KnowledgeTag) error {
	r.knowledge.mu.Lock()
	defer r.knowledge.mu.Unlock()

	cp := *tag
	r.knowledge.tags[tag.ID] = &cp
	return nil
}

func (r *Memory) DeleteKnowledgeTag(_ context.Context, id types.KnowledgeTagID) error {
	r.knowledge.mu.Lock()
	defer r.knowledge.mu.Unlock()

	delete(r.knowledge.tags, id)
	return nil
}
