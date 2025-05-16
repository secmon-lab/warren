package llm_test

import (
	"context"
	_ "embed"
	"encoding/json"
	"os"
	"testing"

	"github.com/m-mizutani/gollem/llm/gemini"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/service/llm"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

//go:embed testdata/schema.json
var testSchemaData []byte

func TestSummaryWithSchemaData(t *testing.T) {
	ctx := t.Context()
	ctx = msg.With(ctx,
		func(ctx context.Context, msg string) {
			t.Log("Notify", msg)
		},
		func(ctx context.Context, msg string) func(ctx context.Context, msg string) {
			t.Log("NewTrace", msg)
			return func(ctx context.Context, msg string) {
				t.Log("Trace", msg)
			}
		},
	)

	projectID, ok := os.LookupEnv("TEST_GEMINI_PROJECT_ID")
	if !ok {
		t.Skip("GEMINI_PROJECT_ID is not set")
	}
	location, ok := os.LookupEnv("TEST_GEMINI_LOCATION")
	if !ok {
		t.Skip("GEMINI_LOCATION is not set")
	}

	geminiClient, err := gemini.New(ctx, projectID, location)
	gt.NoError(t, err).Required()

	var schema []any
	gt.NoError(t, json.Unmarshal(testSchemaData, &schema))

	prompt := `
	# Instruction
	The provided data is a BigQuery table schema. We will analyze it to determine if there are any security impacts when security alerts occur. Based on the summary, we will construct queries for analysis. Therefore, please list field names accurately. Please select fields that could be useful for security analysis.

	# About the table
	This is a Google Cloud audit log table. The Activity logs record changes related to resources.
	`
	summary, err := llm.Summary(ctx, geminiClient, prompt, schema, llm.WithMaxPartSize(100000))
	gt.NoError(t, err).Required()

	t.Log(summary)
}
