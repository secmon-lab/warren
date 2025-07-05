package usecase

import (
	"context"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service/slack"
)

var (
	ErrSlackServiceNotConfigured = goerr.New("slack service not configured")
)

type UseCases struct {
	// services and adapters
	slackService    *slack.Service
	llmClient       gollem.LLMClient
	embeddingClient interfaces.EmbeddingClient
	repository      interfaces.Repository
	storageClient   interfaces.StorageClient
	policyClient    interfaces.PolicyClient

	tools []gollem.ToolSet

	// configs
	timeSpan      time.Duration
	actionLimit   int
	findingLimit  int
	storagePrefix string
}

var _ interfaces.AlertUsecases = &UseCases{}
var _ interfaces.SlackEventUsecases = &UseCases{}
var _ interfaces.SlackInteractionUsecases = &UseCases{}
var _ interfaces.ApiUsecases = &UseCases{}

type Option func(*UseCases)

func WithLLMClient(llmClient gollem.LLMClient) Option {
	return func(u *UseCases) {
		u.llmClient = llmClient
	}
}

func WithSlackService(slackService *slack.Service) Option {
	return func(u *UseCases) {
		u.slackService = slackService
	}
}

func WithEmbeddingClient(embeddingClient interfaces.EmbeddingClient) Option {
	return func(u *UseCases) {
		u.embeddingClient = embeddingClient
	}
}

func WithPolicyClient(policyClient interfaces.PolicyClient) Option {
	return func(u *UseCases) {
		u.policyClient = policyClient
	}
}

func WithStorageClient(storageClient interfaces.StorageClient) Option {
	return func(u *UseCases) {
		u.storageClient = storageClient
	}
}

func WithRepository(repository interfaces.Repository) Option {
	return func(u *UseCases) {
		u.repository = repository
	}
}

func WithTools(tools []gollem.ToolSet) Option {
	return func(u *UseCases) {
		u.tools = append(u.tools, tools...)
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

func WithStoragePrefix(storagePrefix string) Option {
	return func(u *UseCases) {
		u.storagePrefix = storagePrefix
	}
}

type dummyPolicyClient struct{}

func (c *dummyPolicyClient) Query(ctx context.Context, query string, data, result any, queryOptions ...opaq.QueryOption) error {
	return nil
}

func (c *dummyPolicyClient) Sources() map[string]string {
	return map[string]string{}
}

func New(opts ...Option) *UseCases {
	u := &UseCases{
		repository:   repository.NewMemory(),
		policyClient: &dummyPolicyClient{},
		timeSpan:     24 * time.Hour,
		actionLimit:  10,
		findingLimit: 3,
	}

	for _, opt := range opts {
		opt(u)
	}

	return u
}
