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

	// Create core.Clients without TicketUseCase
	clients := core.NewClients(repo, nil, threadService)

	// Create a mock slack message
	createSlackMessage := func(user *slack.User) *slack.Message {
		return nil
	}

	t.Run("shows help without ticket use case", func(t *testing.T) {
		_, err := ticketcmd.Create(ctx, clients, createSlackMessage(user), "help")
		gt.NoError(t, err)
	})

	t.Run("returns error when ticket use case not available", func(t *testing.T) {
		_, err := ticketcmd.Create(ctx, clients, createSlackMessage(user), "")
		gt.Error(t, err)
	})
}
