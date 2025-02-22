package model

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/m-mizutani/goerr/v2"
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
