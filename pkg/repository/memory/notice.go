package memory

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/notice"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// Notice management methods

func (r *Memory) CreateNotice(ctx context.Context, notice *notice.Notice) error {
	r.noticeMu.Lock()
	defer r.noticeMu.Unlock()

	if notice.ID == types.EmptyNoticeID {
		return r.eb.New("notice ID is empty")
	}

	// Check if notice already exists
	if _, exists := r.notices[notice.ID]; exists {
		return r.eb.New("notice already exists", goerr.V("notice_id", notice.ID))
	}

	// Store a copy to prevent external modification
	noticeCopy := *notice
	r.notices[notice.ID] = &noticeCopy

	return nil
}

func (r *Memory) GetNotice(ctx context.Context, id types.NoticeID) (*notice.Notice, error) {
	r.noticeMu.RLock()
	defer r.noticeMu.RUnlock()

	notice, exists := r.notices[id]
	if !exists {
		return nil, r.eb.New("notice not found", goerr.V("notice_id", id))
	}

	// Return a copy to prevent external modification
	noticeCopy := *notice
	return &noticeCopy, nil
}

func (r *Memory) UpdateNotice(ctx context.Context, notice *notice.Notice) error {
	r.noticeMu.Lock()
	defer r.noticeMu.Unlock()

	if notice.ID == types.EmptyNoticeID {
		return r.eb.New("notice ID is empty")
	}

	// Check if notice exists
	if _, exists := r.notices[notice.ID]; !exists {
		return r.eb.New("notice not found", goerr.V("notice_id", notice.ID))
	}

	// Store a copy to prevent external modification
	noticeCopy := *notice
	r.notices[notice.ID] = &noticeCopy

	return nil
}
