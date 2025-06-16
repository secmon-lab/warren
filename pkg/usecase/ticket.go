package usecase

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

// TicketCreationOptions contains options for ticket creation
type TicketCreationOptions struct {
	AlertIDs     []types.AlertID
	SlackThread  *slack.Thread
	Assignee     *slack.User
	Title        string
	Description  string
	FillMetadata bool // Whether to use LLM to fill metadata
	IsTest       bool // Whether this is a test ticket
}

// createTicketWithSlackPosting creates a ticket and posts it to Slack
func (uc *UseCases) createTicketWithSlackPosting(ctx context.Context, opts TicketCreationOptions, alerts alert.Alerts) (*ticket.Ticket, error) {
	// Create new ticket
	newTicket := ticket.New(ctx, opts.AlertIDs, opts.SlackThread)
	newTicket.Assignee = opts.Assignee
	newTicket.IsTest = opts.IsTest

	// Set metadata
	if opts.Title != "" {
		newTicket.Metadata.Title = opts.Title
	}
	if opts.Description != "" {
		newTicket.Metadata.Description = opts.Description
	}

	// Fill metadata using LLM if requested
	if opts.FillMetadata {
		if err := newTicket.FillMetadata(ctx, uc.llmClient, uc.repository); err != nil {
			return nil, goerr.Wrap(err, "failed to fill ticket metadata")
		}
	}

	// Calculate embedding using the unified approach
	if err := newTicket.CalculateEmbedding(ctx, uc.llmClient, uc.repository); err != nil {
		return nil, goerr.Wrap(err, "failed to calculate ticket embedding")
	}

	// Post to Slack if SlackThread is provided
	if opts.SlackThread != nil {
		messageID, err := uc.postTicketToSlack(ctx, &newTicket, *opts.SlackThread, alerts)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to post ticket to slack")
		}
		newTicket.SlackMessageID = messageID
	}

	// Save ticket to repository
	if err := uc.repository.PutTicket(ctx, newTicket); err != nil {
		return nil, goerr.Wrap(err, "failed to save ticket")
	}

	return &newTicket, nil
}

// postTicketToSlack handles Slack posting logic including thread management
func (uc *UseCases) postTicketToSlack(ctx context.Context, newTicket *ticket.Ticket, slackThread slack.Thread, alerts alert.Alerts) (string, error) {
	st := uc.slackService.NewThread(slackThread)
	var timestamp string
	var threadService interface {
		PostComment(context.Context, string) error
	} // To track which service to use for posting comments

	// Check if there are multiple alert lists in the thread (only for alert-based tickets)
	if len(alerts) > 0 {
		alertLists, err := uc.repository.GetAlertListsInThread(ctx, slackThread)
		if err != nil {
			return "", goerr.Wrap(err, "failed to get alert lists in thread")
		}

		if len(alertLists) > 1 {
			// Multiple alert lists exist, post ticket to new thread
			newThreadSvc, ts, err := uc.slackService.PostTicket(ctx, *newTicket, alerts)
			if err != nil {
				return "", goerr.Wrap(err, "failed to post ticket to new thread")
			}
			timestamp = ts
			threadService = newThreadSvc

			// Update ticket's slack thread to the new thread
			newTicket.SlackThread = &slack.Thread{
				ChannelID: newThreadSvc.ChannelID(),
				ThreadID:  newThreadSvc.ThreadID(),
			}

			// Post link to the new ticket in the original thread
			ticketURL := uc.slackService.ToMsgURL(newThreadSvc.ChannelID(), newThreadSvc.ThreadID())
			if err := st.PostLinkToTicket(ctx, ticketURL, newTicket.Metadata.Title); err != nil {
				return "", goerr.Wrap(err, "failed to post link to ticket")
			}
		} else {
			// Single or no alert list - post ticket in the current thread
			ts, err := st.PostTicket(ctx, *newTicket, alerts)
			if err != nil {
				return "", goerr.Wrap(err, "failed to post ticket")
			}
			timestamp = ts
			threadService = st
		}
	} else {
		// Manual ticket - post ticket in the current thread
		ts, err := st.PostTicket(ctx, *newTicket, alerts)
		if err != nil {
			return "", goerr.Wrap(err, "failed to post ticket")
		}
		timestamp = ts
		threadService = st
	}

	// Generate and post initial comment for all tickets (regardless of alerts)
	if comment, err := uc.generateInitialTicketComment(ctx, newTicket, alerts); err != nil {
		_ = msg.Trace(ctx, "💥 Failed to generate initial comment: %s", err.Error())
	} else if comment != "" {
		if err := threadService.PostComment(ctx, comment); err != nil {
			_ = msg.Trace(ctx, "💥 Failed to post initial comment: %s", err.Error())
		}
	}

	return timestamp, nil
}

// CreateManualTicket creates a ticket manually without associated alerts
func (uc *UseCases) CreateManualTicket(ctx context.Context, title, description string, user *slack.User) (*ticket.Ticket, error) {
	return uc.CreateManualTicketWithTest(ctx, title, description, user, false)
}

// CreateManualTicketWithTest creates a ticket manually without associated alerts with test flag
func (uc *UseCases) CreateManualTicketWithTest(ctx context.Context, title, description string, user *slack.User, isTest bool) (*ticket.Ticket, error) {
	// Validate required fields
	if title == "" {
		return nil, goerr.New("title is required")
	}

	// Create ticket using common helper
	opts := TicketCreationOptions{
		AlertIDs:     []types.AlertID{},
		SlackThread:  nil, // Manual tickets don't have Slack threads initially
		Assignee:     user,
		Title:        title,
		Description:  description,
		FillMetadata: false, // Manual tickets don't use LLM to fill metadata
		IsTest:       isTest,
	}

	return uc.createTicketWithSlackPosting(ctx, opts, alert.Alerts{})
}

// UpdateTicket updates a ticket's title and description
func (uc *UseCases) UpdateTicket(ctx context.Context, ticketID types.TicketID, title, description string, user *slack.User) (*ticket.Ticket, error) {
	// Validate required fields
	if title == "" {
		return nil, goerr.New("title is required")
	}

	// Get existing ticket
	existingTicket, err := uc.repository.GetTicket(ctx, ticketID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get ticket")
	}
	if existingTicket == nil {
		return nil, goerr.New("ticket not found", goerr.V("ticket_id", ticketID))
	}

	// Update metadata
	existingTicket.Metadata.Title = title
	existingTicket.Metadata.Description = description
	existingTicket.UpdatedAt = clock.Now(ctx)

	// Recalculate embedding since title/description changed
	if err := existingTicket.CalculateEmbedding(ctx, uc.llmClient, uc.repository); err != nil {
		return nil, goerr.Wrap(err, "failed to recalculate ticket embedding")
	}

	// Save updated ticket
	if err := uc.repository.PutTicket(ctx, *existingTicket); err != nil {
		return nil, goerr.Wrap(err, "failed to save updated ticket")
	}

	// Update Slack post if ticket has a Slack thread
	if existingTicket.SlackThread != nil {
		// Get associated alerts for Slack update
		alerts, err := uc.repository.BatchGetAlerts(ctx, existingTicket.AlertIDs)
		if err != nil {
			// Log error but don't fail the update
			_ = msg.Trace(ctx, "💥 Failed to get alerts for Slack update: %s", err.Error())
		} else {
			st := uc.slackService.NewThread(*existingTicket.SlackThread)
			if _, err := st.PostTicket(ctx, *existingTicket, alerts); err != nil {
				// Log error but don't fail the update
				_ = msg.Trace(ctx, "💥 Failed to update Slack post: %s", err.Error())
			}
		}
	}

	return existingTicket, nil
}
