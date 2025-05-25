package command

import (
	"context"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/service/command/core"
	slack_svc "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/utils/ptr"
)

type Service struct {
	clients *core.Clients
}

func New(repo interfaces.Repository, llm gollem.LLMClient, thread *slack_svc.ThreadService) *Service {
	return &Service{
		clients: core.NewClients(repo, llm, thread),
	}
}

var (
	ErrUnknownCommand = goerr.New("unknown command")
)

type Command func(ctx context.Context, clients *core.Clients, msg *slack.Message, input string) error

func (x *Service) Execute(ctx context.Context, msg *slack.Message, input string) error {
	commands := map[string]Command{
		"l":         list.Create,
		"ls":        list.Create,
		"list":      list.Create,
		"a":         aggregate.Create,
		"aggr":      aggregate.Create,
		"aggregate": aggregate.Create,
		"t":         ticket.Create,
		"ticket":    ticket.Create,
	}

	cmd, remaining := messageToArgs(input)
	if cmd == "" {
		return ErrUnknownCommand
	}

	switch cmd {
	case "l", "ls", "list":
		_, err := svc.List(ctx, threadSvc, ptr.Ref(slackMsg.User()), remaining)
		if err != nil {
			return eb.Wrap(err, "failed to run list command")
		}
		return nil

	case "a", "aggr", "aggregate":
		if latestList == nil {
			msg.Notify(ctx, "🤔 No alert list found in this thread. Please create one first.")
			return eb.Wrap(errNoRequiredData, "no alert list found in thread", goerr.V("thread", slackMsg.Thread()))
		}

		if err := svc.Aggregate(ctx, threadSvc, slackMsg.User(), latestList, remaining); err != nil {
			return eb.Wrap(err, "failed to run aggregate command")
		}
		return nil

	case "t", "ticket":
		err := svc.Ticket(ctx, threadSvc, ptr.Ref(slackMsg.User()), remaining)
		if err != nil {
			return eb.Wrap(err, "failed to run ticket command")
		}
		return nil

	case "alert":
		// @TODO: Fix it
		if latestAlert == nil {
			msg.Notify(ctx, "🤔 No alert found in this thread. Please create one first.")
			return eb.Wrap(errNoRequiredData, "no alert found in thread", goerr.V("thread", slackMsg.Thread()))
		}
		msg.Notify(ctx, "🤔 Alert found in this thread. Please use `alert` command to manage the alert.")
		return nil

	default:
		return errUnknownCommand
	}

}

func messageToArgs(message string) (string, string) {
	args := strings.SplitN(message, " ", 2)
	if len(args) == 0 {
		return "", ""
	}
	if len(args) == 1 {
		return strings.ToLower(strings.TrimSpace(args[0])), ""
	}
	return strings.ToLower(strings.TrimSpace(args[0])), strings.TrimSpace(args[1])
}
