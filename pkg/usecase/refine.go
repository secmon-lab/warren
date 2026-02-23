package usecase

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/lang"
	"github.com/secmon-lab/warren/pkg/domain/model/prompt"
	"github.com/secmon-lab/warren/pkg/domain/model/refine"
	modelslack "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/llm"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	slackSDK "github.com/slack-go/slack"
)

//go:embed prompt/refine_ticket_review.md
var refineTicketReviewPromptTemplate string

//go:embed prompt/refine_alert_summary.md
var refineAlertSummaryPromptTemplate string

//go:embed prompt/refine_alert_consolidation.md
var refineAlertConsolidationPromptTemplate string

const maxUnboundAlerts = 100

// LLM response types

type ticketReviewResult struct {
	Message string `json:"message"`
	Reason  string `json:"reason"`
}

type alertSummaryResult struct {
	AlertID    string   `json:"alert_id"`
	Title      string   `json:"title"`
	Identities []string `json:"identities"`
	Parameters []string `json:"parameters"`
	Context    string   `json:"context"`
	RootCause  string   `json:"root_cause"`
}

type consolidationResult struct {
	Groups []consolidationGroup `json:"groups"`
}

type consolidationGroup struct {
	Reason         string   `json:"reason"`
	PrimaryAlertID string   `json:"primary_alert_id"`
	AlertIDs       []string `json:"alert_ids"`
}

// Refine executes the refine process: review open tickets and consolidate unbound alerts.
func (uc *UseCases) Refine(ctx context.Context) error {
	if uc.llmClient == nil {
		return goerr.New("LLM client is required for refine")
	}

	logger := logging.From(ctx)

	logger.Info("starting refine: reviewing open tickets")
	if err := uc.reviewOpenTickets(ctx); err != nil {
		logger.Error("failed to review open tickets", "error", err)
	}

	logger.Info("starting refine: consolidating unbound alerts")
	if err := uc.consolidateUnboundAlerts(ctx); err != nil {
		logger.Error("failed to consolidate unbound alerts", "error", err)
	}

	logger.Info("refine completed")
	return nil
}

// reviewOpenTickets reviews all open tickets and posts follow-up messages where needed.
func (uc *UseCases) reviewOpenTickets(ctx context.Context) error {
	logger := logging.From(ctx)

	tickets, err := uc.repository.GetTicketsByStatus(ctx, []types.TicketStatus{types.TicketStatusOpen}, "", "", 0, 0)
	if err != nil {
		return goerr.Wrap(err, "failed to get open tickets")
	}

	if len(tickets) == 0 {
		logger.Info("no open tickets found")
		return nil
	}

	for _, t := range tickets {
		if err := uc.reviewSingleTicket(ctx, t); err != nil {
			logger.Error("failed to review ticket", "ticket_id", t.ID, "error", err)
		}
	}

	return nil
}

func (uc *UseCases) reviewSingleTicket(ctx context.Context, t *ticket.Ticket) error {
	logger := logging.From(ctx)

	// Get ticket comments
	comments, err := uc.repository.GetTicketComments(ctx, t.ID)
	if err != nil {
		return goerr.Wrap(err, "failed to get ticket comments", goerr.V("ticket_id", t.ID))
	}

	// Get alerts linked to this ticket
	alerts, err := uc.repository.BatchGetAlerts(ctx, t.AlertIDs)
	if err != nil {
		return goerr.Wrap(err, "failed to get alerts for ticket", goerr.V("ticket_id", t.ID))
	}

	// Build assignee string
	assignee := ""
	if t.Assignee != nil {
		assignee = t.Assignee.Name
	}

	// Generate prompt
	reviewPrompt, err := prompt.GenerateWithStruct(ctx, refineTicketReviewPromptTemplate, map[string]any{
		"title":       t.Title,
		"description": t.Description,
		"status":      t.Status.String(),
		"created_at":  t.CreatedAt.Format("2006-01-02 15:04"),
		"assignee":    assignee,
		"alerts":      alerts,
		"comments":    comments,
		"now":         clock.Now(ctx).Format("2006-01-02 15:04"),
		"lang":        lang.From(ctx),
	})
	if err != nil {
		return goerr.Wrap(err, "failed to generate ticket review prompt")
	}

	// Ask LLM
	result, err := llm.Ask[ticketReviewResult](ctx, uc.llmClient, reviewPrompt)
	if err != nil {
		return goerr.Wrap(err, "failed to get ticket review from LLM", goerr.V("ticket_id", t.ID))
	}

	logger.Info("ticket review result",
		"ticket_id", t.ID,
		"has_message", result.Message != "",
		"reason", result.Reason,
	)

	if result.Message == "" {
		return nil
	}

	// Post follow-up message to ticket's Slack thread
	if uc.slackService != nil && t.SlackThread != nil {
		threadSvc := uc.slackService.NewThread(*t.SlackThread)
		if err := threadSvc.PostComment(ctx, result.Message); err != nil {
			logger.Error("failed to post refine comment to ticket thread",
				"ticket_id", t.ID, "error", err)
		}
	} else {
		// Console output for CLI mode
		logger.Info("refine follow-up",
			"ticket_id", t.ID,
			"message", result.Message,
		)
	}

	return nil
}

// consolidateUnboundAlerts finds unbound alerts that can be grouped and proposes consolidation.
func (uc *UseCases) consolidateUnboundAlerts(ctx context.Context) error {
	logger := logging.From(ctx)

	// Get unbound alerts (with limit)
	unboundAlerts, err := uc.repository.GetAlertWithoutTicket(ctx, 0, maxUnboundAlerts)
	if err != nil {
		return goerr.Wrap(err, "failed to get unbound alerts")
	}

	if len(unboundAlerts) < 2 {
		logger.Info("not enough unbound alerts for consolidation", "count", len(unboundAlerts))
		return nil
	}

	// Phase 1: Generate individual summaries
	logger.Info("phase 1: generating alert summaries", "count", len(unboundAlerts))
	var summaries []alertSummaryResult
	for _, a := range unboundAlerts {
		summary, err := uc.summarizeAlert(ctx, a)
		if err != nil {
			logger.Error("failed to summarize alert", "alert_id", a.ID, "error", err)
			continue
		}
		summaries = append(summaries, *summary)
	}

	if len(summaries) < 2 {
		logger.Info("not enough summarized alerts for consolidation", "count", len(summaries))
		return nil
	}

	// Phase 2: Find consolidation groups
	logger.Info("phase 2: finding consolidation groups", "summaries", len(summaries))
	groups, err := uc.findConsolidationGroups(ctx, summaries)
	if err != nil {
		return goerr.Wrap(err, "failed to find consolidation groups")
	}

	if len(groups) == 0 {
		logger.Info("no consolidation groups found")
		return nil
	}

	// Phase 3: Save groups and post proposals
	logger.Info("phase 3: posting consolidation proposals", "groups", len(groups))
	for _, group := range groups {
		if err := uc.postConsolidationProposal(ctx, group); err != nil {
			logger.Error("failed to post consolidation proposal", "error", err)
		}
	}

	return nil
}

func (uc *UseCases) summarizeAlert(ctx context.Context, a *alert.Alert) (*alertSummaryResult, error) {
	dataJSON, err := json.Marshal(a.Data)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to marshal alert data")
	}

	summaryPrompt, err := prompt.Generate(ctx, refineAlertSummaryPromptTemplate, map[string]any{
		"alert_id":    a.ID.String(),
		"title":       a.Title,
		"description": a.Description,
		"schema":      string(a.Schema),
		"created_at":  a.CreatedAt.Format("2006-01-02 15:04"),
		"data":        string(dataJSON),
	})
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate alert summary prompt")
	}

	result, err := llm.Ask[alertSummaryResult](ctx, uc.llmClient, summaryPrompt)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get alert summary from LLM")
	}

	return result, nil
}

func (uc *UseCases) findConsolidationGroups(ctx context.Context, summaries []alertSummaryResult) ([]consolidationGroup, error) {
	consolidationPrompt, err := prompt.GenerateWithStruct(ctx, refineAlertConsolidationPromptTemplate, map[string]any{
		"summaries": summaries,
	})
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate consolidation prompt")
	}

	result, err := llm.Ask[consolidationResult](ctx, uc.llmClient, consolidationPrompt,
		llm.WithValidate(func(r consolidationResult) error {
			for _, g := range r.Groups {
				if len(g.AlertIDs) < 2 {
					return goerr.New("group must contain at least 2 alerts")
				}
				if len(g.AlertIDs) > 10 {
					return goerr.New("group must contain at most 10 alerts")
				}
				if g.PrimaryAlertID == "" {
					return goerr.New("primary alert ID is required")
				}
				if g.Reason == "" {
					return goerr.New("reason is required")
				}
			}
			return nil
		}),
	)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get consolidation groups from LLM")
	}

	return result.Groups, nil
}

func (uc *UseCases) postConsolidationProposal(ctx context.Context, group consolidationGroup) error {
	logger := logging.From(ctx)

	// Convert string IDs to typed IDs
	primaryAlertID := types.AlertID(group.PrimaryAlertID)
	alertIDs := make([]types.AlertID, len(group.AlertIDs))
	for i, id := range group.AlertIDs {
		alertIDs[i] = types.AlertID(id)
	}

	// Save RefineGroup to DB
	refineGroup := &refine.Group{
		ID:             types.NewRefineGroupID(),
		PrimaryAlertID: primaryAlertID,
		AlertIDs:       alertIDs,
		Reason:         group.Reason,
		CreatedAt:      clock.Now(ctx),
		Status:         refine.GroupStatusPending,
	}

	if err := uc.repository.PutRefineGroup(ctx, refineGroup); err != nil {
		return goerr.Wrap(err, "failed to save refine group")
	}

	// Get primary alert for Slack thread info
	primaryAlert, err := uc.repository.GetAlert(ctx, primaryAlertID)
	if err != nil {
		return goerr.Wrap(err, "failed to get primary alert", goerr.V("alert_id", primaryAlertID))
	}
	if primaryAlert == nil {
		return goerr.New("primary alert not found", goerr.V("alert_id", primaryAlertID))
	}

	// Get all alerts in the group to build links
	groupAlerts, err := uc.repository.BatchGetAlerts(ctx, alertIDs)
	if err != nil {
		return goerr.Wrap(err, "failed to get group alerts")
	}

	if uc.slackService == nil || !primaryAlert.HasSlackThread() {
		// Console output for CLI mode
		logger.Info("consolidation proposal",
			"group_id", refineGroup.ID,
			"primary_alert", primaryAlertID,
			"alert_count", len(alertIDs),
			"reason", group.Reason,
		)
		return nil
	}

	// Build Slack Block Kit message
	blocks := uc.buildConsolidationBlocks(refineGroup, groupAlerts)

	// Post to primary alert's thread with reply_broadcast
	client := uc.slackService.GetClient()
	_, _, err = client.PostMessageContext(
		ctx,
		primaryAlert.SlackThread.ChannelID,
		slackSDK.MsgOptionTS(primaryAlert.SlackThread.ThreadID),
		slackSDK.MsgOptionBroadcast(),
		slackSDK.MsgOptionBlocks(blocks...),
	)
	if err != nil {
		return goerr.Wrap(err, "failed to post consolidation proposal to Slack")
	}

	logger.Info("posted consolidation proposal",
		"group_id", refineGroup.ID,
		"channel", primaryAlert.SlackThread.ChannelID,
		"thread", primaryAlert.SlackThread.ThreadID,
	)

	return nil
}

// handleCreateTicketFromRefine handles the "Create Ticket" button press from a consolidation proposal.
func (uc *UseCases) handleCreateTicketFromRefine(ctx context.Context, slackUser modelslack.User, slackThread modelslack.Thread, groupID types.RefineGroupID) error {
	logger := logging.From(ctx)

	// Get the refine group from DB
	group, err := uc.repository.GetRefineGroup(ctx, groupID)
	if err != nil {
		return goerr.Wrap(err, "failed to get refine group", goerr.V("group_id", groupID))
	}

	// Create ticket from the group's alerts using existing logic
	newTicket, err := uc.CreateTicketFromAlerts(ctx, group.AlertIDs, &slackUser, &slackThread)
	if err != nil {
		return goerr.Wrap(err, "failed to create ticket from refine group", goerr.V("group_id", groupID))
	}

	// Update group status to accepted
	group.Status = refine.GroupStatusAccepted
	if err := uc.repository.PutRefineGroup(ctx, group); err != nil {
		logger.Error("failed to update refine group status", "group_id", groupID, "error", err)
	}

	logger.Info("created ticket from refine group",
		"ticket_id", newTicket.ID,
		"group_id", groupID,
		"alert_count", len(group.AlertIDs),
	)

	return nil
}

// buildConsolidationBlocks builds the Slack Block Kit blocks for a consolidation proposal.
func (uc *UseCases) buildConsolidationBlocks(group *refine.Group, alerts alert.Alerts) []slackSDK.Block {
	var blocks []slackSDK.Block

	// Header
	blocks = append(blocks, slackSDK.NewHeaderBlock(
		slackSDK.NewTextBlockObject(slackSDK.PlainTextType, "🔗 Alert Consolidation Proposal", false, false),
	))

	// Reason
	blocks = append(blocks, slackSDK.NewSectionBlock(
		slackSDK.NewTextBlockObject(slackSDK.MarkdownType, fmt.Sprintf("*Reason:* %s", group.Reason), false, false),
		nil, nil,
	))

	blocks = append(blocks, slackSDK.NewDividerBlock())

	// Alert list
	alertText := "*Candidate Alerts:*\n"
	for _, a := range alerts {
		if a == nil {
			continue
		}
		if a.HasSlackThread() && uc.slackService != nil {
			alertURL := uc.slackService.ToExternalMsgURL(a.SlackThread.ChannelID, a.SlackThread.ThreadID)
			alertText += fmt.Sprintf("• <%s|%s> (`%s`)\n", alertURL, a.Title, a.ID)
		} else {
			alertText += fmt.Sprintf("• %s (`%s`)\n", a.Title, a.ID)
		}
	}

	blocks = append(blocks, slackSDK.NewSectionBlock(
		slackSDK.NewTextBlockObject(slackSDK.MarkdownType, alertText, false, false),
		nil, nil,
	))

	// Create Ticket button
	blocks = append(blocks, slackSDK.NewActionBlock(
		"refine_create_ticket",
		slackSDK.NewButtonBlockElement(
			modelslack.ActionIDCreateTicketFromRefine.String(),
			string(group.ID),
			slackSDK.NewTextBlockObject(slackSDK.PlainTextType, "Create Ticket", false, false),
		).WithStyle(slackSDK.StylePrimary),
	))

	return blocks
}
