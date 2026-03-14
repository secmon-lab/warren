package firestore

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/diagnosis"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	collectionDiagnoses = "diagnoses"
	subcollectionIssues = "issues"
)

// PutDiagnosis saves or updates a diagnosis header.
func (r *Firestore) PutDiagnosis(ctx context.Context, d *diagnosis.Diagnosis) error {
	doc := r.db.Collection(collectionDiagnoses).Doc(d.ID.String())
	if _, err := doc.Set(ctx, d); err != nil {
		return r.eb.Wrap(err, "failed to put diagnosis", goerr.V("id", d.ID))
	}
	return nil
}

// GetDiagnosis retrieves a diagnosis by ID.
func (r *Firestore) GetDiagnosis(ctx context.Context, id types.DiagnosisID) (*diagnosis.Diagnosis, error) {
	doc, err := r.db.Collection(collectionDiagnoses).Doc(id.String()).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, r.eb.Wrap(goerr.New("diagnosis not found"),
				"not found",
				goerr.T(errutil.TagNotFound),
				goerr.V("id", id))
		}
		return nil, r.eb.Wrap(err, "failed to get diagnosis", goerr.V("id", id))
	}

	var d diagnosis.Diagnosis
	if err := doc.DataTo(&d); err != nil {
		return nil, r.eb.Wrap(err, "failed to unmarshal diagnosis", goerr.V("id", id))
	}
	return &d, nil
}

// ListDiagnoses returns a paginated list of diagnoses ordered by CreatedAt DESC.
func (r *Firestore) ListDiagnoses(ctx context.Context, offset, limit int) ([]*diagnosis.Diagnosis, int, error) {
	col := r.db.Collection(collectionDiagnoses)

	// Count total
	aggResult, err := col.NewAggregationQuery().WithCount("total").Get(ctx)
	if err != nil {
		return nil, 0, r.eb.Wrap(err, "failed to count diagnoses")
	}
	total, err := extractCountFromAggregationResult(aggResult, "total")
	if err != nil {
		return nil, 0, r.eb.Wrap(err, "failed to extract diagnosis count")
	}

	// Fetch paginated
	iter := col.OrderBy("CreatedAt", firestore.Desc).
		Offset(offset).
		Limit(limit).
		Documents(ctx)

	var result []*diagnosis.Diagnosis
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, 0, r.eb.Wrap(err, "failed to iterate diagnoses")
		}
		var d diagnosis.Diagnosis
		if err := doc.DataTo(&d); err != nil {
			return nil, 0, r.eb.Wrap(err, "failed to unmarshal diagnosis", goerr.V("id", doc.Ref.ID))
		}
		result = append(result, &d)
	}

	return result, total, nil
}

// PutDiagnosisIssue saves or updates a diagnosis issue.
// Path: diagnoses/{diagnosisID}/issues/{issueID}
func (r *Firestore) PutDiagnosisIssue(ctx context.Context, issue *diagnosis.Issue) error {
	doc := r.db.Collection(collectionDiagnoses).Doc(issue.DiagnosisID.String()).
		Collection(subcollectionIssues).Doc(issue.ID)
	if _, err := doc.Set(ctx, issue); err != nil {
		return r.eb.Wrap(err, "failed to put diagnosis issue",
			goerr.V("diagnosis_id", issue.DiagnosisID),
			goerr.V("issue_id", issue.ID))
	}
	return nil
}

// ListDiagnosisIssues returns paginated issues for a diagnosis ordered by CreatedAt ASC.
// status and ruleID are optional server-side filters.
func (r *Firestore) ListDiagnosisIssues(ctx context.Context, diagnosisID types.DiagnosisID, offset, limit int, issueStatus *diagnosis.IssueStatus, ruleID *diagnosis.RuleID) ([]*diagnosis.Issue, int, error) {
	col := r.db.Collection(collectionDiagnoses).Doc(diagnosisID.String()).Collection(subcollectionIssues)

	q := col.Query
	if issueStatus != nil {
		q = q.Where("Status", "==", string(*issueStatus))
	}
	if ruleID != nil {
		q = q.Where("RuleID", "==", string(*ruleID))
	}

	// Count matching total
	aggResult, err := q.NewAggregationQuery().WithCount("total").Get(ctx)
	if err != nil {
		return nil, 0, r.eb.Wrap(err, "failed to count diagnosis issues", goerr.V("diagnosis_id", diagnosisID))
	}
	total, err := extractCountFromAggregationResult(aggResult, "total")
	if err != nil {
		return nil, 0, r.eb.Wrap(err, "failed to extract issue count", goerr.V("diagnosis_id", diagnosisID))
	}

	// Fetch paginated
	iter := q.OrderBy("CreatedAt", firestore.Asc).
		Offset(offset).
		Limit(limit).
		Documents(ctx)

	var result []*diagnosis.Issue
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, 0, r.eb.Wrap(err, "failed to iterate diagnosis issues", goerr.V("diagnosis_id", diagnosisID))
		}
		var iss diagnosis.Issue
		if err := doc.DataTo(&iss); err != nil {
			return nil, 0, r.eb.Wrap(err, "failed to unmarshal diagnosis issue", goerr.V("id", doc.Ref.ID))
		}
		result = append(result, &iss)
	}

	return result, total, nil
}

// GetDiagnosisIssue retrieves a specific issue by diagnosisID and issueID.
func (r *Firestore) GetDiagnosisIssue(ctx context.Context, diagnosisID types.DiagnosisID, issueID string) (*diagnosis.Issue, error) {
	doc, err := r.db.Collection(collectionDiagnoses).Doc(diagnosisID.String()).
		Collection(subcollectionIssues).Doc(issueID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, r.eb.Wrap(goerr.New("diagnosis issue not found"),
				"not found",
				goerr.T(errutil.TagNotFound),
				goerr.V("diagnosis_id", diagnosisID),
				goerr.V("issue_id", issueID))
		}
		return nil, r.eb.Wrap(err, "failed to get diagnosis issue",
			goerr.V("diagnosis_id", diagnosisID),
			goerr.V("issue_id", issueID))
	}

	var iss diagnosis.Issue
	if err := doc.DataTo(&iss); err != nil {
		return nil, r.eb.Wrap(err, "failed to unmarshal diagnosis issue",
			goerr.V("diagnosis_id", diagnosisID),
			goerr.V("issue_id", issueID))
	}
	return &iss, nil
}

// CountDiagnosisIssues counts issues for a diagnosis, optionally filtered by status.
func (r *Firestore) CountDiagnosisIssues(ctx context.Context, diagnosisID types.DiagnosisID, issueStatus *diagnosis.IssueStatus) (int, error) {
	col := r.db.Collection(collectionDiagnoses).Doc(diagnosisID.String()).Collection(subcollectionIssues)

	var q firestore.Query
	if issueStatus != nil {
		q = col.Where("Status", "==", string(*issueStatus))
	} else {
		q = col.Query
	}

	aggResult, err := q.NewAggregationQuery().WithCount("total").Get(ctx)
	if err != nil {
		return 0, r.eb.Wrap(err, "failed to count diagnosis issues",
			goerr.V("diagnosis_id", diagnosisID))
	}
	return extractCountFromAggregationResult(aggResult, "total")
}

// GetDiagnosisIssueCounts returns all status counts for a diagnosis using 3 concurrent aggregation queries.
func (r *Firestore) GetDiagnosisIssueCounts(ctx context.Context, diagnosisID types.DiagnosisID) (diagnosis.IssueCounts, error) {
	col := r.db.Collection(collectionDiagnoses).Doc(diagnosisID.String()).Collection(subcollectionIssues)

	type countResult struct {
		status diagnosis.IssueStatus
		count  int
		err    error
	}

	statuses := []diagnosis.IssueStatus{
		diagnosis.IssueStatusPending,
		diagnosis.IssueStatusFixed,
		diagnosis.IssueStatusFailed,
	}

	ch := make(chan countResult, len(statuses))
	for _, s := range statuses {
		s := s
		go func() {
			q := col.Where("Status", "==", string(s))
			aggResult, err := q.NewAggregationQuery().WithCount("total").Get(ctx)
			if err != nil {
				ch <- countResult{status: s, err: err}
				return
			}
			n, err := extractCountFromAggregationResult(aggResult, "total")
			ch <- countResult{status: s, count: n, err: err}
		}()
	}

	var counts diagnosis.IssueCounts
	for range statuses {
		res := <-ch
		if res.err != nil {
			return diagnosis.IssueCounts{}, r.eb.Wrap(res.err, "failed to count diagnosis issues by status",
				goerr.V("diagnosis_id", diagnosisID),
				goerr.V("status", res.status))
		}
		switch res.status {
		case diagnosis.IssueStatusPending:
			counts.Pending = res.count
		case diagnosis.IssueStatusFixed:
			counts.Fixed = res.count
		case diagnosis.IssueStatusFailed:
			counts.Failed = res.count
		}
	}
	counts.Total = counts.Pending + counts.Fixed + counts.Failed
	return counts, nil
}

// BatchGetDiagnosisIssueCounts returns issue counts for multiple diagnoses.
func (r *Firestore) BatchGetDiagnosisIssueCounts(ctx context.Context, diagnosisIDs []types.DiagnosisID) (map[types.DiagnosisID]diagnosis.IssueCounts, error) {
	result := make(map[types.DiagnosisID]diagnosis.IssueCounts, len(diagnosisIDs))
	for _, id := range diagnosisIDs {
		counts, err := r.GetDiagnosisIssueCounts(ctx, id)
		if err != nil {
			return nil, err
		}
		result[id] = counts
	}
	return result, nil
}

// ListPendingDiagnosisIssues returns all pending issues for a diagnosis.
func (r *Firestore) ListPendingDiagnosisIssues(ctx context.Context, diagnosisID types.DiagnosisID) ([]*diagnosis.Issue, error) {
	iter := r.db.Collection(collectionDiagnoses).Doc(diagnosisID.String()).
		Collection(subcollectionIssues).
		Where("Status", "==", string(diagnosis.IssueStatusPending)).
		Documents(ctx)

	var result []*diagnosis.Issue
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, r.eb.Wrap(err, "failed to iterate pending issues", goerr.V("diagnosis_id", diagnosisID))
		}
		var iss diagnosis.Issue
		if err := doc.DataTo(&iss); err != nil {
			return nil, r.eb.Wrap(err, "failed to unmarshal diagnosis issue", goerr.V("id", doc.Ref.ID))
		}
		result = append(result, &iss)
	}
	return result, nil
}
