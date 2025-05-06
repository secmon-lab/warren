package cli

import (
	"context"
	"log/slog"

	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/tool/abusech"
	"github.com/secmon-lab/warren/pkg/tool/ipdb"
	"github.com/secmon-lab/warren/pkg/tool/otx"
	"github.com/secmon-lab/warren/pkg/tool/shodan"
	"github.com/secmon-lab/warren/pkg/tool/urlscan"
	"github.com/secmon-lab/warren/pkg/tool/vt"
	"github.com/urfave/cli/v3"
)

type builtinTool interface {
	Name() string
	Flags() []cli.Flag
	Configure(ctx context.Context) error
	LogValue() slog.Value
	gollem.ToolSet
}

type toolList []builtinTool

var tools = toolList{
	&urlscan.Action{},
	&otx.Action{},
	&vt.Action{},
	&shodan.Action{},
	&abusech.Action{},
	&ipdb.Action{},
	// &bigquery.Action{},
}

func (x toolList) Flags() []cli.Flag {
	flags := []cli.Flag{}
	for _, tool := range x {
		flags = append(flags, tool.Flags()...)
	}
	return flags
}

func (x toolList) LogValue() slog.Value {
	var attrs []slog.Attr
	for _, tool := range x {
		attrs = append(attrs, slog.Any(tool.Name(), tool.LogValue()))
	}
	return slog.GroupValue(attrs...)
}

func (x toolList) ToolSets(ctx context.Context) ([]gollem.ToolSet, error) {
	toolSets := []gollem.ToolSet{}
	for _, tool := range x {
		if err := tool.Configure(ctx); err != nil {
			if err == errs.ErrActionUnavailable {
				continue
			}
			return nil, err
		}
		toolSets = append(toolSets, tool)
	}
	return toolSets, nil
}
