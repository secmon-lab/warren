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
	timeSpan  time.Duration
	loopLimit int
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

func WithLoopLimit(loopLimit int) Option {
	return func(u *UseCases) {
		u.loopLimit = loopLimit
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
			PostAlertFunc: func(ctx context.Context, alert model.Alert) (string, string, error) {
				return "test", "test", nil
			},
			UpdateAlertFunc: func(ctx context.Context, alert model.Alert) error {
				return nil
			},
			VerifyRequestFunc: func(header http.Header, body []byte) error {
				return nil
			},
		},
		policyClient:  policyClient,
		repository:    repository.NewMemory(),
		actionService: service.NewActionService([]interfaces.Action{}),

		timeSpan:  24 * time.Hour,
		loopLimit: 10,
	}

	for _, opt := range opts {
		opt(u)
	}

	return u
}
