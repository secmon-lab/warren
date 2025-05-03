package cli

import (
	"log/slog"

	"github.com/m-mizutani/gollam"
	"github.com/secmon-lab/warren/pkg/action/abusech"
	"github.com/secmon-lab/warren/pkg/action/ipdb"
	"github.com/secmon-lab/warren/pkg/action/otx"
	"github.com/secmon-lab/warren/pkg/action/shodan"
	"github.com/secmon-lab/warren/pkg/action/urlscan"
	"github.com/secmon-lab/warren/pkg/action/vt"
	"github.com/urfave/cli/v3"
)

type builtinAction interface {
	Name() string
	Flags() []cli.Flag
	LogValue() slog.Value
	gollam.ToolSet
}

type actionList []builtinAction

var actions = actionList{
	&urlscan.Action{},
	&otx.Action{},
	&vt.Action{},
	&shodan.Action{},
	&abusech.Action{},
	&ipdb.Action{},
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

func (x actionList) ToolSets() []gollam.ToolSet {
	toolSets := []gollam.ToolSet{}
	for _, action := range x {
		toolSets = append(toolSets, action)
	}
	return toolSets
}
