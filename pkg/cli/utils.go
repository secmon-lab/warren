package cli

import (
	"context"
	"log/slog"

	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/tool/abusech"
	"github.com/secmon-lab/warren/pkg/tool/bigquery"
	"github.com/secmon-lab/warren/pkg/tool/github"
	"github.com/secmon-lab/warren/pkg/tool/ipdb"
	"github.com/secmon-lab/warren/pkg/tool/otx"
	"github.com/secmon-lab/warren/pkg/tool/shodan"
	"github.com/secmon-lab/warren/pkg/tool/slack"
	"github.com/secmon-lab/warren/pkg/tool/urlscan"
	"github.com/secmon-lab/warren/pkg/tool/vt"
	"github.com/urfave/cli/v3"
)

func joinFlags(flags ...[]cli.Flag) []cli.Flag {
	var result []cli.Flag
	for _, flag := range flags {
		result = append(result, flag...)
	}
	return result
}

type toolList []interfaces.Tool

var tools = toolList{
	// All external tools restored with gemini-2.5-flash
	&otx.Action{},
	&urlscan.Action{},
	&vt.Action{},
	&shodan.Action{},
	&abusech.Action{},
	&ipdb.Action{},
	&bigquery.Action{},
	&slack.Action{},
	&github.Action{},
}

// InjectDependencies injects repository and embedding client into tools that support them
func (x *toolList) InjectDependencies(repo interfaces.Repository, embeddingClient interfaces.EmbeddingClient) {
	// NOTE: knowledge tool is not added here anymore.
	// It is created per chat session with the appropriate topic in usecase/chat.go

	for _, tool := range *x {
		// Check if tool supports repository injection
		if repoSetter, ok := tool.(interface{ SetRepository(interfaces.Repository) }); ok {
			repoSetter.SetRepository(repo)
		}
		// Check if tool supports embedding client injection
		if embeddingSetter, ok := tool.(interface {
			SetEmbeddingClient(interfaces.EmbeddingClient)
		}); ok {
			embeddingSetter.SetEmbeddingClient(embeddingClient)
		}
	}
}

// InjectSlackClient injects slack client into tools that support it
func (x toolList) InjectSlackClient(slackClient interfaces.SlackClient) {
	for _, tool := range x {
		// Check if tool supports slack client injection
		if slackSetter, ok := tool.(interface{ SetSlackClient(interfaces.SlackClient) }); ok {
			slackSetter.SetSlackClient(slackClient)
		}
	}
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
