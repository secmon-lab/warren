package ticket_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/m-mizutani/gt"

	mock "github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/command"
	"github.com/secmon-lab/warren/pkg/service/command/core"
	ticketcmd "github.com/secmon-lab/warren/pkg/service/command/ticket"
	slacksvc "github.com/secmon-lab/warren/pkg/service/slack"
)

func setupTicketTestService(t *testing.T) (*command.Service, *mock.RepositoryMock, *slacksvc.ThreadService, *slack.User, []*ticket.Ticket) {
	ctx := context.Background()
	repo := &mock.RepositoryMock{}
	slackService := slacksvc.NewTestService(t)
	threadService, err := slackService.PostMessage(ctx, "test message")
	gt.NoError(t, err).Required()
	cmdSvc := command.New(repo, nil, threadService)
	user := &slack.User{
		ID:   "U0123456789",
		Name: "Test User",
	}

	fixedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	tickets := []*ticket.Ticket{
		{
			ID: types.NewTicketID(),
			Metadata: ticket.Metadata{
				Title:       "Ticket 1",
				Description: "Test ticket 1",
			},
			Status:    types.TicketStatusOpen,
			CreatedAt: fixedTime.Add(-1 * time.Hour),
			SlackThread: &slack.Thread{
				ChannelID: "C0123456789",
				ThreadID:  "T0123456789",
			},
		},
		{
			ID: types.NewTicketID(),
			Metadata: ticket.Metadata{
				Title:       "Ticket 2",
				Description: "Test ticket 2",
			},
			Status:    types.TicketStatusOpen,
			CreatedAt: fixedTime.Add(-2 * time.Hour),
			SlackThread: &slack.Thread{
				ChannelID: "C0123456789",
				ThreadID:  "T0123456789",
			},
		},
		{
			ID: types.NewTicketID(),
			Metadata: ticket.Metadata{
				Title:       "Ticket 3",
				Description: "Test ticket 3",
			},
			Status:    types.TicketStatusResolved,
			CreatedAt: fixedTime.Add(-3 * time.Hour),
			SlackThread: &slack.Thread{
				ChannelID: "C0123456789",
				ThreadID:  "T0123456789",
			},
		},
	}

	repo.GetTicketsByStatusFunc = func(ctx context.Context, statuses []types.TicketStatus, offset, limit int) ([]*ticket.Ticket, error) {
		var result []*ticket.Ticket
		for _, t := range tickets {
			if slices.Contains(statuses, t.Status) {
				result = append(result, t)
			}
		}
		return result, nil
	}

	repo.GetTicketsBySpanFunc = func(ctx context.Context, begin, end time.Time) ([]*ticket.Ticket, error) {
		var result []*ticket.Ticket
		for _, t := range tickets {
			if t.CreatedAt.After(begin) && t.CreatedAt.Before(end) {
				result = append(result, t)
			}
		}
		return result, nil
	}

	return cmdSvc, repo, threadService, user, tickets
}

func TestService_Ticket(t *testing.T) {
	_, repo, threadService, user, _ := setupTicketTestService(t)
	ctx := context.Background()

	// Create core.Clients directly
	clients := core.NewClients(repo, nil, threadService)

	// Create a mock slack message
	createSlackMessage := func(user *slack.User) *slack.Message {
		return nil
	}

	t.Run("show help message", func(t *testing.T) {
		_, err := ticketcmd.Create(ctx, clients, createSlackMessage(user), "help")
		gt.NoError(t, err)
	})

	t.Run("show unresolved tickets by default", func(t *testing.T) {
		_, err := ticketcmd.Create(ctx, clients, createSlackMessage(user), "")
		gt.NoError(t, err)
	})

	t.Run("show unresolved tickets with explicit command", func(t *testing.T) {
		_, err := ticketcmd.Create(ctx, clients, createSlackMessage(user), "unresolved")
		gt.NoError(t, err)
	})

	t.Run("show unresolved tickets with alias", func(t *testing.T) {
		_, err := ticketcmd.Create(ctx, clients, createSlackMessage(user), "u")
		gt.NoError(t, err)
	})

	t.Run("show tickets with time range", func(t *testing.T) {
		now := time.Now()
		from := now.Add(-2 * time.Hour).Format("15:04")
		to := now.Add(-1 * time.Hour).Format("15:04")

		_, err := ticketcmd.Create(ctx, clients, createSlackMessage(user), "from "+from+" to "+to)
		gt.NoError(t, err)
	})

	t.Run("show tickets after date", func(t *testing.T) {
		date := time.Now().Add(-2 * time.Hour).Format("2006-01-02")
		_, err := ticketcmd.Create(ctx, clients, createSlackMessage(user), "after "+date)
		gt.NoError(t, err)
	})

	t.Run("show tickets since duration", func(t *testing.T) {
		_, err := ticketcmd.Create(ctx, clients, createSlackMessage(user), "since 2h")
		gt.NoError(t, err)
	})

	t.Run("error on invalid time range format", func(t *testing.T) {
		_, err := ticketcmd.Create(ctx, clients, createSlackMessage(user), "from 10:00")
		gt.Error(t, err)
	})

	t.Run("error on invalid date format", func(t *testing.T) {
		_, err := ticketcmd.Create(ctx, clients, createSlackMessage(user), "after invalid-date")
		gt.Error(t, err)
	})

	t.Run("error on invalid duration format", func(t *testing.T) {
		_, err := ticketcmd.Create(ctx, clients, createSlackMessage(user), "since invalid-duration")
		gt.Error(t, err)
	})

	t.Run("error on unknown command", func(t *testing.T) {
		_, err := ticketcmd.Create(ctx, clients, createSlackMessage(user), "unknown-command")
		gt.Error(t, err)
	})
}
