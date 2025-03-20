package policy

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

/*
type PolicyResult struct {
	Alert []PolicyAlert `json:"alert"`
}

type PolicyAlert struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Attrs       []Attribute `json:"attrs"`
	Data        any         `json:"data"`
}

type PolicyAuth struct {
	Allow bool `json:"allow"`
}
*/

type TestDataSet struct {
	Detect *TestData `json:"detect"`
	Ignore *TestData `json:"ignore"`
}

func NewTestDataSet() *TestDataSet {
	return &TestDataSet{
		Detect: NewTestData(),
		Ignore: NewTestData(),
	}
}

type TestData struct {
	Metafiles map[string]map[string]string
	Data      map[string]map[string]any
}

func (x *TestData) Add(schema string, filename string, data any) {
	if x.Data[schema] == nil {
		x.Data[schema] = make(map[string]any)
	}
	x.Data[schema][filename] = data
}

func (x *TestData) Clone() *TestData {
	clone := NewTestData()
	clone.Data = make(map[string]map[string]any)
	for schema, dataSets := range x.Data {
		clone.Data[schema] = make(map[string]any)
		for filename, data := range dataSets {
			clone.Data[schema][filename] = data
		}
	}
	clone.Metafiles = make(map[string]map[string]string)
	for schema, metafiles := range x.Metafiles {
		clone.Metafiles[schema] = make(map[string]string)
		for filename, content := range metafiles {
			clone.Metafiles[schema][filename] = content
		}
	}
	return clone
}

func NewTestData() *TestData {
	return &TestData{
		Metafiles: make(map[string]map[string]string),
		Data:      make(map[string]map[string]any),
	}
}

func (x TestData) LogValue() slog.Value {
	values := make([]slog.Attr, 0, len(x.Data))

	for schema, dataSets := range x.Data {
		files := []string{}
		for filename := range dataSets {
			files = append(files, filename)
		}
		sort.Strings(files)

		values = append(values, slog.Any(schema, files))
	}

	return slog.GroupValue(values...)
}

type PolicyData struct {
	Data      map[string]string `json:"data"`
	CreatedAt time.Time         `json:"created_at"`
}

type PolicyDiffID string

func (x PolicyDiffID) String() string {
	return string(x)
}

func NewPolicyDiffID() PolicyDiffID {
	return PolicyDiffID(uuid.New().String())
}

type PolicyDiff struct {
	ID             PolicyDiffID      `json:"id"`
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	CreatedAt      time.Time         `json:"created_at"`
	New            map[string]string `json:"new"`
	Old            map[string]string `json:"old"`
	NewTestDataSet *TestDataSet      `json:"new_test_data_set"`
}

func NewPolicyDiff(ctx context.Context, id PolicyDiffID, title, description string, new, old map[string]string, newTestDataSet *TestDataSet) *PolicyDiff {
	return &PolicyDiff{
		ID:             id,
		Title:          title,
		Description:    description,
		New:            new,
		Old:            old,
		NewTestDataSet: newTestDataSet,
		CreatedAt:      clock.Now(ctx),
	}
}

func (x *PolicyDiff) DiffPolicy() map[string]string {
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
