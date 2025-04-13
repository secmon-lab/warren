package policy_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/adapter/gemini"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/policy"
	"github.com/secmon-lab/warren/pkg/domain/model/source"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	svc "github.com/secmon-lab/warren/pkg/service/policy"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

var ignoreTestData = "testdata/ignore"

func loadJson(t *testing.T, baseDir, path string) map[string]any {
	t.Helper()
	fullPath := filepath.Join(baseDir, path)
	data, err := os.ReadFile(fullPath)
	gt.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(data, &result)
	gt.NoError(t, err)
	return result
}

func TestGenerateIgnorePolicy(t *testing.T) {
	ctx := t.Context()

	geminiClient := gemini.NewTestClient(t, gemini.WithContentType("application/json"))

	policyClient, err := opaq.New(opaq.Files("./testdata/ignore/policy"))
	gt.NoError(t, err)

	repo := repository.NewMemory()
	testDataSet := &policy.TestDataSet{
		Detect: &policy.TestData{
			Data: map[types.AlertSchema]map[string]any{
				"guardduty": {
					"alert/detect.json": loadJson(t, ignoreTestData, "alert/detect.json"),
				},
			},
		},
		Ignore: &policy.TestData{
			Data: map[types.AlertSchema]map[string]any{
				"guardduty": {
					"alert/ignore.json": loadJson(t, ignoreTestData, "alert/ignore.json"),
				},
			},
		},
	}

	alerts := []*alert.Alert{
		{
			Schema: "guardduty",
			ID:     "034f3664616c49cb85349d0511ecd306",
			Data:   loadJson(t, ignoreTestData, "alert/new.json"),
		},
	}

	ctx = msg.With(ctx, func(ctx context.Context, msg string) {
		t.Log(msg)
	}, nil)

	ssn := geminiClient.StartChat()
	input := svc.GenerateIgnorePolicyInput{
		Repo:         repo,
		Source:       source.Static(alerts),
		LLM:          ssn.SendMessage,
		PolicyClient: policyClient,
		TestDataSet:  testDataSet,
	}

	diff, err := svc.GenerateIgnorePolicy(ctx, input)
	gt.NoError(t, err)
	gt.NotNil(t, diff)
}

func TestFormatRegoPolicy(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "format indentation",
			input: `package example
	
	allow if {
	# no indent
	input.color == "red"
	}
	`,
			expected: `package example

allow if {
	# no indent
	input.color == "red"
}
`,
		},
		{
			name: "space to indent	",
			input: `package example

allow if {
    input.color == "red"
}
`,
			expected: `package example

allow if {
	input.color == "red"
}
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			contents, err := svc.FormatRegoPolicy("test.rego", []byte(tc.input))
			gt.NoError(t, err)
			gt.Value(t, string(contents)).Equal(tc.expected)
		})
	}
}
