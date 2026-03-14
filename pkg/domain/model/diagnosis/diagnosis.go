package diagnosis

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

// RuleID is the unique identifier for a diagnosis rule.
// It is stored as Issue.RuleID to identify which rule detected and can fix an issue.
type RuleID string

func (r RuleID) String() string {
	return string(r)
}

// IssueStatus represents the fix status of an issue.
type IssueStatus string

const (
	IssueStatusPending IssueStatus = "pending"
	IssueStatusFixed   IssueStatus = "fixed"
	IssueStatusFailed  IssueStatus = "failed"
)

// DiagnosisStatus represents the overall status of a diagnosis run.
type DiagnosisStatus string

const (
	DiagnosisStatusPending        DiagnosisStatus = "pending"
	DiagnosisStatusFixed          DiagnosisStatus = "fixed"
	DiagnosisStatusPartiallyFixed DiagnosisStatus = "partially_fixed"
)

// Issue represents a single detected inconsistency.
// Stored as a subcollection: diagnoses/{diagnosisID}/issues/{issueID}
// TargetID interpretation depends on RuleID (AlertID or TicketID, etc.).
type Issue struct {
	ID          string            `json:"id" firestore:"ID"`
	DiagnosisID types.DiagnosisID `json:"diagnosis_id" firestore:"DiagnosisID"`
	RuleID      RuleID            `json:"rule_id" firestore:"RuleID"`
	TargetID    string            `json:"target_id" firestore:"TargetID"`
	Description string            `json:"description" firestore:"Description"`
	Status      IssueStatus       `json:"status" firestore:"Status"`
	FixedAt     *time.Time        `json:"fixed_at,omitempty" firestore:"FixedAt"`
	FailReason  string            `json:"fail_reason,omitempty" firestore:"FailReason"`
	CreatedAt   time.Time         `json:"created_at" firestore:"CreatedAt"`
}

func NewIssue(diagnosisID types.DiagnosisID, ruleID RuleID, targetID, description string) Issue {
	return Issue{
		ID:          uuid.New().String(),
		DiagnosisID: diagnosisID,
		RuleID:      ruleID,
		TargetID:    targetID,
		Description: description,
		Status:      IssueStatusPending,
	}
}

// IssueCounts holds aggregated issue counts for a diagnosis broken down by status.
type IssueCounts struct {
	Total   int
	Pending int
	Fixed   int
	Failed  int
}

// Diagnosis represents a single diagnosis run header.
// Count fields are not stored here; they are derived from subcollection aggregation queries.
type Diagnosis struct {
	ID        types.DiagnosisID `json:"id" firestore:"ID"`
	Status    DiagnosisStatus   `json:"status" firestore:"Status"`
	CreatedAt time.Time         `json:"created_at" firestore:"CreatedAt"`
	UpdatedAt time.Time         `json:"updated_at" firestore:"UpdatedAt"`
}

func New(ctx context.Context) *Diagnosis {
	now := clock.Now(ctx)
	return &Diagnosis{
		ID:        types.NewDiagnosisID(),
		Status:    DiagnosisStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
