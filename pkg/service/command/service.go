package command

import (
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/service/slack"
)

type Service struct {
	repo   interfaces.Repository
	llm    gollem.LLMClient
	thread *slack.ThreadService
}

func New(repo interfaces.Repository, llm gollem.LLMClient, thread *slack.ThreadService) *Service {
	return &Service{
		repo:   repo,
		llm:    llm,
		thread: thread,
	}
}
