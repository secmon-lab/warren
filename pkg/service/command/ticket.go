package command

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	slackmodel "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	slacksvc "github.com/secmon-lab/warren/pkg/service/slack"
)

//go:embed help/ticket.md
var ticketHelp string

type ticketCommand struct {
	command string
	args    []string
}

func parseTicketCommand(input string) (*ticketCommand, error) {
	if input == "help" {
		return &ticketCommand{command: "help"}, nil
	}

	commands := strings.Fields(input)
	if len(commands) == 0 {
		return &ticketCommand{command: "unresolved"}, nil
	}

	return &ticketCommand{
		command: commands[0],
		args:    commands[1:],
	}, nil
}

func (x *Service) Ticket(ctx context.Context, th *slacksvc.ThreadService, user *slackmodel.User, input string) error {
	cmd, err := parseTicketCommand(input)
	if err != nil {
		th.Reply(ctx, ticketHelp)
		return goerr.Wrap(err, "failed to parse ticket command")
	}

	switch cmd.command {
	case "help":
		th.Reply(ctx, ticketHelp)
		return nil

	case "unresolved", "u":
		return x.handleUnresolvedTickets(ctx, th)

	case "from":
		if len(cmd.args) < 3 || cmd.args[1] != "to" {
			th.Reply(ctx, ticketHelp)
			return goerr.New("invalid time range format. expected: from <time> to <time>")
		}

		from, err := ParseTime(cmd.args[0])
		if err != nil {
			th.Reply(ctx, ticketHelp)
			return goerr.Wrap(err, "failed to parse from time")
		}

		to, err := ParseTime(cmd.args[2])
		if err != nil {
			th.Reply(ctx, ticketHelp)
			return goerr.Wrap(err, "failed to parse to time")
		}

		return x.handleTicketsBySpan(ctx, th, from, to)

	case "after":
		if len(cmd.args) < 1 {
			th.Reply(ctx, ticketHelp)
			return goerr.New("invalid date format. expected: after <date>")
		}

		date, err := ParseTime(cmd.args[0])
		if err != nil {
			th.Reply(ctx, ticketHelp)
			return goerr.Wrap(err, "failed to parse date")
		}

		return x.handleTicketsBySpan(ctx, th, date, time.Now())

	case "since":
		if len(cmd.args) < 1 {
			th.Reply(ctx, ticketHelp)
			return goerr.New("invalid duration format. expected: since <duration>")
		}

		duration, err := ParseDuration(cmd.args[0])
		if err != nil {
			th.Reply(ctx, ticketHelp)
			return goerr.Wrap(err, "failed to parse duration")
		}

		since := time.Now().Add(-duration)
		return x.handleTicketsBySpan(ctx, th, since, time.Now())

	default:
		th.Reply(ctx, ticketHelp)
		return goerr.New(fmt.Sprintf("unknown command: %s", cmd.command))
	}
}

func (x *Service) handleUnresolvedTickets(ctx context.Context, th *slacksvc.ThreadService) error {
	tickets, err := x.repo.GetTicketsByStatus(ctx, types.TicketStatusInvestigating)
	if err != nil {
		return goerr.Wrap(err, "failed to get tickets by status")
	}

	if err := th.PostTicketList(ctx, tickets); err != nil {
		return goerr.Wrap(err, "failed to post ticket list")
	}
	return nil
}

func (x *Service) handleTicketsBySpan(ctx context.Context, th *slacksvc.ThreadService, begin, end time.Time) error {
	tickets, err := x.repo.GetTicketsBySpan(ctx, begin, end)
	if err != nil {
		return goerr.Wrap(err, "failed to get tickets by span")
	}

	if err := th.PostTicketList(ctx, tickets); err != nil {
		return goerr.Wrap(err, "failed to post ticket list")
	}
	return nil
}
