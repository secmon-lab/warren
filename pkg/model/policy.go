package model

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

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

type TestDataSet struct {
	Detect *TestData `json:"detect"`
	Ignore *TestData `json:"ignore"`
}

func NewTestDataSet() *TestDataSet {
	return &TestDataSet{
		Detect: &TestData{},
		Ignore: &TestData{},
	}
}

type TestData struct {
	BasePath string
	Data     map[string]map[string]any
}

func (x *TestData) Add(schema string, filename string, data any) {
	if x.Data[schema] == nil {
		x.Data[schema] = make(map[string]any)
	}
	x.Data[schema][filename] = data
}

func (x *TestData) Clone() *TestData {
	clone := NewTestData()
	clone.BasePath = x.BasePath
	clone.Data = make(map[string]map[string]any)
	for schema, dataSets := range x.Data {
		clone.Data[schema] = make(map[string]any)
		for filename, data := range dataSets {
			clone.Data[schema][filename] = data
		}
	}
	return clone
}

func (x *TestData) Save(dir string) error {
	for schema, dataSets := range x.Data {
		for filename, data := range dataSets {
			jsonData, err := json.Marshal(data)
			if err != nil {
				return goerr.Wrap(err, "failed to marshal test data", goerr.V("schema", schema), goerr.V("filename", filename))
			}

			fpath := filepath.Join(x.BasePath, dir, schema, filename)
			if err := os.WriteFile(filepath.Clean(fpath), jsonData, 0644); err != nil {
				return goerr.Wrap(err, "failed to save test data", goerr.V("schema", schema), goerr.V("filename", filename))
			}
		}
	}

	return nil
}

func NewTestData() *TestData {
	return &TestData{
		Data: make(map[string]map[string]any),
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
	Hash      string            `firestore:"hash"`
	Data      map[string]string `firestore:"data"`
	CreatedAt time.Time         `firestore:"created_at"`
}

type PolicyDiffID string

func (x PolicyDiffID) String() string {
	return string(x)
}

func NewPolicyDiffID() PolicyDiffID {
	return PolicyDiffID(uuid.New().String())
}

type PolicyDiff struct {
	ID          PolicyDiffID      `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	CreatedAt   time.Time         `json:"created_at"`
	New         map[string]string `json:"new"`
	Old         map[string]string `json:"old"`
	TestDataSet *TestDataSet      `json:"test_data_set"`
}

func NewPolicyDiff(ctx context.Context, id PolicyDiffID, title, description string, new, old map[string]string, testDataSet *TestDataSet) *PolicyDiff {
	return &PolicyDiff{
		ID:          id,
		Title:       title,
		Description: description,
		New:         new,
		Old:         old,
		TestDataSet: testDataSet,
		CreatedAt:   clock.Now(ctx),
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
