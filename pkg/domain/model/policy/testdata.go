package policy

import (
	"context"
	"errors"
	"log/slog"
	"sort"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

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
	// Readme is the metadata (README) of the test data that is generated by the AI.
	Readme map[types.AlertSchema]map[string]string
	// Data is the test data that is from the original alert or modified.
	Data map[types.AlertSchema]map[string]any
}

func (x *TestData) Add(schema types.AlertSchema, filename string, data any) {
	if x.Data[schema] == nil {
		x.Data[schema] = make(map[string]any)
	}
	x.Data[schema][filename] = data
}

func (x *TestData) Clone() *TestData {
	clone := NewTestData()
	clone.Data = make(map[types.AlertSchema]map[string]any)
	for schema, dataSets := range x.Data {
		clone.Data[schema] = make(map[string]any)
		for filename, data := range dataSets {
			clone.Data[schema][filename] = data
		}
	}
	clone.Readme = make(map[types.AlertSchema]map[string]string)
	for schema, readmes := range x.Readme {
		clone.Readme[schema] = make(map[string]string)
		for filename, content := range readmes {
			clone.Readme[schema][filename] = content
		}
	}
	return clone
}

func NewTestData() *TestData {
	return &TestData{
		Readme: make(map[types.AlertSchema]map[string]string),
		Data:   make(map[types.AlertSchema]map[string]any),
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

		values = append(values, slog.Any(string(schema), files))
	}

	return slog.GroupValue(values...)
}

type QueryFunc func(ctx context.Context, query string, data any, result any, queryOptions ...opaq.QueryOption) error

func (x *TestDataSet) Test(ctx context.Context, queryFunc QueryFunc) []error {
	errors := []error{}

	errors = append(errors, test(ctx, queryFunc, x.Detect, true)...)
	errors = append(errors, test(ctx, queryFunc, x.Ignore, false)...)

	return errors
}

func test(ctx context.Context, queryFunc QueryFunc, testData *TestData, shouldDetect bool) []error {
	var results []error
	for schema, dataSet := range testData.Data {
		for filename, testData := range dataSet {
			var resp alert.QueryOutput

			if err := queryFunc(ctx, "data.alert."+schema.String(), testData, &resp); err != nil {
				if errors.Is(err, opaq.ErrNoEvalResult) {
					if shouldDetect {
						results = append(results, goerr.Wrap(err, "should be detected, but not detected",
							goerr.V("schema", schema),
							goerr.V("filename", filename),
							goerr.T(errs.TagTestFailed)))
					}
					continue
				}

				if len(resp.Alert) == 0 && shouldDetect {
					results = append(results, goerr.Wrap(err, "should be detected, but not detected",
						goerr.V("schema", schema),
						goerr.V("filename", filename),
						goerr.T(errs.TagTestFailed)))
					continue
				}

				if len(resp.Alert) > 0 && !shouldDetect {
					results = append(results, goerr.Wrap(err, "should be ignored, but detected",
						goerr.V("schema", schema),
						goerr.V("filename", filename),
						goerr.T(errs.TagTestFailed)))
				}
			}

			if !shouldDetect && len(resp.Alert) > 0 {
				results = append(results, goerr.New("should be ignored, but detected",
					goerr.V("schema", schema),
					goerr.V("filename", filename),
					goerr.T(errs.TagTestFailed)))
			}
		}
	}

	return results
}
