package diagnosis

import (
	"context"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	diagnosismodel "github.com/secmon-lab/warren/pkg/domain/model/diagnosis"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
)

const (
	RuleIDMissingAlertEmbedding  diagnosismodel.RuleID = "missing_alert_embedding"
	RuleIDMissingTicketEmbedding diagnosismodel.RuleID = "missing_ticket_embedding"
)

// MissingAlertEmbeddingRule detects alerts with empty embeddings.
type MissingAlertEmbeddingRule struct {
	llmClient gollem.LLMClient
}

func NewMissingAlertEmbeddingRule(llmClient gollem.LLMClient) *MissingAlertEmbeddingRule {
	return &MissingAlertEmbeddingRule{llmClient: llmClient}
}

func (r *MissingAlertEmbeddingRule) ID() diagnosismodel.RuleID {
	return RuleIDMissingAlertEmbedding
}

func (r *MissingAlertEmbeddingRule) Check(ctx context.Context, repo interfaces.Repository) ([]diagnosismodel.Issue, error) {
	// Reuse existing repository method that finds alerts without valid embedding
	alerts, err := repo.GetAlertsWithInvalidEmbedding(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get alerts with invalid embedding")
	}

	var issues []diagnosismodel.Issue
	for _, a := range alerts {
		issue := diagnosismodel.NewIssue(
			types.EmptyDiagnosisID, // DiagnosisID will be set by the usecase
			RuleIDMissingAlertEmbedding,
			string(a.ID),
			fmt.Sprintf("Alert %q has no embedding (title: %s)", a.ID, a.Title),
		)
		issue.CreatedAt = clock.Now(ctx)
		issues = append(issues, issue)
	}
	return issues, nil
}

func (r *MissingAlertEmbeddingRule) Fix(ctx context.Context, repo interfaces.Repository, issue diagnosismodel.Issue) error {
	if r.llmClient == nil {
		return goerr.New("LLM client is required to fix missing embeddings",
			goerr.T(errutil.TagInternal))
	}

	alertID := types.AlertID(issue.TargetID)
	a, err := repo.GetAlert(ctx, alertID)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert", goerr.V("alert_id", alertID))
	}

	availableTags, err := repo.ListAllTags(ctx)
	if err != nil {
		return goerr.Wrap(err, "failed to list tags", goerr.V("alert_id", alertID))
	}

	if err := a.FillMetadata(ctx, r.llmClient, availableTags); err != nil {
		return goerr.Wrap(err, "failed to fill alert metadata/embedding", goerr.V("alert_id", alertID))
	}

	if err := repo.PutAlert(ctx, *a); err != nil {
		return goerr.Wrap(err, "failed to save alert after embedding generation", goerr.V("alert_id", alertID))
	}
	return nil
}

// MissingTicketEmbeddingRule detects tickets with empty embeddings.
type MissingTicketEmbeddingRule struct {
	llmClient gollem.LLMClient
}

func NewMissingTicketEmbeddingRule(llmClient gollem.LLMClient) *MissingTicketEmbeddingRule {
	return &MissingTicketEmbeddingRule{llmClient: llmClient}
}

func (r *MissingTicketEmbeddingRule) ID() diagnosismodel.RuleID {
	return RuleIDMissingTicketEmbedding
}

func (r *MissingTicketEmbeddingRule) Check(ctx context.Context, repo interfaces.Repository) ([]diagnosismodel.Issue, error) {
	tickets, err := repo.GetTicketsWithInvalidEmbedding(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get tickets with invalid embedding")
	}

	var issues []diagnosismodel.Issue
	for _, t := range tickets {
		issue := diagnosismodel.NewIssue(
			types.EmptyDiagnosisID,
			RuleIDMissingTicketEmbedding,
			string(t.ID),
			fmt.Sprintf("Ticket %q has no embedding (title: %s)", t.ID, t.Title),
		)
		issue.CreatedAt = clock.Now(ctx)
		issues = append(issues, issue)
	}
	return issues, nil
}

func (r *MissingTicketEmbeddingRule) Fix(ctx context.Context, repo interfaces.Repository, issue diagnosismodel.Issue) error {
	if r.llmClient == nil {
		return goerr.New("LLM client is required to fix missing embeddings",
			goerr.T(errutil.TagInternal))
	}

	ticketID := types.TicketID(issue.TargetID)
	t, err := repo.GetTicket(ctx, ticketID)
	if err != nil {
		return goerr.Wrap(err, "failed to get ticket", goerr.V("ticket_id", ticketID))
	}

	if err := t.CalculateEmbedding(ctx, r.llmClient, repo); err != nil {
		return goerr.Wrap(err, "failed to calculate ticket embedding", goerr.V("ticket_id", ticketID))
	}

	if err := repo.PutTicket(ctx, *t); err != nil {
		return goerr.Wrap(err, "failed to save ticket after embedding generation", goerr.V("ticket_id", ticketID))
	}
	return nil
}
