package hitl_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	hitlModel "github.com/secmon-lab/warren/pkg/domain/model/hitl"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository/memory"
	"github.com/secmon-lab/warren/pkg/service/hitl"
)

type mockPresenter struct {
	presented *hitlModel.Request
}

func (m *mockPresenter) Present(_ context.Context, req *hitlModel.Request) error {
	m.presented = req
	return nil
}

func TestService_RequestAndWait_Approved(t *testing.T) {
	repo := memory.New()
	svc := hitl.New(repo, hitl.WithTimeout(10*time.Second))
	presenter := &mockPresenter{}

	req := &hitlModel.Request{
		ID:        types.NewHITLRequestID(),
		SessionID: types.NewSessionID(),
		Type:      hitlModel.RequestTypeToolApproval,
		Payload:   map[string]any{"tool_name": "web_fetch"},
		Status:    hitlModel.StatusPending,
		UserID:    "U12345",
		CreatedAt: time.Now(),
		SlackThread: slack.Thread{
			ChannelID: "C123",
			ThreadID:  "1234.5678",
		},
	}

	// Respond after a delay
	go func() {
		time.Sleep(300 * time.Millisecond)
		err := svc.Respond(t.Context(), req.ID, hitlModel.StatusApproved, "U67890", map[string]any{"comment": "ok"})
		if err != nil {
			t.Errorf("Respond failed: %v", err)
		}
	}()

	result, err := svc.RequestAndWait(t.Context(), req, presenter)
	gt.NoError(t, err).Required()
	gt.Value(t, result.Status).Equal(hitlModel.StatusApproved)
	gt.Value(t, result.RespondedBy).Equal("U67890")

	// Verify presenter was called
	gt.Value(t, presenter.presented).NotEqual(nil)
	gt.Value(t, presenter.presented.ID).Equal(req.ID)
}

func TestService_RequestAndWait_Denied(t *testing.T) {
	repo := memory.New()
	svc := hitl.New(repo, hitl.WithTimeout(10*time.Second))
	presenter := &mockPresenter{}

	req := &hitlModel.Request{
		ID:        types.NewHITLRequestID(),
		SessionID: types.NewSessionID(),
		Type:      hitlModel.RequestTypeToolApproval,
		Payload:   map[string]any{"tool_name": "web_fetch"},
		Status:    hitlModel.StatusPending,
		UserID:    "U12345",
		CreatedAt: time.Now(),
		SlackThread: slack.Thread{
			ChannelID: "C123",
			ThreadID:  "1234.5678",
		},
	}

	go func() {
		time.Sleep(300 * time.Millisecond)
		_ = svc.Respond(t.Context(), req.ID, hitlModel.StatusDenied, "U67890", map[string]any{"comment": "nope"})
	}()

	result, err := svc.RequestAndWait(t.Context(), req, presenter)
	gt.NoError(t, err).Required()
	gt.Value(t, result.Status).Equal(hitlModel.StatusDenied)
}

func TestService_RequestAndWait_Timeout(t *testing.T) {
	repo := memory.New()
	svc := hitl.New(repo, hitl.WithTimeout(500*time.Millisecond))
	presenter := &mockPresenter{}

	req := &hitlModel.Request{
		ID:        types.NewHITLRequestID(),
		SessionID: types.NewSessionID(),
		Type:      hitlModel.RequestTypeToolApproval,
		Payload:   map[string]any{"tool_name": "web_fetch"},
		Status:    hitlModel.StatusPending,
		UserID:    "U12345",
		CreatedAt: time.Now(),
		SlackThread: slack.Thread{
			ChannelID: "C123",
			ThreadID:  "1234.5678",
		},
	}

	// Don't respond - should timeout
	_, err := svc.RequestAndWait(t.Context(), req, presenter)
	gt.Error(t, err)
}

func TestService_Respond(t *testing.T) {
	repo := memory.New()
	svc := hitl.New(repo)

	ctx := t.Context()

	req := &hitlModel.Request{
		ID:        types.NewHITLRequestID(),
		SessionID: types.NewSessionID(),
		Type:      hitlModel.RequestTypeToolApproval,
		Payload:   map[string]any{"tool_name": "web_fetch"},
		Status:    hitlModel.StatusPending,
		UserID:    "U12345",
		CreatedAt: time.Now(),
		SlackThread: slack.Thread{
			ChannelID: "C123",
			ThreadID:  "1234.5678",
		},
	}

	gt.NoError(t, repo.PutHITLRequest(ctx, req)).Required()

	gt.NoError(t, svc.Respond(ctx, req.ID, hitlModel.StatusApproved, "U67890", map[string]any{"comment": "approved"})).Required()

	got, err := repo.GetHITLRequest(ctx, req.ID)
	gt.NoError(t, err).Required()
	gt.Value(t, got.Status).Equal(hitlModel.StatusApproved)
	gt.Value(t, got.RespondedBy).Equal("U67890")
}
