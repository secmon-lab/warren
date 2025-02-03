package interfaces

import (
	"context"

	"github.com/secmon-lab/warren/pkg/model"
)

type Action interface {
	Spec() model.ActionSpec
	Execute(ctx context.Context, slack SlackService, ssn GenAIChatSession, args model.Arguments) (string, error)
}
