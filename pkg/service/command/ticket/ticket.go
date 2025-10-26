package ticket

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/command/core"
)

//go:embed ticket.help.md
var ticketHelp string

type ticketCommand struct {
	command string
	args    []string
}

var knownSubCommands = []string{"help", "unresolved", "u", "from", "after", "since"}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func parseTicketCommand(input string) (*ticketCommand, error) {
	if input == "help" {
		return &ticketCommand{command: "help"}, nil
	}

	commands := strings.Fields(input)
	if len(commands) == 0 {
		// No arguments - create from conversation
		return &ticketCommand{command: "create_from_conversation"}, nil
	}

	// Check if it's a known subcommand
	if contains(knownSubCommands, commands[0]) {
		return &ticketCommand{
			command: commands[0],
			args:    commands[1:],
		}, nil
	}

	// Unknown command - treat as user context for conversation-based ticket creation
	return &ticketCommand{
		command: "create_from_conversation",
		args:    []string{input}, // Whole input as context
	}, nil
}

func Create(ctx context.Context, clients *core.Clients, msg *slack.Message, input string) (any, error) {
	th := clients.Thread()

	cmd, err := parseTicketCommand(input)
	if err != nil {
		th.Reply(ctx, ticketHelp)
		return nil, goerr.Wrap(err, "failed to parse ticket command")
	}

	switch cmd.command {
	case "help":
		th.Reply(ctx, ticketHelp)
		return nil, nil

	case "unresolved", "u":
		return nil, handleUnresolvedTickets(ctx, clients, th)

	case "from":
		if len(cmd.args) < 3 || cmd.args[1] != "to" {
			th.Reply(ctx, ticketHelp)
			return nil, goerr.New("invalid time range format. expected: from <time> to <time>")
		}

		from, err := core.ParseTime(cmd.args[0])
		if err != nil {
			th.Reply(ctx, ticketHelp)
			return nil, goerr.Wrap(err, "failed to parse from time")
		}

		to, err := core.ParseTime(cmd.args[2])
		if err != nil {
			th.Reply(ctx, ticketHelp)
			return nil, goerr.Wrap(err, "failed to parse to time")
		}

		return nil, handleTicketsBySpan(ctx, clients, th, from, to)

	case "after":
		if len(cmd.args) < 1 {
			th.Reply(ctx, ticketHelp)
			return nil, goerr.New("invalid date format. expected: after <date>")
		}

		date, err := core.ParseTime(cmd.args[0])
		if err != nil {
			th.Reply(ctx, ticketHelp)
			return nil, goerr.Wrap(err, "failed to parse date")
		}

		return nil, handleTicketsBySpan(ctx, clients, th, date, time.Now())

	case "since":
		if len(cmd.args) < 1 {
			th.Reply(ctx, ticketHelp)
			return nil, goerr.New("invalid duration format. expected: since <duration>")
		}

		duration, err := core.ParseDuration(cmd.args[0])
		if err != nil {
			th.Reply(ctx, ticketHelp)
			return nil, goerr.Wrap(err, "failed to parse duration")
		}

		since := time.Now().Add(-duration)
		return nil, handleTicketsBySpan(ctx, clients, th, since, time.Now())

	case "create_from_conversation":
		userContext := ""
		if len(cmd.args) > 0 {
			userContext = cmd.args[0]
		}
		return nil, handleCreateFromConversation(ctx, clients, msg, userContext)

	default:
		th.Reply(ctx, ticketHelp)
		return nil, goerr.New(fmt.Sprintf("unknown command: %s", cmd.command))
	}
}

func handleUnresolvedTickets(ctx context.Context, clients *core.Clients, th interfaces.SlackThreadService) error {
	tickets, err := clients.Repo().GetTicketsByStatus(ctx, []types.TicketStatus{types.TicketStatusOpen}, 0, 0)
	if err != nil {
		return goerr.Wrap(err, "failed to get tickets by status")
	}

	if err := th.PostTicketList(ctx, tickets); err != nil {
		return goerr.Wrap(err, "failed to post ticket list")
	}
	return nil
}

func handleTicketsBySpan(ctx context.Context, clients *core.Clients, th interfaces.SlackThreadService, begin, end time.Time) error {
	tickets, err := clients.Repo().GetTicketsBySpan(ctx, begin, end)
	if err != nil {
		return goerr.Wrap(err, "failed to get tickets by span")
	}

	if err := th.PostTicketList(ctx, tickets); err != nil {
		return goerr.Wrap(err, "failed to post ticket list")
	}
	return nil
}

func handleCreateFromConversation(
	ctx context.Context,
	clients *core.Clients,
	msg *slack.Message,
	userContext string,
) error {
	// Check if UseCase is available
	ticketUC := clients.TicketUseCase()
	if ticketUC == nil {
		clients.Thread().Reply(ctx, "Ticket creation from conversation is not available")
		return goerr.New("ticket use case not available")
	}

	// Create ticket from conversation using UseCase
	ticket, err := ticketUC.CreateTicketFromConversation(
		ctx,
		msg.Thread(),
		msg.User(),
		userContext,
	)
	if err != nil {
		clients.Thread().Reply(ctx, fmt.Sprintf("Failed to create ticket: %v", err))
		return goerr.Wrap(err, "failed to create ticket from conversation")
	}

	// Success message is not needed as ticket posting is done by UseCase
	_ = ticket
	return nil
}
