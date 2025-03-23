package service

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/action"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

type Service struct {
	actions map[string]interfaces.Action
}

func New(actions []interfaces.Action) *Service {
	actionsMap := make(map[string]interfaces.Action)
	for _, a := range actions {
		if a.Spec().Name == "" {
			panic(goerr.New("action name is required", goerr.V("action", a)))
		}
		if _, ok := actionsMap[a.Spec().Name]; ok {
			panic(goerr.New("action name is duplicated", goerr.V("action", a)))
		}
		actionsMap[a.Spec().Name] = a
	}

	return &Service{actions: actionsMap}
}

func (x *Service) Spec() []action.ActionSpec {
	specs := make([]action.ActionSpec, 0, len(x.actions))
	for _, a := range x.actions {
		specs = append(specs, a.Spec())
	}
	return specs
}

func (x *Service) Execute(ctx context.Context, slack interfaces.SlackThreadService, name string, llm interfaces.LLMClient, args action.Arguments) (*action.ActionResult, error) {
	logger := logging.From(ctx)
	action, ok := x.actions[name]
	if !ok {
		return nil, goerr.New("unknown action", goerr.V("name", name))
	}

	logger.Info("executing action", "name", name, "args", args)
	return action.Execute(ctx, llm, args)
}
