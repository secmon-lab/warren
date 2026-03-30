package bigquery_test

import (
	"context"
	"strings"
	"testing"

	"github.com/m-mizutani/gt"
	bqagent "github.com/secmon-lab/warren/pkg/agents/bigquery"
)

func TestToolSet_ID(t *testing.T) {
	ts := bqagent.NewToolSetForTest(&bqagent.Config{
		Tables:        []bqagent.TableConfig{},
		ScanSizeLimit: 1000000,
	}, "test-project", "")

	gt.V(t, ts.ID()).Equal("bigquery")
}

func TestToolSet_Description(t *testing.T) {
	t.Run("without tables", func(t *testing.T) {
		ts := bqagent.NewToolSetForTest(&bqagent.Config{
			Tables:        []bqagent.TableConfig{},
			ScanSizeLimit: 1000000,
		}, "test-project", "")

		description := ts.Description()
		gt.V(t, description).NotEqual("")
		gt.True(t, strings.Contains(description, "BigQuery"))
		// Should not contain "Available tables:" when no tables configured
		gt.False(t, strings.Contains(description, "Available tables:"))
	})

	t.Run("with tables", func(t *testing.T) {
		ts := bqagent.NewToolSetForTest(&bqagent.Config{
			Tables: []bqagent.TableConfig{
				{
					ProjectID:   "test-project",
					DatasetID:   "test-dataset",
					TableID:     "test-table",
					Description: "Test table",
				},
			},
			ScanSizeLimit: 1000000,
		}, "test-project", "")

		description := ts.Description()
		gt.True(t, strings.Contains(description, "Available tables:"))
		gt.True(t, strings.Contains(description, "test-project.test-dataset.test-table"))
	})
}

func TestToolSet_Prompt(t *testing.T) {
	ts := bqagent.NewToolSetForTest(&bqagent.Config{
		Tables: []bqagent.TableConfig{
			{
				ProjectID:   "test-project",
				DatasetID:   "test-dataset",
				TableID:     "test-table",
				Description: "Test table",
			},
		},
		ScanSizeLimit: 1000000,
	}, "test-project", "")

	ctx := context.Background()
	prompt, err := ts.Prompt(ctx)
	gt.NoError(t, err)
	gt.True(t, len(prompt) > 0)
}

func TestToolSet_Specs(t *testing.T) {
	ts := bqagent.NewToolSetForTest(&bqagent.Config{
		Tables: []bqagent.TableConfig{
			{
				ProjectID:   "test-project",
				DatasetID:   "test-dataset",
				TableID:     "test-table",
				Description: "Test table",
			},
		},
		ScanSizeLimit: 1000000,
	}, "test-project", "")

	ctx := context.Background()
	specs, err := ts.Specs(ctx)
	gt.NoError(t, err)
	// Should have at least bigquery_query and bigquery_schema
	gt.N(t, len(specs)).GreaterOrEqual(2)

	// Verify expected tool names
	names := make(map[string]bool)
	for _, s := range specs {
		names[s.Name] = true
	}
	gt.True(t, names["bigquery_query"])
	gt.True(t, names["bigquery_schema"])
}
