package usecase

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	diagnosismodel "github.com/secmon-lab/warren/pkg/domain/model/diagnosis"
	"github.com/secmon-lab/warren/pkg/domain/types"
	diagnosisrule "github.com/secmon-lab/warren/pkg/usecase/diagnosis"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

// RunDiagnosis executes all registered diagnosis rules, collects issues,
// saves them as a subcollection under a new Diagnosis header, and returns the header.
func (u *UseCases) RunDiagnosis(ctx context.Context) (*diagnosismodel.Diagnosis, error) {
	rules := u.buildDiagnosisRules()

	diag := diagnosismodel.New(ctx)

	// Persist the header first so issues can reference the DiagnosisID
	if err := u.repository.PutDiagnosis(ctx, diag); err != nil {
		return nil, goerr.Wrap(err, "failed to save diagnosis header")
	}

	logger := logging.From(ctx)
	var totalIssues int

	for _, rule := range rules {
		issues, err := rule.Check(ctx, u.repository)
		if err != nil {
			// Log and continue; a single rule failure should not abort the whole diagnosis
			logger.Warn("diagnosis rule check failed", "rule", rule.ID(), "error", err)
			continue
		}

		for i := range issues {
			issues[i].DiagnosisID = diag.ID
			issues[i].CreatedAt = clock.Now(ctx)
			if err := u.repository.PutDiagnosisIssue(ctx, &issues[i]); err != nil {
				return nil, goerr.Wrap(err, "failed to save diagnosis issue",
					goerr.V("rule", rule.ID()),
					goerr.V("target_id", issues[i].TargetID))
			}
		}
		totalIssues += len(issues)
		logger.Info("diagnosis rule completed", "rule", rule.ID(), "issues", len(issues))
	}

	logger.Info("diagnosis completed", "total_issues", totalIssues)
	return diag, nil
}

// FixDiagnosis executes Fix for all pending issues in the given diagnosis.
// It updates each issue's status and the diagnosis overall status.
func (u *UseCases) FixDiagnosis(ctx context.Context, id types.DiagnosisID) (*diagnosismodel.Diagnosis, error) {
	diag, err := u.repository.GetDiagnosis(ctx, id)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get diagnosis", goerr.V("id", id))
	}

	pendingIssues, err := u.repository.ListPendingDiagnosisIssues(ctx, id)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to list pending issues", goerr.V("diagnosis_id", id))
	}

	if len(pendingIssues) == 0 {
		return diag, nil
	}

	// Build rule dispatch map
	dispatch := u.buildRuleDispatch()

	logger := logging.From(ctx)
	fixedCount := 0
	failedCount := 0

	for _, issue := range pendingIssues {
		rule, ok := dispatch[issue.RuleID]
		if !ok {
			// Unknown rule - mark as failed
			now := clock.Now(ctx)
			issue.Status = diagnosismodel.IssueStatusFailed
			issue.FailReason = "unknown rule ID"
			issue.FixedAt = &now
			if err := u.repository.PutDiagnosisIssue(ctx, issue); err != nil {
				logger.Warn("failed to update issue status", "issue_id", issue.ID, "error", err)
			}
			failedCount++
			continue
		}

		fixErr := rule.Fix(ctx, u.repository, *issue)
		now := clock.Now(ctx)
		issue.FixedAt = &now
		if fixErr != nil {
			logger.Warn("fix failed", "rule", issue.RuleID, "issue_id", issue.ID, "error", fixErr)
			issue.Status = diagnosismodel.IssueStatusFailed
			issue.FailReason = fixErr.Error()
			failedCount++
		} else {
			issue.Status = diagnosismodel.IssueStatusFixed
			fixedCount++
		}

		if err := u.repository.PutDiagnosisIssue(ctx, issue); err != nil {
			return nil, goerr.Wrap(err, "failed to update issue status", goerr.V("issue_id", issue.ID))
		}
	}

	// Update diagnosis status
	if failedCount == 0 {
		diag.Status = diagnosismodel.DiagnosisStatusFixed
	} else {
		diag.Status = diagnosismodel.DiagnosisStatusPartiallyFixed
	}
	diag.UpdatedAt = clock.Now(ctx)

	if err := u.repository.PutDiagnosis(ctx, diag); err != nil {
		return nil, goerr.Wrap(err, "failed to update diagnosis status", goerr.V("id", id))
	}

	logger.Info("fix completed", "fixed", fixedCount, "failed", failedCount)
	return diag, nil
}

// GetDiagnoses returns a paginated list of diagnoses.
func (u *UseCases) GetDiagnoses(ctx context.Context, offset, limit int) ([]*diagnosismodel.Diagnosis, int, error) {
	return u.repository.ListDiagnoses(ctx, offset, limit)
}

// GetDiagnosis retrieves a single diagnosis by ID.
func (u *UseCases) GetDiagnosis(ctx context.Context, id types.DiagnosisID) (*diagnosismodel.Diagnosis, error) {
	d, err := u.repository.GetDiagnosis(ctx, id)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get diagnosis", goerr.V("id", id))
	}
	return d, nil
}

// GetDiagnosisIssues returns paginated issues for a diagnosis.
// status and ruleID are optional filters.
func (u *UseCases) GetDiagnosisIssues(ctx context.Context, id types.DiagnosisID, offset, limit int, status *diagnosismodel.IssueStatus, ruleID *diagnosismodel.RuleID) ([]*diagnosismodel.Issue, int, error) {
	return u.repository.ListDiagnosisIssues(ctx, id, offset, limit, status, ruleID)
}

// CountDiagnosisIssues returns issue counts for a diagnosis using a single repository call.
func (u *UseCases) CountDiagnosisIssues(ctx context.Context, id types.DiagnosisID) (diagnosismodel.IssueCounts, error) {
	return u.repository.GetDiagnosisIssueCounts(ctx, id)
}

// BatchCountDiagnosisIssues returns issue counts for multiple diagnoses in one call.
func (u *UseCases) BatchCountDiagnosisIssues(ctx context.Context, ids []types.DiagnosisID) (map[types.DiagnosisID]diagnosismodel.IssueCounts, error) {
	return u.repository.BatchGetDiagnosisIssueCounts(ctx, ids)
}

// buildDiagnosisRules constructs the list of all registered diagnosis rules.
func (u *UseCases) buildDiagnosisRules() []diagnosisrule.Rule {
	rules := []diagnosisrule.Rule{
		diagnosisrule.NewMissingAlertEmbeddingRule(u.llmClient),
		diagnosisrule.NewMissingTicketEmbeddingRule(u.llmClient),
		diagnosisrule.NewLegacyAlertStatusRule(),
		diagnosisrule.NewLegacyTicketStatusRule(),
		diagnosisrule.NewBindingMismatchRule(),
		diagnosisrule.NewOrphanedTagIDRule(),
		diagnosisrule.NewMissingAlertMetadataRule(u.llmClient),
	}
	return rules
}

// buildRuleDispatch creates a map from RuleID to Rule for efficient dispatch during Fix.
func (u *UseCases) buildRuleDispatch() map[diagnosismodel.RuleID]diagnosisrule.Rule {
	dispatch := make(map[diagnosismodel.RuleID]diagnosisrule.Rule)
	for _, rule := range u.buildDiagnosisRules() {
		dispatch[rule.ID()] = rule
	}
	return dispatch
}

// DiagnosisUsecases defines the interface for diagnosis-related use cases used by the API layer.
type DiagnosisUsecases interface {
	RunDiagnosis(ctx context.Context) (*diagnosismodel.Diagnosis, error)
	FixDiagnosis(ctx context.Context, id types.DiagnosisID) (*diagnosismodel.Diagnosis, error)
	GetDiagnoses(ctx context.Context, offset, limit int) ([]*diagnosismodel.Diagnosis, int, error)
	GetDiagnosis(ctx context.Context, id types.DiagnosisID) (*diagnosismodel.Diagnosis, error)
	GetDiagnosisIssues(ctx context.Context, id types.DiagnosisID, offset, limit int, status *diagnosismodel.IssueStatus, ruleID *diagnosismodel.RuleID) ([]*diagnosismodel.Issue, int, error)
	CountDiagnosisIssues(ctx context.Context, id types.DiagnosisID) (diagnosismodel.IssueCounts, error)
	BatchCountDiagnosisIssues(ctx context.Context, ids []types.DiagnosisID) (map[types.DiagnosisID]diagnosismodel.IssueCounts, error)
}

var _ DiagnosisUsecases = &UseCases{}
