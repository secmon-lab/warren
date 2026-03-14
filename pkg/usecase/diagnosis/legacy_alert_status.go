package diagnosis

import (
	"context"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	alertmodel "github.com/secmon-lab/warren/pkg/domain/model/alert"
	diagnosismodel "github.com/secmon-lab/warren/pkg/domain/model/diagnosis"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

const RuleIDLegacyAlertStatus diagnosismodel.RuleID = "legacy_alert_status"

// LegacyAlertStatusRule detects alerts whose Status is "" or "unbound" (pre-v0.10.0 data).
type LegacyAlertStatusRule struct{}

func NewLegacyAlertStatusRule() *LegacyAlertStatusRule {
	return &LegacyAlertStatusRule{}
}

func (r *LegacyAlertStatusRule) ID() diagnosismodel.RuleID {
	return RuleIDLegacyAlertStatus
}

func (r *LegacyAlertStatusRule) Check(ctx context.Context, repo interfaces.Repository) ([]diagnosismodel.Issue, error) {
	alerts, err := repo.GetAllAlerts(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get all alerts")
	}

	var issues []diagnosismodel.Issue
	for _, a := range alerts {
		if a.Status == "" || a.Status == "unbound" {
			issue := diagnosismodel.NewIssue(
				types.EmptyDiagnosisID,
				RuleIDLegacyAlertStatus,
				string(a.ID),
				fmt.Sprintf("Alert %q has legacy status %q (should be %q)", a.ID, a.Status, alertmodel.AlertStatusActive),
			)
			issue.CreatedAt = clock.Now(ctx)
			issues = append(issues, issue)
		}
	}
	return issues, nil
}

func (r *LegacyAlertStatusRule) Fix(ctx context.Context, repo interfaces.Repository, issue diagnosismodel.Issue) error {
	alertID := types.AlertID(issue.TargetID)
	a, err := repo.GetAlert(ctx, alertID)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert", goerr.V("alert_id", alertID))
	}

	a.Status = alertmodel.AlertStatusActive
	if err := repo.PutAlert(ctx, *a); err != nil {
		return goerr.Wrap(err, "failed to save alert with updated status", goerr.V("alert_id", alertID))
	}
	return nil
}
