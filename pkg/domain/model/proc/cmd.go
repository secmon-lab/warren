package proc

import (
	"bytes"
	"context"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

type Proc struct {
	ID      types.ProcID
	Args    []string
	clients clients
}

type clients struct {
	repo      interfaces.Repository
	llmClient interfaces.LLMClient
	slack     slack.ThreadService
}

func New(repo interfaces.Repository, llmClient interfaces.LLMClient, slackThread slack.ThreadService, args []string) *Proc {

	return &Proc{
		ID:   types.NewProcID(),
		Args: args,
		clients: clients{
			repo:      repo,
			llmClient: llmClient,
			slack:     slackThread,
		},
	}
}

func (x *Proc) Run(ctx context.Context) error {
	var buf bytes.Buffer
	cmd := cli.Command{
		Name:     "warren",
		Usage:    "Slack bot for security monitoring",
		Commands: []*cli.Command{},
		Writer:   &buf,
	}

	err := cmd.Run(ctx, x.Args)
	if err != nil {
		logging.From(ctx).Error("Failed to run command", "error", err)
	}

	if buf.String() != "" {
		x.clients.slack.Reply(ctx, "```"+buf.String()+"```")
	}

	return nil
}
