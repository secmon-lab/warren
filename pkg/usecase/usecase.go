package usecase

import (
	"time"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/policy"
	"github.com/secmon-lab/warren/pkg/repository"
)

type UseCases struct {
	// services and adapters
	llmClient       interfaces.LLMClient
	embeddingClient interfaces.EmbeddingClient
	repository      interfaces.Repository
	queryFunc       policy.QueryFunc

	// configs
	timeSpan     time.Duration
	actionLimit  int
	findingLimit int
}

// var _ interfaces.UseCase = &UseCases{}

type Option func(*UseCases)

func WithLLMClient(llmClient interfaces.LLMClient) Option {
	return func(u *UseCases) {
		u.llmClient = llmClient
	}
}

func WithEmbeddingClient(embeddingClient interfaces.EmbeddingClient) Option {
	return func(u *UseCases) {
		u.embeddingClient = embeddingClient
	}
}

func WithQueryFunc(queryFunc policy.QueryFunc) Option {
	return func(u *UseCases) {
		u.queryFunc = queryFunc
	}
}

func WithRepository(repository interfaces.Repository) Option {
	return func(u *UseCases) {
		u.repository = repository
	}
}

// WithTimeSpan is used to set the time span for fetching alerts to search similar alerts
func WithTimeSpan(timeSpan time.Duration) Option {
	return func(u *UseCases) {
		u.timeSpan = timeSpan
	}
}

func WithActionLimit(actionLimit int) Option {
	return func(u *UseCases) {
		u.actionLimit = actionLimit
	}
}

func WithFindingLimit(findingLimit int) Option {
	return func(u *UseCases) {
		u.findingLimit = findingLimit
	}
}

func New(opts ...Option) *UseCases {
	u := &UseCases{
		repository: repository.NewMemory(),

		timeSpan:     24 * time.Hour,
		actionLimit:  10,
		findingLimit: 3,
	}

	for _, opt := range opts {
		opt(u)
	}

	return u
}
