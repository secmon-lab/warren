package session

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
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
	return nil
}
