package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/safe"
)

const (
	StorageSchemaVersion = "v1"
)

type Service struct {
	prefix        string
	storageClient interfaces.StorageClient
}

func New(storageClient interfaces.StorageClient, opts ...Option) *Service {
	s := &Service{storageClient: storageClient}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

type Option func(*Service)

func WithPrefix(prefix string) Option {
	return func(s *Service) {
		s.prefix = prefix
	}
}

func pathToLatestHistory(prefix string, ticketID types.TicketID) string {
	return fmt.Sprintf("%s%s/ticket/%s/latest.json", prefix, StorageSchemaVersion, ticketID)
}

// PutLatestHistory saves the history as the latest snapshot for the given ticket.
// The object at latest.json is overwritten on each call.
func (s *Service) PutLatestHistory(ctx context.Context, ticketID types.TicketID, history *gollem.History) error {
	if s.storageClient == nil {
		return nil
	}

	path := pathToLatestHistory(s.prefix, ticketID)

	w := s.storageClient.PutObject(ctx, path)

	if err := json.NewEncoder(w).Encode(history); err != nil {
		return goerr.Wrap(err, "failed to save latest history",
			goerr.V("path", path),
			goerr.V("ticket_id", ticketID))
	}

	if err := w.Close(); err != nil {
		return goerr.Wrap(err, "failed to close latest history",
			goerr.V("path", path),
			goerr.V("ticket_id", ticketID))
	}

	return nil
}

// GetLatestHistory retrieves the latest history snapshot for the given ticket.
// Returns nil History and nil error if the storage client is not configured.
func (s *Service) GetLatestHistory(ctx context.Context, ticketID types.TicketID) (*gollem.History, error) {
	if s.storageClient == nil {
		return nil, nil
	}

	path := pathToLatestHistory(s.prefix, ticketID)

	r, err := s.storageClient.GetObject(ctx, path)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get latest history",
			goerr.V("path", path),
			goerr.V("ticket_id", ticketID))
	}
	if r == nil {
		return nil, nil
	}
	defer safe.Close(ctx, r)

	var history gollem.History
	if err := json.NewDecoder(r).Decode(&history); err != nil {
		return nil, goerr.Wrap(err, "failed to unmarshal latest history",
			goerr.V("path", path),
			goerr.V("ticket_id", ticketID))
	}

	return &history, nil
}

// HistoryRepo implements gollem.HistoryRepository using the storage service.
// It persists the latest history snapshot for a ticket, keyed by ticket ID.
// Save errors are handled via errutil but not returned to avoid interrupting agent execution.
type HistoryRepo struct {
	svc      *Service
	ticketID types.TicketID
}

// NewHistoryRepo creates a new HistoryRepo for the given ticket.
func NewHistoryRepo(svc *Service, ticketID types.TicketID) *HistoryRepo {
	return &HistoryRepo{
		svc:      svc,
		ticketID: ticketID,
	}
}

// Load retrieves the latest history for the ticket.
// Returns nil History and nil error if not found.
func (r *HistoryRepo) Load(ctx context.Context, _ string) (*gollem.History, error) {
	history, err := r.svc.GetLatestHistory(ctx, r.ticketID)
	if err != nil {
		errutil.Handle(ctx, goerr.Wrap(err, "failed to load latest history from repository", goerr.V("ticket_id", r.ticketID)))
		return nil, nil
	}
	return history, nil
}

// Save persists the history as the latest snapshot for the ticket.
// Errors are handled via errutil but not returned to avoid interrupting agent execution.
func (r *HistoryRepo) Save(ctx context.Context, _ string, history *gollem.History) error {
	if err := r.svc.PutLatestHistory(ctx, r.ticketID, history); err != nil {
		errutil.Handle(ctx, goerr.Wrap(err, "failed to save latest history to repository", goerr.V("ticket_id", r.ticketID)))
	}
	return nil
}

// NewHistoryRepoFromContext creates a new HistoryRepo for the given ticket.
func NewHistoryRepoFromContext(_ context.Context, svc *Service, ticketID types.TicketID) *HistoryRepo {
	return NewHistoryRepo(svc, ticketID)
}
