package usecase

import (
	"context"
	"net/http"
	"time"

	"github.com/m-mizutani/opac"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/mock"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service"
)

type UseCases struct {
	slackService    interfaces.SlackService
	policyClient    interfaces.PolicyClient
	geminiStartChat interfaces.GetGeminiStartChat
	repository      interfaces.Repository
	actionService   *service.ActionService

	// configs
	timeSpan     time.Duration
	actionLimit  int
	findingLimit int
}

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

func New(geminiStartChat interfaces.GetGeminiStartChat, opts ...Option) *UseCases {
	policyClient, err := opac.New(opac.Data(map[string]string{
		"policy": "package alert.dummy",
	}))
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
			VerifyRequestFunc: func(header http.Header, body []byte) error {
				return nil
			},
		},
		policyClient:  policyClient,
		repository:    repository.NewMemory(),
		actionService: service.NewActionService([]interfaces.Action{}),

		timeSpan:     24 * time.Hour,
		actionLimit:  10,
		findingLimit: 3,
	}

	for _, opt := range opts {
		opt(u)
	}

	return u
}
