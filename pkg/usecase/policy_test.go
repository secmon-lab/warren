package usecase_test

import (
	"context"
	"embed"
	"encoding/json"
	"testing"

	"cloud.google.com/go/vertexai/genai"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service/policy"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/utils/test"
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

	vars := test.NewEnvVars(t, "TEST_GEMINI_PROJECT_ID", "TEST_GEMINI_LOCATION")
	ai, err := genai.NewClient(ctx, vars.Get("TEST_GEMINI_PROJECT_ID"), vars.Get("TEST_GEMINI_LOCATION"))
	gt.NoError(t, err)
	geminiModel := ai.GenerativeModel("gemini-2.0-flash")
	geminiModel.GenerationConfig.ResponseMIMEType = "application/json"

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

	uc := usecase.New(func() interfaces.GenAIChatSession {
		return geminiModel.StartChat()
	}, usecase.WithPolicyService(policyService))

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
	policy, err := uc.GenerateIgnorePolicy(ctx, alerts, "")
	gt.NoError(t, err)
	gt.NotNil(t, policy)
}
