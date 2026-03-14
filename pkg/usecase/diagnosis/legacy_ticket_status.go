package diagnosis

import (
	"context"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	diagnosismodel "github.com/secmon-lab/warren/pkg/domain/model/diagnosis"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

const RuleIDLegacyTicketStatus diagnosismodel.RuleID = "legacy_ticket_status"

// LegacyTicketStatusRule detects tickets whose Status is "pending" (pre-deprecation data).
// NormalizeLegacyStatus() handles this at read-time, but the DB record remains dirty.
type LegacyTicketStatusRule struct{}

func NewLegacyTicketStatusRule() *LegacyTicketStatusRule {
	return &LegacyTicketStatusRule{}
}

func (r *LegacyTicketStatusRule) ID() diagnosismodel.RuleID {
	return RuleIDLegacyTicketStatus
}

func (r *LegacyTicketStatusRule) Check(ctx context.Context, repo interfaces.Repository) ([]diagnosismodel.Issue, error) {
	tickets, err := repo.GetAllTickets(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get all tickets")
	}

	var issues []diagnosismodel.Issue
	for _, t := range tickets {
		if string(t.Status) == "pending" {
			issue := diagnosismodel.NewIssue(
				types.EmptyDiagnosisID,
				RuleIDLegacyTicketStatus,
				string(t.ID),
				fmt.Sprintf("Ticket %q has legacy status %q (should be %q)", t.ID, t.Status, types.TicketStatusOpen),
			)
			issue.CreatedAt = clock.Now(ctx)
			issues = append(issues, issue)
		}
	}
	return issues, nil
}

func (r *LegacyTicketStatusRule) Fix(ctx context.Context, repo interfaces.Repository, issue diagnosismodel.Issue) error {
	ticketID := types.TicketID(issue.TargetID)
	t, err := repo.GetTicket(ctx, ticketID)
	if err != nil {
		return goerr.Wrap(err, "failed to get ticket", goerr.V("ticket_id", ticketID))
	}

	t.Status = types.TicketStatusOpen
	if err := repo.PutTicket(ctx, *t); err != nil {
		return goerr.Wrap(err, "failed to save ticket with updated status", goerr.V("ticket_id", ticketID))
	}
	return nil
}
