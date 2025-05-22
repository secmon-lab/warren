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
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/usecase"

	storage_svc "github.com/secmon-lab/warren/pkg/service/storage"
)

func TestHandlePrompt(t *testing.T) {
	ctx := context.Background()

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

	input := usecase.HandlePromptInput{
		Ticket:        &ticket.Ticket{ID: types.NewTicketID()},
		Prompt:        "prompt:1",
		LLMClient:     mockLLM,
		Repo:          mockRepo,
		StorageClient: mockStorage,
		PolicyClient:  mockPolicy,
	}

	err := usecase.HandlePrompt(ctx, input)
	gt.NoError(t, err)

	latestHistory, err := mockRepo.GetLatestHistory(ctx, input.Ticket.ID)
	gt.NoError(t, err)
	gt.NotNil(t, latestHistory)

	storageSvc := storage_svc.New(mockStorage)
	history, err := storageSvc.GetHistory(ctx, input.Ticket.ID, latestHistory.ID)
	gt.NoError(t, err)
	geminiHistory, err := history.ToGemini()
	gt.NoError(t, err)
	gt.A(t, geminiHistory).Length(2).At(0, func(t testing.TB, v *genai.Content) {
		gt.Equal(t, v.Role, "user")
		p := gt.Cast[genai.Text](t, v.Parts[0])
		gt.Equal(t, p, "prompt:1")
	})

	input.Prompt = "prompt:2"
	err = usecase.HandlePrompt(ctx, input)
	gt.NoError(t, err)

	latestHistory, err = mockRepo.GetLatestHistory(ctx, input.Ticket.ID)
	gt.NoError(t, err)
	gt.NotNil(t, latestHistory)

	gt.Equal(t, newSessionCount, 2)
}
