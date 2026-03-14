package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	diagnosismodel "github.com/secmon-lab/warren/pkg/domain/model/diagnosis"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
)

func TestDiagnosisCRUD(t *testing.T) {
	repositories := []struct {
		name string
		repo func(t *testing.T) interfaces.Repository
	}{
		{
			name: "Memory",
			repo: func(t *testing.T) interfaces.Repository {
				return repository.NewMemory()
			},
		},
		{
			name: "Firestore",
			repo: func(t *testing.T) interfaces.Repository {
				return newFirestoreClient(t)
			},
		},
	}

	for _, repoTest := range repositories {
		t.Run(repoTest.name, func(t *testing.T) {
			repo := repoTest.repo(t)
			ctx := context.Background()

			t.Run("PutAndGetDiagnosis", func(t *testing.T) {
				now := time.Now().Truncate(time.Millisecond)
				d := &diagnosismodel.Diagnosis{
					ID:        types.NewDiagnosisID(),
					Status:    diagnosismodel.DiagnosisStatusPending,
					CreatedAt: now,
					UpdatedAt: now,
				}

				err := repo.PutDiagnosis(ctx, d)
				gt.NoError(t, err).Required()

				got, err := repo.GetDiagnosis(ctx, d.ID)
				gt.NoError(t, err).Required()
				gt.Value(t, got.ID).Equal(d.ID)
				gt.Value(t, got.Status).Equal(diagnosismodel.DiagnosisStatusPending)
			})

			t.Run("UpdateDiagnosisStatus", func(t *testing.T) {
				now := time.Now().Truncate(time.Millisecond)
				d := &diagnosismodel.Diagnosis{
					ID:        types.NewDiagnosisID(),
					Status:    diagnosismodel.DiagnosisStatusPending,
					CreatedAt: now,
					UpdatedAt: now,
				}
				gt.NoError(t, repo.PutDiagnosis(ctx, d)).Required()

				d.Status = diagnosismodel.DiagnosisStatusFixed
				d.UpdatedAt = time.Now()
				gt.NoError(t, repo.PutDiagnosis(ctx, d)).Required()

				got, err := repo.GetDiagnosis(ctx, d.ID)
				gt.NoError(t, err).Required()
				gt.Value(t, got.Status).Equal(diagnosismodel.DiagnosisStatusFixed)
			})

			t.Run("ListDiagnoses", func(t *testing.T) {
				base := time.Now()
				d1 := &diagnosismodel.Diagnosis{
					ID:        types.NewDiagnosisID(),
					Status:    diagnosismodel.DiagnosisStatusPending,
					CreatedAt: base.Add(-2 * time.Second),
					UpdatedAt: base.Add(-2 * time.Second),
				}
				d2 := &diagnosismodel.Diagnosis{
					ID:        types.NewDiagnosisID(),
					Status:    diagnosismodel.DiagnosisStatusFixed,
					CreatedAt: base.Add(-1 * time.Second),
					UpdatedAt: base.Add(-1 * time.Second),
				}

				gt.NoError(t, repo.PutDiagnosis(ctx, d1)).Required()
				gt.NoError(t, repo.PutDiagnosis(ctx, d2)).Required()

				results, total, err := repo.ListDiagnoses(ctx, 0, 100)
				gt.NoError(t, err).Required()
				gt.Number(t, total).GreaterOrEqual(2)
				gt.Number(t, len(results)).GreaterOrEqual(2)

				// Verify IDs are present
				idSet := map[types.DiagnosisID]bool{}
				for _, r := range results {
					idSet[r.ID] = true
				}
				gt.True(t, idSet[d1.ID])
				gt.True(t, idSet[d2.ID])
			})

			t.Run("GetDiagnosisNotFound", func(t *testing.T) {
				_, err := repo.GetDiagnosis(ctx, types.NewDiagnosisID())
				gt.Error(t, err)
			})
		})
	}
}

func TestDiagnosisIssueCRUD(t *testing.T) {
	repositories := []struct {
		name string
		repo func(t *testing.T) interfaces.Repository
	}{
		{
			name: "Memory",
			repo: func(t *testing.T) interfaces.Repository {
				return repository.NewMemory()
			},
		},
		{
			name: "Firestore",
			repo: func(t *testing.T) interfaces.Repository {
				return newFirestoreClient(t)
			},
		},
	}

	for _, repoTest := range repositories {
		t.Run(repoTest.name, func(t *testing.T) {
			repo := repoTest.repo(t)
			ctx := context.Background()

			// Create a parent diagnosis first
			now := time.Now().Truncate(time.Millisecond)
			diagID := types.NewDiagnosisID()
			diag := &diagnosismodel.Diagnosis{
				ID:        diagID,
				Status:    diagnosismodel.DiagnosisStatusPending,
				CreatedAt: now,
				UpdatedAt: now,
			}
			gt.NoError(t, repo.PutDiagnosis(ctx, diag)).Required()

			t.Run("PutAndGetIssue", func(t *testing.T) {
				issue := diagnosismodel.NewIssue(diagID, "missing_alert_embedding", "alert-001", "Embedding is empty")
				issue.CreatedAt = time.Now().Truncate(time.Millisecond)

				err := repo.PutDiagnosisIssue(ctx, &issue)
				gt.NoError(t, err).Required()

				got, err := repo.GetDiagnosisIssue(ctx, diagID, issue.ID)
				gt.NoError(t, err).Required()
				gt.Value(t, got.ID).Equal(issue.ID)
				gt.Value(t, got.DiagnosisID).Equal(diagID)
				gt.Value(t, got.RuleID).Equal(diagnosismodel.RuleID("missing_alert_embedding"))
				gt.Value(t, got.TargetID).Equal("alert-001")
				gt.Value(t, got.Description).Equal("Embedding is empty")
				gt.Value(t, got.Status).Equal(diagnosismodel.IssueStatusPending)
			})

			t.Run("UpdateIssueStatus", func(t *testing.T) {
				issue := diagnosismodel.NewIssue(diagID, "legacy_alert_status", "alert-002", "Status is unbound")
				issue.CreatedAt = time.Now().Truncate(time.Millisecond)

				gt.NoError(t, repo.PutDiagnosisIssue(ctx, &issue)).Required()

				fixedAt := time.Now()
				issue.Status = diagnosismodel.IssueStatusFixed
				issue.FixedAt = &fixedAt
				gt.NoError(t, repo.PutDiagnosisIssue(ctx, &issue)).Required()

				got, err := repo.GetDiagnosisIssue(ctx, diagID, issue.ID)
				gt.NoError(t, err).Required()
				gt.Value(t, got.Status).Equal(diagnosismodel.IssueStatusFixed)
				gt.Value(t, got.FixedAt).NotNil()
			})

			t.Run("ListDiagnosisIssues", func(t *testing.T) {
				listDiagID := types.NewDiagnosisID()
				listDiag := &diagnosismodel.Diagnosis{
					ID:        listDiagID,
					Status:    diagnosismodel.DiagnosisStatusPending,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				gt.NoError(t, repo.PutDiagnosis(ctx, listDiag)).Required()

				for i := range 3 {
					issue := diagnosismodel.NewIssue(listDiagID, "test_rule", "target-"+string(rune('A'+i)), "desc")
					issue.CreatedAt = time.Now().Add(time.Duration(i) * time.Millisecond)
					gt.NoError(t, repo.PutDiagnosisIssue(ctx, &issue)).Required()
				}

				issues, total, err := repo.ListDiagnosisIssues(ctx, listDiagID, 0, 10)
				gt.NoError(t, err).Required()
				gt.Number(t, total).Equal(3)
				gt.Number(t, len(issues)).Equal(3)
			})

			t.Run("CountDiagnosisIssues", func(t *testing.T) {
				countDiagID := types.NewDiagnosisID()
				countDiag := &diagnosismodel.Diagnosis{
					ID:        countDiagID,
					Status:    diagnosismodel.DiagnosisStatusPending,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				gt.NoError(t, repo.PutDiagnosis(ctx, countDiag)).Required()

				// Add 2 pending, 1 fixed
				for i := range 2 {
					issue := diagnosismodel.NewIssue(countDiagID, "test_rule", "target-"+string(rune('X'+i)), "desc")
					issue.CreatedAt = time.Now()
					gt.NoError(t, repo.PutDiagnosisIssue(ctx, &issue)).Required()
				}
				fixedIssue := diagnosismodel.NewIssue(countDiagID, "test_rule", "target-Z", "desc fixed")
				fixedIssue.CreatedAt = time.Now()
				fixedIssue.Status = diagnosismodel.IssueStatusFixed
				gt.NoError(t, repo.PutDiagnosisIssue(ctx, &fixedIssue)).Required()

				// Count all
				totalCount, err := repo.CountDiagnosisIssues(ctx, countDiagID, nil)
				gt.NoError(t, err).Required()
				gt.Number(t, totalCount).Equal(3)

				// Count pending
				pendingStatus := diagnosismodel.IssueStatusPending
				pendingCount, err := repo.CountDiagnosisIssues(ctx, countDiagID, &pendingStatus)
				gt.NoError(t, err).Required()
				gt.Number(t, pendingCount).Equal(2)

				// Count fixed
				fixedStatus := diagnosismodel.IssueStatusFixed
				fixedCount, err := repo.CountDiagnosisIssues(ctx, countDiagID, &fixedStatus)
				gt.NoError(t, err).Required()
				gt.Number(t, fixedCount).Equal(1)
			})

			t.Run("ListPendingDiagnosisIssues", func(t *testing.T) {
				pendDiagID := types.NewDiagnosisID()
				pendDiag := &diagnosismodel.Diagnosis{
					ID:        pendDiagID,
					Status:    diagnosismodel.DiagnosisStatusPending,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				gt.NoError(t, repo.PutDiagnosis(ctx, pendDiag)).Required()

				pending1 := diagnosismodel.NewIssue(pendDiagID, "rule_a", "t1", "desc")
				pending1.CreatedAt = time.Now()
				gt.NoError(t, repo.PutDiagnosisIssue(ctx, &pending1)).Required()

				fixed1 := diagnosismodel.NewIssue(pendDiagID, "rule_b", "t2", "desc")
				fixed1.Status = diagnosismodel.IssueStatusFixed
				fixed1.CreatedAt = time.Now()
				gt.NoError(t, repo.PutDiagnosisIssue(ctx, &fixed1)).Required()

				results, err := repo.ListPendingDiagnosisIssues(ctx, pendDiagID)
				gt.NoError(t, err).Required()
				gt.Number(t, len(results)).Equal(1)
				gt.Value(t, results[0].ID).Equal(pending1.ID)
				gt.Value(t, results[0].Status).Equal(diagnosismodel.IssueStatusPending)
			})

			t.Run("GetDiagnosisIssueNotFound", func(t *testing.T) {
				_, err := repo.GetDiagnosisIssue(ctx, diagID, "nonexistent-id")
				gt.Error(t, err)
			})
		})
	}
}
