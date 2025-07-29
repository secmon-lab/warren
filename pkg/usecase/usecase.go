package usecase

import (
	"context"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service/command"
	slackService "github.com/secmon-lab/warren/pkg/service/slack"
)

var (
	ErrSlackServiceNotConfigured = goerr.New("slack service not configured")
)

type UseCases struct {
	// services and adapters
	slackNotifier   interfaces.SlackNotifier
	slackService    *slackService.Service // Keep concrete service for additional functionality
	llmClient       gollem.LLMClient
	embeddingClient interfaces.EmbeddingClient
	repository      interfaces.Repository
	storageClient   interfaces.StorageClient
	policyClient    interfaces.PolicyClient

	tools []gollem.ToolSet

	// use cases
	ClusteringUC *ClusteringUseCase

	// configs
	timeSpan      time.Duration
	actionLimit   int
	findingLimit  int
	storagePrefix string
	strictAlert   bool
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

func WithSlackNotifier(slackNotifier interfaces.SlackNotifier) Option {
	return func(u *UseCases) {
		u.slackNotifier = slackNotifier
	}
}

// Deprecated: Use WithSlackNotifier instead
func WithSlackService(slackService *slackService.Service) Option {
	return func(u *UseCases) {
		u.slackNotifier = slackService // Set the service as notifier to avoid breaking changes
		u.slackService = slackService  // Keep concrete service for commands
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

func WithStrictAlert(strict bool) Option {
	return func(u *UseCases) {
		u.strictAlert = strict
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
		repository:    repository.NewMemory(),
		policyClient:  &dummyPolicyClient{},
		slackNotifier: NewDiscardSlackNotifier(), // Default to discard implementation
		timeSpan:      24 * time.Hour,
		actionLimit:   10,
		findingLimit:  3,
	}

	for _, opt := range opts {
		opt(u)
	}

	// Initialize clustering use case
	u.ClusteringUC = NewClusteringUseCase(u.repository)

	return u
}

// IsSlackEnabled returns whether Slack functionality is enabled
func (u *UseCases) IsSlackEnabled() bool {
	return u.slackNotifier.IsEnabled()
}

// executeSlackCommand executes a Slack command using the concrete slack service
func (uc *UseCases) executeSlackCommand(ctx context.Context, slackMsg *slack.Message, threadSvc interfaces.SlackThreadService, commandStr string) error {
	// Commands require concrete slack service
	if uc.slackService == nil {
		return command.ErrUnknownCommand
	}

	// Create thread service from thread info
	thread := slack.Thread{
		ChannelID: threadSvc.ChannelID(),
		ThreadID:  threadSvc.ThreadID(),
	}

	// Use concrete slack service to create ThreadService through interface
	threadService := uc.slackService.NewThread(thread)

	cmdSvc := command.New(uc.repository, uc.llmClient, threadService)
	return cmdSvc.Execute(ctx, slackMsg, commandStr)
}
