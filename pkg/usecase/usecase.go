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
	"github.com/secmon-lab/warren/pkg/service/memory"
	"github.com/secmon-lab/warren/pkg/service/notifier"
	slackService "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/service/tag"
)

var (
	ErrSlackServiceNotConfigured = goerr.New("slack service not configured")
)

type UseCases struct {
	// services and adapters
	slackService    *slackService.Service
	tagService      *tag.Service
	memoryService   *memory.Service
	llmClient       gollem.LLMClient
	embeddingClient interfaces.EmbeddingClient
	repository      interfaces.Repository
	storageClient   interfaces.StorageClient
	policyClient    interfaces.PolicyClient

	tools []gollem.ToolSet

	// use cases
	ClusteringUC *ClusteringUseCase
	TagUC        *TagUseCase

	// configs
	timeSpan        time.Duration
	actionLimit     int
	findingLimit    int
	storagePrefix   string
	strictAlert     bool
	noAuthorization bool
	frontendURL     string

	// GenAI
	promptService interfaces.PromptService

	// notifiers
	consoleNotifier interfaces.Notifier
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

func WithSlackService(slackService *slackService.Service) Option {
	return func(u *UseCases) {
		u.slackService = slackService
	}
}

func WithTagService(tagService *tag.Service) Option {
	return func(u *UseCases) {
		u.tagService = tagService
	}
}

func WithMemoryService(memoryService *memory.Service) Option {
	return func(u *UseCases) {
		u.memoryService = memoryService
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

func WithNoAuthorization(noAuthorization bool) Option {
	return func(u *UseCases) {
		u.noAuthorization = noAuthorization
	}
}

func WithPromptService(promptService interfaces.PromptService) Option {
	return func(u *UseCases) {
		u.promptService = promptService
	}
}

func WithFrontendURL(frontendURL string) Option {
	return func(u *UseCases) {
		u.frontendURL = frontendURL
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

	// Initialize console notifier for pipeline events
	if u.consoleNotifier == nil {
		u.consoleNotifier = notifier.NewConsoleNotifier()
	}

	// Initialize clustering use case
	u.ClusteringUC = NewClusteringUseCase(u.repository)

	// Initialize tag use case if tag service is available
	if u.tagService != nil {
		u.TagUC = NewTagUseCase(u.tagService)
	}

	return u
}

// IsSlackEnabled returns whether Slack functionality is enabled
func (u *UseCases) IsSlackEnabled() bool {
	return u.slackService != nil
}

// GetTagService returns the tag service if available
func (u *UseCases) GetTagService() *tag.Service {
	return u.tagService
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

	cmdSvc := command.NewWithUseCase(uc.repository, uc.llmClient, threadService, uc, uc.slackService.GetClient())
	return cmdSvc.Execute(ctx, slackMsg, commandStr)
}
