package diagnosis

import (
	"context"
	"fmt"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	diagnosismodel "github.com/secmon-lab/warren/pkg/domain/model/diagnosis"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

const RuleIDOrphanedTagID diagnosismodel.RuleID = "orphaned_tag_id"

// OrphanedTagIDRule detects alerts and tickets that reference tag IDs that no longer exist in the DB.
type OrphanedTagIDRule struct{}

func NewOrphanedTagIDRule() *OrphanedTagIDRule {
	return &OrphanedTagIDRule{}
}

func (r *OrphanedTagIDRule) ID() diagnosismodel.RuleID {
	return RuleIDOrphanedTagID
}

func (r *OrphanedTagIDRule) Check(ctx context.Context, repo interfaces.Repository) ([]diagnosismodel.Issue, error) {
	// Build set of valid tag IDs
	allTags, err := repo.ListAllTags(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to list all tags")
	}
	validTagIDs := make(map[string]bool, len(allTags))
	for _, tag := range allTags {
		validTagIDs[tag.ID] = true
	}

	var issues []diagnosismodel.Issue

	// Check alerts
	alerts, err := repo.GetAllAlerts(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get all alerts")
	}
	for _, a := range alerts {
		var orphaned []string
		for tagID := range a.TagIDs {
			if !validTagIDs[tagID] {
				orphaned = append(orphaned, tagID)
			}
		}
		if len(orphaned) > 0 {
			issue := diagnosismodel.NewIssue(
				types.EmptyDiagnosisID,
				RuleIDOrphanedTagID,
				string(a.ID),
				fmt.Sprintf("Alert %q has orphaned tag IDs: %s", a.ID, strings.Join(orphaned, ", ")),
			)
			issue.CreatedAt = clock.Now(ctx)
			issues = append(issues, issue)
		}
	}

	// Check tickets
	tickets, err := repo.GetAllTickets(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get all tickets")
	}
	for _, t := range tickets {
		var orphaned []string
		for tagID := range t.TagIDs {
			if !validTagIDs[tagID] {
				orphaned = append(orphaned, tagID)
			}
		}
		if len(orphaned) > 0 {
			// Prefix target ID with "ticket:" to distinguish from alert IDs
			issue := diagnosismodel.NewIssue(
				types.EmptyDiagnosisID,
				RuleIDOrphanedTagID,
				"ticket:"+string(t.ID),
				fmt.Sprintf("Ticket %q has orphaned tag IDs: %s", t.ID, strings.Join(orphaned, ", ")),
			)
			issue.CreatedAt = clock.Now(ctx)
			issues = append(issues, issue)
		}
	}

	return issues, nil
}

func (r *OrphanedTagIDRule) Fix(ctx context.Context, repo interfaces.Repository, issue diagnosismodel.Issue) error {
	// Rebuild the valid tag ID set
	allTags, err := repo.ListAllTags(ctx)
	if err != nil {
		return goerr.Wrap(err, "failed to list all tags")
	}
	validTagIDs := make(map[string]bool, len(allTags))
	for _, tag := range allTags {
		validTagIDs[tag.ID] = true
	}

	targetID := issue.TargetID
	if strings.HasPrefix(targetID, "ticket:") {
		// Fix ticket
		ticketIDStr := strings.TrimPrefix(targetID, "ticket:")
		ticketID := types.TicketID(ticketIDStr)
		t, err := repo.GetTicket(ctx, ticketID)
		if err != nil {
			return goerr.Wrap(err, "failed to get ticket", goerr.V("ticket_id", ticketID))
		}
		for tagID := range t.TagIDs {
			if !validTagIDs[tagID] {
				delete(t.TagIDs, tagID)
			}
		}
		if err := repo.PutTicket(ctx, *t); err != nil {
			return goerr.Wrap(err, "failed to save ticket after removing orphaned tags", goerr.V("ticket_id", ticketID))
		}
	} else {
		// Fix alert
		alertID := types.AlertID(targetID)
		a, err := repo.GetAlert(ctx, alertID)
		if err != nil {
			return goerr.Wrap(err, "failed to get alert", goerr.V("alert_id", alertID))
		}
		for tagID := range a.TagIDs {
			if !validTagIDs[tagID] {
				delete(a.TagIDs, tagID)
			}
		}
		if err := repo.PutAlert(ctx, *a); err != nil {
			return goerr.Wrap(err, "failed to save alert after removing orphaned tags", goerr.V("alert_id", alertID))
		}
	}
	return nil
}
