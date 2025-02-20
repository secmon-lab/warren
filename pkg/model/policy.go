package model

import (
	"log/slog"
	"sort"
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
