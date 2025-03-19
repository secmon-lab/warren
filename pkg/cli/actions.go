package cli

import (
	"context"
	"errors"
	"log/slog"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/action/bigquery"
	"github.com/secmon-lab/warren/pkg/action/otx"
	"github.com/secmon-lab/warren/pkg/action/urlscan"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model"
	"github.com/urfave/cli/v3"
)

type actionList []interfaces.Action

var actions = actionList{
	&urlscan.Action{},
	&otx.Action{},
	&bigquery.Action{},
}

func (x actionList) Flags() []cli.Flag {
	flags := []cli.Flag{}
	for _, action := range x {
		flags = append(flags, action.Flags()...)
	}
	return flags
}

func (x actionList) Configure(ctx context.Context) ([]interfaces.Action, error) {
	actions := []interfaces.Action{}
	for _, action := range x {
		if err := action.Configure(ctx); err != nil {
			if errors.Is(err, model.ErrActionUnavailable) {
				continue
			}
			return nil, goerr.Wrap(err, "failed to configure action", goerr.V("action", action.Spec().Name))
		}

		actions = append(actions, action)
	}

	return actions, nil
}

func (x actionList) LogValue() slog.Value {
	var attrs []slog.Attr
	for _, action := range x {
		attrs = append(attrs, slog.Any(action.Spec().Name, action.LogValue()))
	}
	return slog.GroupValue(attrs...)
}
