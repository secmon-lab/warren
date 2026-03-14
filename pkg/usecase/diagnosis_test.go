package usecase

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	diagnosismodel "github.com/secmon-lab/warren/pkg/domain/model/diagnosis"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
)

func TestRunDiagnosis_Basic(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()

	uc := New(WithRepository(repo))

	diag, err := uc.RunDiagnosis(ctx)
	gt.NoError(t, err)
	gt.Value(t, diag).NotNil()
	gt.Value(t, string(diag.ID)).NotEqual("")
	gt.Value(t, string(diag.Status)).Equal(string(diagnosismodel.DiagnosisStatusPending))

	// Verify the diagnosis was persisted
	stored, err := repo.GetDiagnosis(ctx, diag.ID)
	gt.NoError(t, err)
	gt.Value(t, stored.ID).Equal(diag.ID)
}

func TestGetDiagnoses_Pagination(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	uc := New(WithRepository(repo))

	// Run 3 diagnoses
	for i := 0; i < 3; i++ {
		_, err := uc.RunDiagnosis(ctx)
		gt.NoError(t, err)
	}

	list, total, err := uc.GetDiagnoses(ctx, 0, 10)
	gt.NoError(t, err)
	gt.Value(t, total).Equal(3)
	gt.Array(t, list).Length(3)

	// Pagination
	page1, total1, err := uc.GetDiagnoses(ctx, 0, 2)
	gt.NoError(t, err)
	gt.Value(t, total1).Equal(3)
	gt.Array(t, page1).Length(2)

	page2, _, err := uc.GetDiagnoses(ctx, 2, 2)
	gt.NoError(t, err)
	gt.Array(t, page2).Length(1)
}

func TestGetDiagnosis_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	uc := New(WithRepository(repo))

	_, err := uc.GetDiagnosis(ctx, types.DiagnosisID("nonexistent"))
	gt.Error(t, err)
}

func TestCountDiagnosisIssues(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	uc := New(WithRepository(repo))

	diag, err := uc.RunDiagnosis(ctx)
	gt.NoError(t, err)

	total, pending, fixed, failed, err := uc.CountDiagnosisIssues(ctx, diag.ID)
	gt.NoError(t, err)
	gt.Value(t, total).Equal(pending + fixed + failed)
}

func TestFixDiagnosis_NoPendingIssues(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	uc := New(WithRepository(repo))

	// Create a diagnosis with no pending issues
	diag, err := uc.RunDiagnosis(ctx)
	gt.NoError(t, err)

	// Manually clear any pending issues by marking them fixed
	pendingIssues, err := repo.ListPendingDiagnosisIssues(ctx, diag.ID)
	gt.NoError(t, err)
	for _, issue := range pendingIssues {
		issue.Status = diagnosismodel.IssueStatusFixed
		err = repo.PutDiagnosisIssue(ctx, issue)
		gt.NoError(t, err)
	}

	// FixDiagnosis should succeed with no pending issues
	result, err := uc.FixDiagnosis(ctx, diag.ID)
	gt.NoError(t, err)
	gt.Value(t, result).NotNil()
}

func TestGetDiagnosisIssues(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	uc := New(WithRepository(repo))

	diag, err := uc.RunDiagnosis(ctx)
	gt.NoError(t, err)

	issues, total, err := uc.GetDiagnosisIssues(ctx, diag.ID, 0, 100)
	gt.NoError(t, err)
	gt.Value(t, total).Equal(len(issues))
}
