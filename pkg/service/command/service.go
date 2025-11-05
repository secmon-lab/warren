package command

import (
	"context"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/service/command/core"

	"github.com/secmon-lab/warren/pkg/service/command/repair"
	"github.com/secmon-lab/warren/pkg/service/command/ticket"
)

type Service struct {
	clients *core.Clients
}

func New(repo interfaces.Repository, llm gollem.LLMClient, thread interfaces.SlackThreadService) *Service {
	return &Service{
		clients: core.NewClients(repo, llm, thread),
	}
}

func NewWithUseCase(repo interfaces.Repository, llm gollem.LLMClient, thread interfaces.SlackThreadService, ticketUC core.TicketUseCase, slackClient interfaces.SlackClient) *Service {
	return &Service{
		clients: core.NewClientsWithSlack(repo, llm, thread, ticketUC, slackClient),
	}
}

var (
	ErrUnknownCommand = goerr.New("unknown command")
)

type Command func(ctx context.Context, clients *core.Clients, msg *slack.Message, input string) error

func (x *Service) Execute(ctx context.Context, msg *slack.Message, input string) error {
	commands := map[string]Command{
		"t":      ticket.Create,
		"ticket": ticket.Create,
		"repair": repair.Run,
		"purge":  purge,
	}

	cmd, remaining := messageToArgs(input)
	if cmd == "" {
		return ErrUnknownCommand
	}

	cmdFunc, ok := commands[cmd]
	if !ok {
		return goerr.Wrap(ErrUnknownCommand, "unknown command", goerr.V("command", cmd))
	}

	return cmdFunc(ctx, x.clients, msg, remaining)
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
