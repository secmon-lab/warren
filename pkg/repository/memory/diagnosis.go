package memory

import (
	"context"
	"sort"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/diagnosis"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
)

// PutDiagnosis saves or updates a diagnosis header.
func (r *Memory) PutDiagnosis(_ context.Context, d *diagnosis.Diagnosis) error {
	r.diagnosisMu.Lock()
	defer r.diagnosisMu.Unlock()

	cp := *d
	r.diagnoses[d.ID] = &cp
	return nil
}

// GetDiagnosis retrieves a diagnosis by ID.
func (r *Memory) GetDiagnosis(_ context.Context, id types.DiagnosisID) (*diagnosis.Diagnosis, error) {
	r.diagnosisMu.RLock()
	defer r.diagnosisMu.RUnlock()

	d, ok := r.diagnoses[id]
	if !ok {
		return nil, goerr.New("diagnosis not found",
			goerr.T(errutil.TagNotFound),
			goerr.V("id", id))
	}
	cp := *d
	return &cp, nil
}

// ListDiagnoses returns paginated diagnoses ordered by CreatedAt DESC.
func (r *Memory) ListDiagnoses(_ context.Context, offset, limit int) ([]*diagnosis.Diagnosis, int, error) {
	r.diagnosisMu.RLock()
	defer r.diagnosisMu.RUnlock()

	all := make([]*diagnosis.Diagnosis, 0, len(r.diagnoses))
	for _, d := range r.diagnoses {
		cp := *d
		all = append(all, &cp)
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})

	total := len(all)
	if offset >= total {
		return []*diagnosis.Diagnosis{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

// PutDiagnosisIssue saves or updates a diagnosis issue.
func (r *Memory) PutDiagnosisIssue(_ context.Context, issue *diagnosis.Issue) error {
	r.diagnosisMu.Lock()
	defer r.diagnosisMu.Unlock()

	cp := *issue
	if r.diagnosisIssues[issue.DiagnosisID] == nil {
		r.diagnosisIssues[issue.DiagnosisID] = make(map[string]*diagnosis.Issue)
	}
	r.diagnosisIssues[issue.DiagnosisID][issue.ID] = &cp
	return nil
}

// ListDiagnosisIssues returns paginated issues for a diagnosis ordered by CreatedAt ASC.
func (r *Memory) ListDiagnosisIssues(_ context.Context, diagnosisID types.DiagnosisID, offset, limit int) ([]*diagnosis.Issue, int, error) {
	r.diagnosisMu.RLock()
	defer r.diagnosisMu.RUnlock()

	issueMap := r.diagnosisIssues[diagnosisID]
	all := make([]*diagnosis.Issue, 0, len(issueMap))
	for _, iss := range issueMap {
		cp := *iss
		all = append(all, &cp)
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt.Before(all[j].CreatedAt)
	})

	total := len(all)
	if offset >= total {
		return []*diagnosis.Issue{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

// GetDiagnosisIssue retrieves a specific issue by diagnosisID and issueID.
func (r *Memory) GetDiagnosisIssue(_ context.Context, diagnosisID types.DiagnosisID, issueID string) (*diagnosis.Issue, error) {
	r.diagnosisMu.RLock()
	defer r.diagnosisMu.RUnlock()

	issueMap, ok := r.diagnosisIssues[diagnosisID]
	if ok {
		if iss, ok := issueMap[issueID]; ok {
			cp := *iss
			return &cp, nil
		}
	}
	return nil, goerr.New("diagnosis issue not found",
		goerr.T(errutil.TagNotFound),
		goerr.V("diagnosis_id", diagnosisID),
		goerr.V("issue_id", issueID))
}

// CountDiagnosisIssues counts issues for a diagnosis, optionally filtered by status.
func (r *Memory) CountDiagnosisIssues(_ context.Context, diagnosisID types.DiagnosisID, status *diagnosis.IssueStatus) (int, error) {
	r.diagnosisMu.RLock()
	defer r.diagnosisMu.RUnlock()

	issueMap := r.diagnosisIssues[diagnosisID]
	if status == nil {
		return len(issueMap), nil
	}

	count := 0
	for _, iss := range issueMap {
		if iss.Status == *status {
			count++
		}
	}
	return count, nil
}

// ListPendingDiagnosisIssues returns all pending issues for a diagnosis.
func (r *Memory) ListPendingDiagnosisIssues(_ context.Context, diagnosisID types.DiagnosisID) ([]*diagnosis.Issue, error) {
	r.diagnosisMu.RLock()
	defer r.diagnosisMu.RUnlock()

	issueMap := r.diagnosisIssues[diagnosisID]
	var result []*diagnosis.Issue
	for _, iss := range issueMap {
		if iss.Status == diagnosis.IssueStatusPending {
			cp := *iss
			result = append(result, &cp)
		}
	}
	return result, nil
}
