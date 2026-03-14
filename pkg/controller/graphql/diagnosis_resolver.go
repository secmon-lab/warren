package graphql

import (
	"context"

	goerr "github.com/m-mizutani/goerr/v2"
	diagnosismodel "github.com/secmon-lab/warren/pkg/domain/model/diagnosis"
	graphql1 "github.com/secmon-lab/warren/pkg/domain/model/graphql"
)

// diagnosisToGraphQL converts a domain Diagnosis to a GraphQL Diagnosis, fetching issue counts.
func (r *Resolver) diagnosisToGraphQL(ctx context.Context, d *diagnosismodel.Diagnosis) (*graphql1.Diagnosis, error) {
	total, pending, fixed, failed, err := r.uc.CountDiagnosisIssues(ctx, d.ID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to count diagnosis issues", goerr.V("id", d.ID))
	}

	return &graphql1.Diagnosis{
		ID:           string(d.ID),
		Status:       string(d.Status),
		TotalCount:   total,
		PendingCount: pending,
		FixedCount:   fixed,
		FailedCount:  failed,
		CreatedAt:    d.CreatedAt.Format("2006/01/02 15:04:05"),
		UpdatedAt:    d.UpdatedAt.Format("2006/01/02 15:04:05"),
	}, nil
}

// issueToGraphQL converts a domain Issue to a GraphQL DiagnosisIssue.
func issueToGraphQL(iss *diagnosismodel.Issue) *graphql1.DiagnosisIssue {
	gql := &graphql1.DiagnosisIssue{
		ID:          iss.ID,
		DiagnosisID: string(iss.DiagnosisID),
		RuleID:      string(iss.RuleID),
		TargetID:    iss.TargetID,
		Description: iss.Description,
		Status:      string(iss.Status),
		CreatedAt:   iss.CreatedAt.Format("2006/01/02 15:04:05"),
	}

	if iss.FixedAt != nil {
		s := iss.FixedAt.Format("2006/01/02 15:04:05") //nolint:govet
		gql.FixedAt = &s
	}
	if iss.FailReason != "" {
		gql.FailReason = &iss.FailReason
	}

	return gql
}
