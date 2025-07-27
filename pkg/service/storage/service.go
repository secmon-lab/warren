package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/types"
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

func pathToHistory(prefix string, ticketID types.TicketID, historyID types.HistoryID) string {
	return fmt.Sprintf("%s%s/ticket/%s/history/%s.json", prefix, StorageSchemaVersion, ticketID, historyID)
}

func (s *Service) PutHistory(ctx context.Context, ticketID types.TicketID, historyID types.HistoryID, history *gollem.History) error {
	path := pathToHistory(s.prefix, ticketID, historyID)

	w := s.storageClient.PutObject(ctx, path)

	// Debug: Log what we're saving
	historyJSON, _ := json.Marshal(history)
	fmt.Printf("DEBUG: Saving history to %s: %s\n", path, string(historyJSON))

	if err := json.NewEncoder(w).Encode(history); err != nil {
		return goerr.Wrap(err, "failed to save history")
	}

	if err := w.Close(); err != nil {
		return goerr.Wrap(err, "failed to close history")
	}

	return nil
}

func (s *Service) GetHistory(ctx context.Context, ticketID types.TicketID, historyID types.HistoryID) (*gollem.History, error) {
	path := pathToHistory(s.prefix, ticketID, historyID)

	r, err := s.storageClient.GetObject(ctx, path)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get history")
	}
	defer safe.Close(ctx, r)

	// Debug: Read raw data first
	rawData := make([]byte, 0)
	buf := make([]byte, 1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			rawData = append(rawData, buf[:n]...)
		}
		if err != nil {
			break
		}
	}
	fmt.Printf("DEBUG: Loading history from %s: %s\n", path, string(rawData))

	// Reset reader position
	r, err = s.storageClient.GetObject(ctx, path)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get history (second attempt)")
	}
	defer safe.Close(ctx, r)

	var history gollem.History
	if err := json.NewDecoder(r).Decode(&history); err != nil {
		fmt.Printf("DEBUG: Failed to decode history: %v\n", err)
		return nil, goerr.Wrap(err, "failed to unmarshal history")
	}

	// Test if history can be converted to Gemini format
	if _, err := history.ToGemini(); err != nil {
		fmt.Printf("DEBUG: History ToGemini conversion failed: %v\n", err)
	} else {
		fmt.Printf("DEBUG: Successfully loaded and validated history\n")
	}

	return &history, nil
}
