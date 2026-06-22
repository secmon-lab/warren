package cli

import (
	"context"
	"log/slog"

	"github.com/gollem-dev/gollem"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/tool/abusech"
	"github.com/secmon-lab/warren/pkg/tool/bigquery"
	"github.com/secmon-lab/warren/pkg/tool/falcon"
	"github.com/secmon-lab/warren/pkg/tool/github"
	"github.com/secmon-lab/warren/pkg/tool/intune"
	"github.com/secmon-lab/warren/pkg/tool/ipdb"
	"github.com/secmon-lab/warren/pkg/tool/jira"
	"github.com/secmon-lab/warren/pkg/tool/otx"
	"github.com/secmon-lab/warren/pkg/tool/shodan"
	"github.com/secmon-lab/warren/pkg/tool/slack"
	"github.com/secmon-lab/warren/pkg/tool/urlscan"
	"github.com/secmon-lab/warren/pkg/tool/vt"
	"github.com/secmon-lab/warren/pkg/tool/webfetch"
	"github.com/secmon-lab/warren/pkg/tool/whois"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
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
	&whois.Action{},
	&intune.Action{},
	&webfetch.Action{},
	&falcon.Action{},
	&jira.Action{},
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

// InjectLLMClient injects the warren-wide LLM client into tools that support
// it. Tools that own their own LLM lifecycle (e.g. webfetch, which builds and
// pings its client from its own --webfetch-llm-* flags) do not implement the
// SetLLMClient setter and are skipped by the duck-typed loop below.
func (x toolList) InjectLLMClient(llmClient gollem.LLMClient) {
	for _, tool := range x {
		if setter, ok := tool.(interface{ SetLLMClient(gollem.LLMClient) }); ok {
			setter.SetLLMClient(llmClient)
		}
	}
}

// HITLToolNames returns the names of tool functions (Specs entries) that
// require human-in-the-loop approval, based on each tool's RequiresHITL()
// state. Tools that do not implement the HITL-aware interface are excluded.
//
// Configure() MUST have been called on the tool list before this — for
// flag-driven gating like webfetch's --webfetch-llm-provider, the dynamic
// state needs to be resolved first.
func (x toolList) HITLToolNames(ctx context.Context) ([]string, error) {
	var names []string
	for _, tool := range x {
		aware, ok := tool.(interface{ RequiresHITL() bool })
		if !ok || !aware.RequiresHITL() {
			continue
		}
		specs, err := tool.Specs(ctx)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to read tool specs for HITL list",
				goerr.V("tool", tool.ID()))
		}
		for _, s := range specs {
			names = append(names, s.Name)
		}
	}
	return names, nil
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
		attrs = append(attrs, slog.Any(tool.ID(), tool.LogValue()))
	}
	return slog.GroupValue(attrs...)
}

func (x toolList) ToolSets(ctx context.Context) ([]interfaces.ToolSet, error) {
	toolSets := []interfaces.ToolSet{}
	for _, tool := range x {
		if err := tool.Configure(ctx); err != nil {
			if err == errutil.ErrActionUnavailable {
				continue
			}
			return nil, err
		}
		toolSets = append(toolSets, tool)
	}
	return toolSets, nil
}
