package command_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"

	mock "github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/command"
	slacksvc "github.com/secmon-lab/warren/pkg/service/slack"
)

func setupTicketTestService(t *testing.T) (*command.Service, *mock.RepositoryMock, *slacksvc.ThreadService, *slack.User, []*ticket.Ticket) {
	ctx := context.Background()
	repo := &mock.RepositoryMock{}
	cmdSvc := command.New(repo, nil)
	slackService := slacksvc.NewTestService(t)
	threadService, err := slackService.PostMessage(ctx, "test message")
	gt.NoError(t, err).Required()
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
			Status:    types.TicketStatusInvestigating,
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
			Status:    types.TicketStatusInvestigating,
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

	repo.GetTicketsByStatusFunc = func(ctx context.Context, status types.TicketStatus) ([]*ticket.Ticket, error) {
		var result []*ticket.Ticket
		for _, t := range tickets {
			if t.Status == status {
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
	svc, _, threadService, user, _ := setupTicketTestService(t)
	ctx := context.Background()

	t.Run("show help message", func(t *testing.T) {
		err := svc.Ticket(ctx, threadService, user, "help")
		gt.NoError(t, err)
	})

	t.Run("show unresolved tickets by default", func(t *testing.T) {
		err := svc.Ticket(ctx, threadService, user, "")
		gt.NoError(t, err)
	})

	t.Run("show unresolved tickets with explicit command", func(t *testing.T) {
		err := svc.Ticket(ctx, threadService, user, "unresolved")
		gt.NoError(t, err)
	})

	t.Run("show unresolved tickets with alias", func(t *testing.T) {
		err := svc.Ticket(ctx, threadService, user, "u")
		gt.NoError(t, err)
	})

	t.Run("show tickets with time range", func(t *testing.T) {
		now := time.Now()
		from := now.Add(-2 * time.Hour).Format("15:04")
		to := now.Add(-1 * time.Hour).Format("15:04")

		err := svc.Ticket(ctx, threadService, user, "from "+from+" to "+to)
		gt.NoError(t, err)
	})

	t.Run("show tickets after date", func(t *testing.T) {
		date := time.Now().Add(-2 * time.Hour).Format("2006-01-02")
		err := svc.Ticket(ctx, threadService, user, "after "+date)
		gt.NoError(t, err)
	})

	t.Run("show tickets since duration", func(t *testing.T) {
		err := svc.Ticket(ctx, threadService, user, "since 2h")
		gt.NoError(t, err)
	})

	t.Run("error on invalid time range format", func(t *testing.T) {
		err := svc.Ticket(ctx, threadService, user, "from 10:00")
		gt.Error(t, err)
	})

	t.Run("error on invalid date format", func(t *testing.T) {
		err := svc.Ticket(ctx, threadService, user, "after invalid-date")
		gt.Error(t, err)
	})

	t.Run("error on invalid duration format", func(t *testing.T) {
		err := svc.Ticket(ctx, threadService, user, "since invalid-duration")
		gt.Error(t, err)
	})

	t.Run("error on unknown command", func(t *testing.T) {
		err := svc.Ticket(ctx, threadService, user, "unknown-command")
		gt.Error(t, err)
	})
}
