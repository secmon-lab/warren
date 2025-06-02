package usecase_test

import (
	"context"
	"fmt"
	"testing"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/adapter/storage"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	storage_svc "github.com/secmon-lab/warren/pkg/service/storage"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/utils/ptr"
)

func TestHandlePrompt(t *testing.T) {
	ctx := t.Context()

	mockRepo := repository.NewMemory()
	mockStorage := storage.NewMock()

	mockPolicy := &mock.PolicyClientMock{
		SourcesFunc: func() map[string]string {
			return map[string]string{}
		},
	}

	newSessionCount := 0
	genContentCount := 0
	var contents []*genai.Content
	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			newSessionCount++
			cfg := gollem.NewSessionConfig(opts...)
			switch newSessionCount {
			case 1:
				gt.Nil(t, cfg.History())
			case 2:
				gt.NotNil(t, cfg.History())
			}

			return &mock.LLMSessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					genContentCount++
					contents = append(contents, &genai.Content{
						Role:  "user",
						Parts: []genai.Part{genai.Text(fmt.Sprintf("prompt:%d", genContentCount))},
					})
					contents = append(contents, &genai.Content{
						Role:  "assistant",
						Parts: []genai.Part{genai.Text(fmt.Sprintf("result:%d", genContentCount))},
					})

					return &gollem.Response{
						Texts: []string{fmt.Sprintf("result:%d", genContentCount)},
					}, nil
				},
				HistoryFunc: func() *gollem.History {
					return gollem.NewHistoryFromGemini(contents)
				},
			}, nil
		},
	}

	uc := usecase.New(
		usecase.WithPolicyClient(mockPolicy),
		usecase.WithRepository(mockRepo),
		usecase.WithStorageClient(mockStorage),
		usecase.WithLLMClient(mockLLM),
	)

	alerts := alert.Alerts{
		ptr.Ref(alert.New(ctx, types.AlertSchema("test"), map[string]any{}, alert.Metadata{})),
		ptr.Ref(alert.New(ctx, types.AlertSchema("test"), map[string]any{}, alert.Metadata{})),
	}

	gt.NoError(t, mockRepo.BatchPutAlerts(ctx, alerts))

	ticketID := types.NewTicketID()
	err := uc.Chat(ctx, &ticket.Ticket{ID: ticketID, AlertIDs: []types.AlertID{alerts[0].ID, alerts[1].ID}}, "prompt:1")
	gt.NoError(t, err)

	latestHistory, err := mockRepo.GetLatestHistory(ctx, ticketID)
	gt.NoError(t, err)
	gt.NotNil(t, latestHistory)

	storageSvc := storage_svc.New(mockStorage)
	gt.NoError(t, err)
	history, err := storageSvc.GetHistory(ctx, ticketID, latestHistory.ID)
	gt.NoError(t, err)
	geminiHistory, err := history.ToGemini()
	gt.NoError(t, err)
	gt.A(t, geminiHistory).Length(2).At(0, func(t testing.TB, v *genai.Content) {
		gt.Equal(t, v.Role, "user")
		p := gt.Cast[genai.Text](t, v.Parts[0])
		gt.Equal(t, p, "prompt:1")
	})

	err = uc.Chat(ctx, &ticket.Ticket{ID: ticketID}, "prompt:2")
	gt.NoError(t, err)

	latestHistory, err = mockRepo.GetLatestHistory(ctx, ticketID)
	gt.NoError(t, err)
	gt.NotNil(t, latestHistory)

	gt.Equal(t, newSessionCount, 2)
}
