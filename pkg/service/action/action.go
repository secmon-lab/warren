package action

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/action"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

type Service struct {
	actions []interfaces.Action
	route   map[string]interfaces.Action
}

func New(ctx context.Context, actions []interfaces.Action) (*Service, error) {
	routeMap := make(map[string]interfaces.Action)

	configuredActions := make([]interfaces.Action, 0, len(actions))
	for _, a := range actions {
		if a.Name() == "" {
			return nil, goerr.New("action name is required", goerr.V("action", a))
		}

		if err := a.Configure(ctx); err != nil {
			if errors.Is(err, errs.ErrActionUnavailable) {
				continue
			}

			return nil, goerr.Wrap(err, "failed to configure action", goerr.V("action", a.Name()))
		}

		for _, d := range a.Specs() {
			if d.Name == "" {
				return nil, goerr.New("function name is required", goerr.V("action", a), goerr.V("declaration", d))
			}

			if existed, ok := routeMap[d.Name]; ok {
				return nil, goerr.New("function name is conflicted", goerr.V("action", a), goerr.V("conflicted", d), goerr.V("existed", existed))
			}

			routeMap[d.Name] = a
		}

		configuredActions = append(configuredActions, a)
	}

	logging.From(ctx).Info("configured actions", "actions", configuredActions)

	return &Service{actions: configuredActions, route: routeMap}, nil
}

func (x *Service) Specs() []*genai.FunctionDeclaration {
	var specs []*genai.FunctionDeclaration
	for _, a := range x.actions {
		specs = append(specs, a.Specs()...)
	}
	return specs
}

func (x *Service) Execute(ctx context.Context, name string, args map[string]any) (*action.Result, error) {
	logger := logging.From(ctx)
	action, ok := x.route[name]
	if !ok {
		return nil, goerr.New("unknown action", goerr.V("name", name))
	}

	var argsStr []string
	for k, v := range args {
		argsStr = append(argsStr, fmt.Sprintf("🔸 %s: `%v`", k, v))
	}

	msg.State(ctx, "⚡ Exec: `%s` with %s", name, strings.Join(argsStr, ", "))
	logger.Info("executing action", "name", name, "args", args)
	return action.Execute(ctx, name, args)
}
