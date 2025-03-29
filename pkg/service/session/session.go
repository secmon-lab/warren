package session

import (
	"context"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/gemini"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/service/slack"
)

type Service struct {
	repo  interfaces.Repository
	slack *slack.Service
	llm   interfaces.LLMClient
	ssn   *session.Session
}

func New(repository interfaces.Repository, llmClient interfaces.LLMClient, slackService *slack.Service, ssn *session.Session) *Service {
	svc := &Service{
		repo:  repository,
		llm:   llmClient,
		slack: slackService,
		ssn:   ssn,
	}

	return svc
}

func (s *Service) Chat(ctx context.Context, message string) error {
	// Restore history if exists
	histroy, err := s.repo.GetLatestHistory(ctx, s.ssn.ID)
	if err != nil {
		return goerr.Wrap(err, "failed to get latest history")
	}

	llmSession := s.llm.StartChat(gemini.WithHistory(histroy))

	llmSession.SendMessage(ctx, genai.Text(message))

	return nil
}
