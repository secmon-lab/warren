package diagnosis

import (
	"context"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	alertmodel "github.com/secmon-lab/warren/pkg/domain/model/alert"
	diagnosismodel "github.com/secmon-lab/warren/pkg/domain/model/diagnosis"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
)

const RuleIDMissingAlertMetadata diagnosismodel.RuleID = "missing_alert_metadata"

// MissingAlertMetadataRule detects alerts that still have default title or description
// set by the ingest pipeline when LLM generation failed.
type MissingAlertMetadataRule struct {
	llmClient gollem.LLMClient
}

func NewMissingAlertMetadataRule(llmClient gollem.LLMClient) *MissingAlertMetadataRule {
	return &MissingAlertMetadataRule{llmClient: llmClient}
}

func (r *MissingAlertMetadataRule) ID() diagnosismodel.RuleID {
	return RuleIDMissingAlertMetadata
}

func (r *MissingAlertMetadataRule) Check(ctx context.Context, repo interfaces.Repository) ([]diagnosismodel.Issue, error) {
	alerts, err := repo.GetAllAlerts(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get all alerts")
	}

	var issues []diagnosismodel.Issue
	for _, a := range alerts {
		hasDefaultTitle := a.Title == alertmodel.DefaultAlertTitle || a.Title == ""
		hasDefaultDesc := a.Description == alertmodel.DefaultAlertDescription || a.Description == ""
		if hasDefaultTitle || hasDefaultDesc {
			issue := diagnosismodel.NewIssue(
				types.EmptyDiagnosisID,
				RuleIDMissingAlertMetadata,
				string(a.ID),
				fmt.Sprintf("Alert %q has default/empty metadata (title=%q, description=%q)", a.ID, a.Title, a.Description),
			)
			issue.CreatedAt = clock.Now(ctx)
			issues = append(issues, issue)
		}
	}
	return issues, nil
}

func (r *MissingAlertMetadataRule) Fix(ctx context.Context, repo interfaces.Repository, issue diagnosismodel.Issue) error {
	if r.llmClient == nil {
		return goerr.New("LLM client is required to fix missing alert metadata",
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
		return goerr.Wrap(err, "failed to fill alert metadata", goerr.V("alert_id", alertID))
	}

	if err := repo.PutAlert(ctx, *a); err != nil {
		return goerr.Wrap(err, "failed to save alert after metadata generation", goerr.V("alert_id", alertID))
	}
	return nil
}
