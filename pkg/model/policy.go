package model

import (
	"log/slog"
	"sort"
	"time"
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
	Detect TestData `json:"detect"`
	Ignore TestData `json:"ignore"`
}

func NewTestDataSet() *TestDataSet {
	return &TestDataSet{
		Detect: make(TestData),
		Ignore: make(TestData),
	}
}

type TestData map[string]map[string]any

func (x TestData) LogValue() slog.Value {
	values := make([]slog.Attr, 0, len(x))

	for schema, dataSets := range x {
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
