package graphql

import (
	diagnosismodel "github.com/secmon-lab/warren/pkg/domain/model/diagnosis"
	graphql1 "github.com/secmon-lab/warren/pkg/domain/model/graphql"
)

// diagnosisToGraphQL converts a domain Diagnosis to a GraphQL Diagnosis using pre-fetched counts.
func diagnosisToGraphQL(d *diagnosismodel.Diagnosis, counts diagnosismodel.IssueCounts) *graphql1.Diagnosis {
	return &graphql1.Diagnosis{
		ID:           string(d.ID),
		Status:       string(d.Status),
		TotalCount:   counts.Total,
		PendingCount: counts.Pending,
		FixedCount:   counts.Fixed,
		FailedCount:  counts.Failed,
		CreatedAt:    d.CreatedAt.Format("2006/01/02 15:04:05"),
		UpdatedAt:    d.UpdatedAt.Format("2006/01/02 15:04:05"),
	}
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
