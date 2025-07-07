package usecase

import (
	"context"
	"math"

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

// TicketUpdateFunction defines a function that updates a ticket
type TicketUpdateFunction func(ctx context.Context, ticket *ticket.Ticket) error

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
		return nil, goerr.Wrap(err, "failed to put new ticket")
	}

	return &newTicket, nil
}

// postTicketToSlack handles Slack posting logic including thread management
func (uc *UseCases) postTicketToSlack(ctx context.Context, newTicket *ticket.Ticket, slackThread slack.Thread, alerts alert.Alerts) (string, error) {
	var timestamp string
	var threadService interface {
		PostComment(context.Context, string) error
	} // To track which service to use for posting comments

	// Check if ThreadID is empty, indicating we should create a new thread
	if slackThread.ThreadID == "" {
		// Create new thread by posting ticket to service directly
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
	} else {
		// Use existing thread
		st := uc.slackService.NewThread(slackThread)

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
			// Manual ticket - check if there's already a ticket in this thread
			existingTicket, err := uc.repository.GetTicketByThread(ctx, slackThread)
			if err != nil {
				return "", goerr.Wrap(err, "failed to check for existing ticket in thread")
			}

			if existingTicket != nil {
				// Ticket already exists in this thread, post new ticket to separate thread
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
				// No existing ticket - post ticket in the current thread
				ts, err := st.PostTicket(ctx, *newTicket, alerts)
				if err != nil {
					return "", goerr.Wrap(err, "failed to post ticket")
				}
				timestamp = ts
				threadService = st
			}
		}
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

	var slackThread *slack.Thread
	// If Slack service is available, post to default channel
	if uc.slackService != nil {
		// Use a placeholder thread that will trigger posting to new thread
		slackThread = &slack.Thread{
			ChannelID: uc.slackService.DefaultChannelID(),
			ThreadID:  "", // Empty thread ID will create a new thread
		}
	}

	// Create ticket using common helper
	opts := TicketCreationOptions{
		AlertIDs:     []types.AlertID{},
		SlackThread:  slackThread,
		Assignee:     user,
		Title:        title,
		Description:  description,
		FillMetadata: false, // Manual tickets don't use LLM to fill metadata
		IsTest:       isTest,
	}

	return uc.createTicketWithSlackPosting(ctx, opts, alert.Alerts{})
}

// updateTicketWithSlackSync is a common helper function for updating tickets with Slack synchronization
func (uc *UseCases) updateTicketWithSlackSync(ctx context.Context, ticketID types.TicketID, updateFunc TicketUpdateFunction) (*ticket.Ticket, error) {
	// Get existing ticket
	existingTicket, err := uc.repository.GetTicket(ctx, ticketID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get ticket")
	}
	if existingTicket == nil {
		return nil, goerr.New("ticket not found", goerr.V("ticket_id", ticketID))
	}

	// Apply the update function
	if err := updateFunc(ctx, existingTicket); err != nil {
		return nil, err
	}

	// Update timestamp
	existingTicket.UpdatedAt = clock.Now(ctx)

	// Save updated ticket
	if err := uc.repository.PutTicket(ctx, *existingTicket); err != nil {
		return nil, goerr.Wrap(err, "failed to save updated ticket")
	}

	// Update Slack post if ticket has a Slack thread
	if err := uc.syncTicketToSlack(ctx, existingTicket); err != nil {
		// Log error but don't fail the update
		_ = msg.Trace(ctx, "💥 Failed to sync ticket to Slack: %s", err.Error())
	}

	return existingTicket, nil
}

// syncTicketToSlack syncs a single ticket to Slack
func (uc *UseCases) syncTicketToSlack(ctx context.Context, ticket *ticket.Ticket) error {
	if ticket.SlackThread == nil || uc.slackService == nil {
		return nil // No Slack thread or service, skip sync
	}

	// Get associated alerts for Slack update
	alerts, err := uc.repository.BatchGetAlerts(ctx, ticket.AlertIDs)
	if err != nil {
		return goerr.Wrap(err, "failed to get alerts for Slack update")
	}

	st := uc.slackService.NewThread(*ticket.SlackThread)
	if _, err := st.PostTicket(ctx, *ticket, alerts); err != nil {
		return goerr.Wrap(err, "failed to update Slack post")
	}

	return nil
}

// UpdateTicket updates a ticket's title and description
func (uc *UseCases) UpdateTicket(ctx context.Context, ticketID types.TicketID, title, description string, user *slack.User) (*ticket.Ticket, error) {
	// Validate required fields
	if title == "" {
		return nil, goerr.New("title is required")
	}

	updateFunc := func(ctx context.Context, ticket *ticket.Ticket) error {
		// Update metadata
		ticket.Metadata.Title = title
		ticket.Metadata.Description = description

		// Recalculate embedding since title/description changed
		if err := ticket.CalculateEmbedding(ctx, uc.llmClient, uc.repository); err != nil {
			return goerr.Wrap(err, "failed to recalculate ticket embedding")
		}

		return nil
	}

	return uc.updateTicketWithSlackSync(ctx, ticketID, updateFunc)
}

// UpdateTicketStatus updates a ticket's status
func (uc *UseCases) UpdateTicketStatus(ctx context.Context, ticketID types.TicketID, status types.TicketStatus) (*ticket.Ticket, error) {
	// Use batch update to ensure proper activity creation
	if err := uc.repository.BatchUpdateTicketsStatus(ctx, []types.TicketID{ticketID}, status); err != nil {
		return nil, goerr.Wrap(err, "failed to update ticket status")
	}

	// Get the updated ticket
	updatedTicket, err := uc.repository.GetTicket(ctx, ticketID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get updated ticket")
	}

	// Trace ticket status change
	_ = msg.Trace(ctx, "🎫 Ticket status updated: %s",
		status)

	// Update Slack post if ticket has a Slack thread
	if err := uc.syncTicketToSlack(ctx, updatedTicket); err != nil {
		// Log error but don't fail the update
		_ = msg.Trace(ctx, "💥 Failed to sync ticket to Slack (ticket %s): %s", updatedTicket.ID, err.Error())
	}

	return updatedTicket, nil
}

// UpdateTicketConclusion updates a ticket's conclusion and reason
func (uc *UseCases) UpdateTicketConclusion(ctx context.Context, ticketID types.TicketID, conclusion types.AlertConclusion, reason string) (*ticket.Ticket, error) {
	updateFunc := func(ctx context.Context, ticket *ticket.Ticket) error {
		// Only allow updating conclusion for resolved tickets
		if ticket.Status != types.TicketStatusResolved {
			return goerr.New("can only update conclusion for resolved tickets",
				goerr.V("ticket_id", ticketID),
				goerr.V("current_status", ticket.Status))
		}

		// Update conclusion and reason
		ticket.Conclusion = conclusion
		ticket.Reason = reason
		return nil
	}

	return uc.updateTicketWithSlackSync(ctx, ticketID, updateFunc)
}

// UpdateMultipleTicketsStatus updates multiple tickets' status
func (uc *UseCases) UpdateMultipleTicketsStatus(ctx context.Context, ticketIDs []types.TicketID, status types.TicketStatus) ([]*ticket.Ticket, error) {
	// Batch update status in repository
	if err := uc.repository.BatchUpdateTicketsStatus(ctx, ticketIDs, status); err != nil {
		return nil, goerr.Wrap(err, "failed to batch update tickets status")
	}

	// Retrieve updated tickets
	tickets, err := uc.repository.BatchGetTickets(ctx, ticketIDs)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get updated tickets")
	}

	// Trace batch status update
	_ = msg.Trace(ctx, "🎫 Batch updated %d tickets to status %s", len(tickets), status)

	// Update Slack posts for tickets that have Slack threads
	for _, t := range tickets {
		if err := uc.syncTicketToSlack(ctx, t); err != nil {
			// Log error but don't fail the update
			_ = msg.Trace(ctx, "💥 Failed to sync ticket to Slack (ticket %s): %s", t.ID, err.Error())
		}
	}

	return tickets, nil
}

// CreateTicketFromAlerts creates a ticket from one or more alerts (used by both Slack and Web UI)
func (uc *UseCases) CreateTicketFromAlerts(ctx context.Context, alertIDs []types.AlertID, user *slack.User, slackThread *slack.Thread) (*ticket.Ticket, error) {
	if len(alertIDs) == 0 {
		return nil, goerr.New("no alerts provided")
	}

	// Get all alerts to validate they exist and are unbound
	alerts, err := uc.repository.BatchGetAlerts(ctx, alertIDs)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get alerts", goerr.V("alert_ids", alertIDs))
	}

	if len(alerts) != len(alertIDs) {
		return nil, goerr.New("some alerts not found", goerr.V("requested", len(alertIDs)), goerr.V("found", len(alerts)))
	}

	// Check if any alerts are already bound to tickets
	for _, alert := range alerts {
		if alert.TicketID != types.EmptyTicketID {
			return nil, goerr.New("alert is already bound to a ticket",
				goerr.V("alert_id", alert.ID),
				goerr.V("ticket_id", alert.TicketID))
		}
	}

	// For single alert with no explicit slackThread, use the alert's Slack thread if available
	if len(alerts) == 1 && slackThread == nil && alerts[0].SlackThread != nil && uc.slackService != nil {
		slackThread = alerts[0].SlackThread
	}

	// Create ticket
	newTicket := ticket.New(ctx, alertIDs, slackThread)
	newTicket.Assignee = user
	newTicket.IsTest = false

	// Fill metadata using LLM
	if err := newTicket.FillMetadata(ctx, uc.llmClient, uc.repository); err != nil {
		return nil, goerr.Wrap(err, "failed to fill ticket metadata")
	}

	// Calculate embedding
	if err := newTicket.CalculateEmbedding(ctx, uc.llmClient, uc.repository); err != nil {
		return nil, goerr.Wrap(err, "failed to calculate ticket embedding")
	}

	// Save ticket to repository
	if err := uc.repository.PutTicket(ctx, newTicket); err != nil {
		return nil, goerr.Wrap(err, "failed to put new ticket")
	}

	newTicketPtr := &newTicket

	// Update alerts to link them to the ticket
	for _, alert := range alerts {
		alert.TicketID = newTicketPtr.ID
	}

	if err := uc.repository.BatchPutAlerts(ctx, alerts); err != nil {
		return nil, goerr.Wrap(err, "failed to update alerts with ticket ID")
	}

	// Post ticket to Slack if SlackThread is provided (for Slack-originated requests)
	if slackThread != nil && uc.slackService != nil {
		messageID, err := uc.postTicketToSlack(ctx, newTicketPtr, *slackThread, alerts)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to post ticket to slack")
		}
		newTicketPtr.SlackMessageID = messageID

		// Save ticket again with Slack message ID
		if err := uc.repository.PutTicket(ctx, *newTicketPtr); err != nil {
			return nil, goerr.Wrap(err, "failed to update ticket with slack message ID")
		}
	}

	// Update individual alerts' Slack display (for all cases)
	if uc.slackService != nil {
		for _, alert := range alerts {
			if alert.SlackThread != nil {
				st := uc.slackService.NewThread(*alert.SlackThread)
				if err := st.UpdateAlert(ctx, *alert); err != nil {
					// Log error but don't fail the main operation
					_ = msg.Trace(ctx, "💥 Failed to update alert in Slack: %s", err.Error())
				}
			}
		}
	}

	return newTicketPtr, nil
}

// GetSimilarTicketsForAlert finds tickets similar to a given alert based on embedding similarity
func (uc *UseCases) GetSimilarTicketsForAlert(ctx context.Context, alertID types.AlertID, threshold float64, offset, limit int) ([]*ticket.Ticket, int, error) {
	// Get target alert
	targetAlert, err := uc.repository.GetAlert(ctx, alertID)
	if err != nil {
		return nil, 0, goerr.Wrap(err, "failed to get target alert")
	}
	if targetAlert == nil {
		return nil, 0, goerr.New("alert not found", goerr.V("alert_id", alertID))
	}

	// If target alert has no embedding, return empty results
	if len(targetAlert.Embedding) == 0 {
		return []*ticket.Ticket{}, 0, nil
	}

	// Set default limit if not provided
	if limit <= 0 {
		limit = 5 // defaultSimilarTicketsLimit
	}

	// Fetch a fixed, large number of nearest neighbors to ensure stable pagination
	const maxCandidates = 1000 // maxSimilarTicketsCandidates
	candidates, err := uc.repository.FindNearestTickets(ctx, targetAlert.Embedding, maxCandidates)
	if err != nil {
		return nil, 0, goerr.Wrap(err, "failed to find nearest tickets")
	}

	// Filter by threshold to get complete result set
	var filteredTickets []*ticket.Ticket
	for _, candidate := range candidates {
		// Only include tickets with embeddings
		if len(candidate.Embedding) == 0 {
			continue
		}

		// Calculate cosine similarity and apply threshold
		similarity := cosineSimilarity(targetAlert.Embedding, candidate.Embedding)
		if float64(similarity) >= threshold {
			filteredTickets = append(filteredTickets, candidate)
		}
	}

	// Calculate correct total count from complete filtered result set
	totalCount := len(filteredTickets)

	// Apply pagination to the complete filtered result set
	start := offset
	if start > len(filteredTickets) {
		start = len(filteredTickets)
	}

	end := start + limit
	if end > len(filteredTickets) {
		end = len(filteredTickets)
	}

	result := filteredTickets[start:end]

	return result, totalCount, nil
}

// cosineSimilarity calculates the cosine similarity between two float32 vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float32
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}
