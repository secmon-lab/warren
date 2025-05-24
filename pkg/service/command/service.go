package command

import (
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
)

type Service struct {
	repo interfaces.Repository
	llm  gollem.LLMClient
}

func New(repo interfaces.Repository, llm gollem.LLMClient) *Service {
	return &Service{
		repo: repo,
		llm:  llm,
	}
}
