package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

type sessionRecordStore struct {
	mu      sync.RWMutex
	records map[types.SessionRecordID]*session.SessionRecord
}

func newSessionRecordStore() *sessionRecordStore {
	return &sessionRecordStore{
		records: make(map[types.SessionRecordID]*session.SessionRecord),
	}
}

func (r *Memory) PutSessionRecord(ctx context.Context, record *session.SessionRecord) error {
	r.sessionRecord.mu.Lock()
	defer r.sessionRecord.mu.Unlock()

	// Deep copy to prevent external modification
	copied := *record
	r.sessionRecord.records[record.ID] = &copied

	return nil
}

func (r *Memory) GetSessionRecords(ctx context.Context, sessionID types.SessionID) ([]*session.SessionRecord, error) {
	r.sessionRecord.mu.RLock()
	defer r.sessionRecord.mu.RUnlock()

	var records []*session.SessionRecord
	for _, record := range r.sessionRecord.records {
		if record.SessionID == sessionID {
			// Deep copy to prevent external modification
			copied := *record
			records = append(records, &copied)
		}
	}

	// Sort by CreatedAt ascending (time series order)
	sort.Slice(records, func(i, j int) bool {
		return records[i].CreatedAt.Before(records[j].CreatedAt)
	})

	return records, nil
}
