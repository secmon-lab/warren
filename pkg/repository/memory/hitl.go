package memory

import (
	"context"
	"sync"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/hitl"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
)

type hitlStore struct {
	mu       sync.RWMutex
	requests map[types.HITLRequestID]*hitl.Request
	watchers map[types.HITLRequestID][]chan *hitl.Request
}

func newHITLStore() *hitlStore {
	return &hitlStore{
		requests: make(map[types.HITLRequestID]*hitl.Request),
		watchers: make(map[types.HITLRequestID][]chan *hitl.Request),
	}
}

func (r *Memory) PutHITLRequest(ctx context.Context, req *hitl.Request) error {
	r.hitl.mu.Lock()
	defer r.hitl.mu.Unlock()

	reqCopy := *req
	r.hitl.requests[req.ID] = &reqCopy
	return nil
}

func (r *Memory) GetHITLRequest(ctx context.Context, id types.HITLRequestID) (*hitl.Request, error) {
	r.hitl.mu.RLock()
	defer r.hitl.mu.RUnlock()

	req, ok := r.hitl.requests[id]
	if !ok {
		return nil, r.eb.Wrap(goerr.New("HITL request not found"),
			"not found",
			goerr.T(errutil.TagNotFound),
			goerr.V("id", id))
	}

	reqCopy := *req
	return &reqCopy, nil
}

func (r *Memory) UpdateHITLRequestStatus(ctx context.Context, id types.HITLRequestID, status hitl.Status, respondedBy string, response map[string]any) error {
	r.hitl.mu.Lock()
	defer r.hitl.mu.Unlock()

	req, ok := r.hitl.requests[id]
	if !ok {
		return r.eb.Wrap(goerr.New("HITL request not found"),
			"not found",
			goerr.T(errutil.TagNotFound),
			goerr.V("id", id))
	}

	req.Status = status
	req.RespondedBy = respondedBy
	req.Response = response
	req.RespondedAt = time.Now()

	// Notify all watchers
	reqCopy := *req
	for _, ch := range r.hitl.watchers[id] {
		select {
		case ch <- &reqCopy:
		default:
		}
	}

	return nil
}

func (r *Memory) WatchHITLRequest(ctx context.Context, id types.HITLRequestID) (<-chan *hitl.Request, <-chan error) {
	ch := make(chan *hitl.Request, 1)
	errCh := make(chan error, 1)

	r.hitl.mu.Lock()
	r.hitl.watchers[id] = append(r.hitl.watchers[id], ch)
	r.hitl.mu.Unlock()

	// Clean up watcher when context is cancelled
	go func() {
		<-ctx.Done()
		r.hitl.mu.Lock()
		defer r.hitl.mu.Unlock()

		watchers := r.hitl.watchers[id]
		for i, w := range watchers {
			if w == ch {
				r.hitl.watchers[id] = append(watchers[:i], watchers[i+1:]...)
				break
			}
		}
		close(ch)
	}()

	return ch, errCh
}
