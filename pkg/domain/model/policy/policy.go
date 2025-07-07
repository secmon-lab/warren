package policy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

// Contents is a map of file paths to their policy data (Rego).
type Contents map[string]string

// PolicyDiffID is a unique identifier for a policy diff.
type PolicyDiffID string

func (x PolicyDiffID) String() string {
	return string(x)
}

func NewPolicyDiffID() PolicyDiffID {
	return PolicyDiffID(uuid.New().String())
}

type Diff struct {
	ID             PolicyDiffID      `json:"id"`
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	CreatedAt      time.Time         `json:"created_at"`
	New            map[string]string `json:"new"`
	Old            map[string]string `json:"old"`
	NewTestDataSet *TestDataSet      `json:"new_test_data_set"`
}

func NewDiff(ctx context.Context, title, description string, new, old map[string]string, newTestDataSet *TestDataSet) *Diff {
	return &Diff{
		ID:             NewPolicyDiffID(),
		Title:          title,
		Description:    description,
		New:            new,
		Old:            old,
		NewTestDataSet: newTestDataSet,
		CreatedAt:      clock.Now(ctx),
	}
}

func (x *Diff) DiffPolicy() map[string]string {
	return diffPolicy(x.Old, x.New)
}

func diffPolicy(oldPolicy, newPolicy map[string]string) map[string]string {
	result := make(map[string]string)

	// Handle deleted files
	for fileName := range oldPolicy {
		if _, exists := newPolicy[fileName]; !exists {
			lines := strings.Split(strings.TrimSpace(oldPolicy[fileName]), "\n")
			var diff strings.Builder
			for _, line := range lines {
				fmt.Fprintf(&diff, "- %s\n", line)
			}
			result[fileName] = diff.String()
		}
	}

	// Handle new and modified files
	for fileName, newContent := range newPolicy {
		if oldContent, exists := oldPolicy[fileName]; exists {
			if oldContent != newContent {
				// File was modified
				oldLines := strings.Split(strings.TrimSpace(oldContent), "\n")
				newLines := strings.Split(strings.TrimSpace(newContent), "\n")

				var diff strings.Builder
				var i, j int
				for i < len(oldLines) && j < len(newLines) {
					if oldLines[i] == newLines[j] {
						// Lines match, keep going
						fmt.Fprintf(&diff, "  %s\n", oldLines[i])
						i++
						j++
					} else {
						// Find next matching line
						matchFound := false
						for k := 1; k < 3; k++ {
							if i+k < len(oldLines) && oldLines[i+k] == newLines[j] {
								// Found match ahead in old lines - lines were deleted
								for x := 0; x < k; x++ {
									fmt.Fprintf(&diff, "- %s\n", oldLines[i+x])
								}
								i += k
								matchFound = true
								break
							}
							if j+k < len(newLines) && oldLines[i] == newLines[j+k] {
								// Found match ahead in new lines - lines were added
								for x := 0; x < k; x++ {
									fmt.Fprintf(&diff, "+ %s\n", newLines[j+x])
								}
								j += k
								matchFound = true
								break
							}
						}
						if !matchFound {
							// No nearby match found - line was modified
							fmt.Fprintf(&diff, "- %s\n+ %s\n", oldLines[i], newLines[j])
							i++
							j++
						}
					}
				}

				// Handle remaining lines
				for ; i < len(oldLines); i++ {
					fmt.Fprintf(&diff, "- %s\n", oldLines[i])
				}
				for ; j < len(newLines); j++ {
					fmt.Fprintf(&diff, "+ %s\n", newLines[j])
				}

				result[fileName] = diff.String()
			}
		} else {
			// New file added
			lines := strings.Split(strings.TrimSpace(newContent), "\n")
			var diff strings.Builder
			for _, line := range lines {
				fmt.Fprintf(&diff, "+ %s\n", line)
			}
			result[fileName] = diff.String()
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}
