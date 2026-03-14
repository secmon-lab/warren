package diagnosis

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	diagnosismodel "github.com/secmon-lab/warren/pkg/domain/model/diagnosis"
)

// Rule represents a single diagnosis rule.
// The RuleID returned by ID() is stored as Issue.RuleID, establishing a 1:1 mapping.
type Rule interface {
	// ID returns the unique identifier for this rule.
	ID() diagnosismodel.RuleID
	// Check detects issues in the repository and returns them.
	Check(ctx context.Context, repo interfaces.Repository) ([]diagnosismodel.Issue, error)
	// Fix repairs a single issue detected by this rule.
	Fix(ctx context.Context, repo interfaces.Repository, issue diagnosismodel.Issue) error
}
