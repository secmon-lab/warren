package usecase_test

import (
	"context"
	"embed"
	"encoding/json"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service/policy"
	"github.com/secmon-lab/warren/pkg/service/source"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/utils/thread"
)

//go:embed testdata/ignore
var ignoreTestData embed.FS

func loadJson(t *testing.T, fd embed.FS, name string) any {
	data, err := fd.ReadFile(name)
	gt.NoError(t, err)
	var v any
	err = json.Unmarshal(data, &v)
	gt.NoError(t, err)
	return v
}

func TestGenerateIgnorePolicy(t *testing.T) {
	ctx := t.Context()

	geminiClient := genGeminiClient(t)
	policyClient, err := opaq.New(opaq.Files("./testdata/ignore/policy"))
	gt.NoError(t, err)

	repo := repository.NewMemory()
	policyService := policy.New(repo, policyClient, &model.TestDataSet{
		Detect: &model.TestData{
			Data: map[string]map[string]any{
				"guardduty": {
					"alert/detect.json": loadJson(t, ignoreTestData, "testdata/ignore/alert/detect.json"),
				},
			},
		},
		Ignore: &model.TestData{
			Data: map[string]map[string]any{
				"guardduty": {
					"alert/ignore.json": loadJson(t, ignoreTestData, "testdata/ignore/alert/ignore.json"),
				},
			},
		},
	})

	errs := policyService.Test(ctx)
	gt.A(t, errs).Length(0)

	uc := usecase.New(usecase.WithLLMClient(geminiClient), usecase.WithPolicyService(policyService))

	alerts := []model.Alert{
		{
			Schema: "guardduty",
			ID:     "034f3664616c49cb85349d0511ecd306",
			Data:   loadJson(t, ignoreTestData, "testdata/ignore/alert/new.json"),
		},
	}

	ctx = thread.WithReplyFunc(ctx, func(ctx context.Context, msg string) {
		t.Log(msg)
	})
	policy, err := uc.GenerateIgnorePolicy(ctx, source.Static(alerts), "")
	gt.NoError(t, err)
	gt.NotNil(t, policy)
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
			contents, err := usecase.FormatRegoPolicy("test.rego", []byte(tc.input))
			gt.NoError(t, err)
			gt.Equal(t, string(contents), tc.expected)
		})
	}
}
