package hitl

import (
	"context"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/hitl"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

const defaultTimeout = 24 * time.Hour

// Presenter handles displaying HITL requests to users.
// Implementations exist for each transport: Slack, CLI, WebSocket, etc.
type Presenter interface {
	// Present displays the HITL request to the user.
	// Called after the request is saved to the repository.
	Present(ctx context.Context, req *hitl.Request) error
}

// noOpPresenter is a presenter that does nothing.
// Used when no transport-specific presenter is available (e.g., non-Slack environments).
// The HITL request is still saved to the repository and can be answered via Web UI or API.
type noOpPresenter struct{}

func (noOpPresenter) Present(_ context.Context, _ *hitl.Request) error { return nil }

// NoOpPresenter returns a Presenter that does nothing on Present.
func NoOpPresenter() Presenter { return noOpPresenter{} }

// Service manages the HITL request lifecycle.
// It is transport-agnostic: presentation is delegated to a Presenter.
type Service struct {
	repo    interfaces.Repository
	timeout time.Duration
}

// Option configures a Service.
type Option func(*Service)

// WithTimeout sets the maximum time to wait for a HITL response.
func WithTimeout(d time.Duration) Option {
	return func(s *Service) { s.timeout = d }
}

// New creates a new HITL service.
func New(repo interfaces.Repository, opts ...Option) *Service {
	s := &Service{
		repo:    repo,
		timeout: defaultTimeout,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// RequestAndWait creates a HITL request, presents it to the user via the given Presenter,
// and blocks until the user responds or the timeout expires.
func (s *Service) RequestAndWait(ctx context.Context, req *hitl.Request, presenter Presenter) (*hitl.Request, error) {
	logger := logging.From(ctx)

	// Save request to repository
	if err := s.repo.PutHITLRequest(ctx, req); err != nil {
		return nil, goerr.Wrap(err, "failed to save HITL request",
			goerr.V("id", req.ID))
	}

	// Start watching before presenting to avoid race conditions
	watchCtx, watchCancel := context.WithTimeout(ctx, s.timeout)
	defer watchCancel()

	ch, errCh := s.repo.WatchHITLRequest(watchCtx, req.ID)

	// Present to user
	if err := presenter.Present(ctx, req); err != nil {
		return nil, goerr.Wrap(err, "failed to present HITL request",
			goerr.V("id", req.ID))
	}

	logger.Info("HITL request waiting for response",
		"id", req.ID,
		"type", req.Type,
		"user_id", req.UserID,
	)

	// Wait for response or timeout
	select {
	case updated, ok := <-ch:
		if !ok {
			// Channel closed without response (context cancelled)
			if watchCtx.Err() != nil {
				return nil, goerr.Wrap(watchCtx.Err(), "HITL request timed out",
					goerr.V("id", req.ID),
					goerr.V("timeout", s.timeout),
					goerr.T(errutil.TagTimeout))
			}
			return nil, goerr.New("HITL watch channel closed unexpectedly",
				goerr.V("id", req.ID),
				goerr.T(errutil.TagInternal))
		}
		logger.Info("HITL request responded",
			"id", req.ID,
			"status", updated.Status,
			"responded_by", updated.RespondedBy,
		)
		return updated, nil

	case err := <-errCh:
		return nil, goerr.Wrap(err, "HITL watch error",
			goerr.V("id", req.ID))

	case <-watchCtx.Done():
		return nil, goerr.Wrap(watchCtx.Err(), "HITL request timed out",
			goerr.V("id", req.ID),
			goerr.V("timeout", s.timeout),
			goerr.T(errutil.TagTimeout))
	}
}

// Respond processes a user's response to a HITL request.
// Updates the request status in the repository, which triggers the Watcher
// in RequestAndWait to unblock the waiting goroutine.
func (s *Service) Respond(ctx context.Context, id types.HITLRequestID, status hitl.Status, respondedBy string, response map[string]any) error {
	if err := s.repo.UpdateHITLRequestStatus(ctx, id, status, respondedBy, response); err != nil {
		return goerr.Wrap(err, "failed to update HITL request status",
			goerr.V("id", id),
			goerr.V("status", status))
	}
	return nil
}
