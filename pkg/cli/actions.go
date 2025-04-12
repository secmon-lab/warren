package cli

import (
	"log/slog"

	"github.com/secmon-lab/warren/pkg/action/otx"
	"github.com/secmon-lab/warren/pkg/action/urlscan"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/urfave/cli/v3"
)

type actionList []interfaces.Action

var actions = actionList{
	&urlscan.Action{},
	&otx.Action{},
	// &bigquery.Action{},
}

func (x actionList) Flags() []cli.Flag {
	flags := []cli.Flag{}
	for _, action := range x {
		flags = append(flags, action.Flags()...)
	}
	return flags
}

func (x actionList) LogValue() slog.Value {
	var attrs []slog.Attr
	for _, action := range x {
		attrs = append(attrs, slog.Any(action.Name(), action.LogValue()))
	}
	return slog.GroupValue(attrs...)
}
