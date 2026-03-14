package diagnosis

import (
	"context"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	diagnosismodel "github.com/secmon-lab/warren/pkg/domain/model/diagnosis"
	ticketpkg "github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
)

const RuleIDBindingMismatch diagnosismodel.RuleID = "binding_mismatch"

// BindingMismatchRule detects bidirectional reference inconsistencies between
// alerts and tickets. Specifically:
//   - Alert.TicketID points to a ticket that does not exist
//   - Alert.TicketID points to a ticket that does not list the alert in its AlertIDs
type BindingMismatchRule struct{}

func NewBindingMismatchRule() *BindingMismatchRule {
	return &BindingMismatchRule{}
}

func (r *BindingMismatchRule) ID() diagnosismodel.RuleID {
	return RuleIDBindingMismatch
}

func (r *BindingMismatchRule) Check(ctx context.Context, repo interfaces.Repository) ([]diagnosismodel.Issue, error) {
	alerts, err := repo.GetAllAlerts(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get all alerts")
	}

	// Collect unique ticket IDs referenced by alerts to avoid N+1 queries.
	ticketIDSet := make(map[types.TicketID]struct{})
	for _, a := range alerts {
		if a.TicketID != types.EmptyTicketID {
			ticketIDSet[a.TicketID] = struct{}{}
		}
	}

	ticketIDs := make([]types.TicketID, 0, len(ticketIDSet))
	for id := range ticketIDSet {
		ticketIDs = append(ticketIDs, id)
	}

	tickets, err := repo.BatchGetTickets(ctx, ticketIDs)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to batch get tickets")
	}

	ticketMap := make(map[types.TicketID]*ticketpkg.Ticket, len(tickets))
	for _, t := range tickets {
		if t != nil {
			ticketMap[t.ID] = t
		}
	}

	now := clock.Now(ctx)
	var issues []diagnosismodel.Issue
	for _, a := range alerts {
		if a.TicketID == types.EmptyTicketID {
			continue
		}

		t, ok := ticketMap[a.TicketID]
		if !ok {
			// Alert points to a non-existent ticket
			issue := diagnosismodel.NewIssue(
				types.EmptyDiagnosisID,
				RuleIDBindingMismatch,
				string(a.ID),
				fmt.Sprintf("Alert %q references non-existent ticket %q", a.ID, a.TicketID),
			)
			issue.CreatedAt = now
			issues = append(issues, issue)
			continue
		}

		// Check if the ticket's AlertIDs contains this alert
		found := false
		for _, aid := range t.AlertIDs {
			if aid == a.ID {
				found = true
				break
			}
		}
		if !found {
			issue := diagnosismodel.NewIssue(
				types.EmptyDiagnosisID,
				RuleIDBindingMismatch,
				string(a.ID),
				fmt.Sprintf("Alert %q references ticket %q, but ticket does not list this alert in its AlertIDs", a.ID, a.TicketID),
			)
			issue.CreatedAt = now
			issues = append(issues, issue)
		}
	}
	return issues, nil
}

func (r *BindingMismatchRule) Fix(ctx context.Context, repo interfaces.Repository, issue diagnosismodel.Issue) error {
	alertID := types.AlertID(issue.TargetID)
	a, err := repo.GetAlert(ctx, alertID)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert", goerr.V("alert_id", alertID))
	}

	if a.TicketID == types.EmptyTicketID {
		// Already fixed (nothing to do)
		return nil
	}

	t, err := repo.GetTicket(ctx, a.TicketID)
	if err != nil {
		if isNotFound(err) {
			// Ticket does not exist: clear the TicketID reference on the alert
			a.TicketID = types.EmptyTicketID
			if err := repo.PutAlert(ctx, *a); err != nil {
				return goerr.Wrap(err, "failed to clear TicketID on alert", goerr.V("alert_id", alertID))
			}
			return nil
		}
		return goerr.Wrap(err, "failed to get ticket", goerr.V("ticket_id", a.TicketID))
	}

	// Ticket exists but does not list this alert: add it
	for _, aid := range t.AlertIDs {
		if aid == alertID {
			return nil // Already consistent
		}
	}
	t.AlertIDs = append(t.AlertIDs, alertID)
	if err := repo.PutTicket(ctx, *t); err != nil {
		return goerr.Wrap(err, "failed to add alert to ticket AlertIDs",
			goerr.V("ticket_id", t.ID),
			goerr.V("alert_id", alertID))
	}
	return nil
}

// isNotFound returns true if the error is tagged as not-found.
func isNotFound(err error) bool {
	return goerr.HasTag(err, errutil.TagNotFound)
}
