package usecase

import (
	"context"
	"time"

	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/mock"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service"
)

type UseCases struct {
	// services and adapters
	slackService    interfaces.SlackService
	policyClient    interfaces.PolicyClient
	geminiStartChat interfaces.GetGeminiStartChat
	repository      interfaces.Repository
	actionService   *service.ActionService

	// configs
	timeSpan     time.Duration
	actionLimit  int
	findingLimit int

	// test data
	detectData map[string]any
	ignoreData map[string]any
}

var _ interfaces.UseCase = &UseCases{}

type Option func(*UseCases)

func WithSlackService(slackService interfaces.SlackService) Option {
	return func(u *UseCases) {
		u.slackService = slackService
	}
}

func WithPolicyClient(policyClient interfaces.PolicyClient) Option {
	return func(u *UseCases) {
		u.policyClient = policyClient
	}
}

func WithRepository(repository interfaces.Repository) Option {
	return func(u *UseCases) {
		u.repository = repository
	}
}

func WithActionService(actionService *service.ActionService) Option {
	return func(u *UseCases) {
		u.actionService = actionService
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

func WithTestData(detectData map[string]any, ignoreData map[string]any) Option {
	return func(u *UseCases) {
		u.detectData = detectData
		u.ignoreData = ignoreData
	}
}

func New(geminiStartChat interfaces.GetGeminiStartChat, opts ...Option) *UseCases {
	policyClient, err := opaq.New(opaq.Data("policy", "package alert.dummy"))
	if err != nil {
		panic(err)
	}

	u := &UseCases{
		geminiStartChat: geminiStartChat,
		slackService: &mock.SlackServiceMock{
			PostAlertFunc: func(ctx context.Context, alert model.Alert) (interfaces.SlackThreadService, error) {
				return &mock.SlackThreadServiceMock{
					ChannelIDFunc: func() string {
						return "test"
					},
					ThreadIDFunc: func() string {
						return "test"
					},
				}, nil
			},
		},
		policyClient:  policyClient,
		repository:    repository.NewMemory(),
		actionService: service.NewActionService([]interfaces.Action{}),

		timeSpan:     24 * time.Hour,
		actionLimit:  10,
		findingLimit: 3,

		detectData: make(map[string]any),
		ignoreData: make(map[string]any),
	}

	for _, opt := range opts {
		opt(u)
	}

	return u
}
