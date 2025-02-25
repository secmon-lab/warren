package usecase_test

import (
	"context"
	"embed"
	"testing"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/mock"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/utils/test"
)

func buildGenAIModelForTest(t *testing.T) *genai.GenerativeModel {
	vars := test.NewEnvVars(t, "TEST_GEMINI_PROJECT_ID", "TEST_GEMINI_LOCATION")
	ai, err := genai.NewClient(t.Context(), vars.Get("TEST_GEMINI_PROJECT_ID"), vars.Get("TEST_GEMINI_LOCATION"))
	gt.NoError(t, err)
	geminiModel := ai.GenerativeModel("gemini-2.0-flash")
	geminiModel.GenerationConfig.ResponseMIMEType = "application/json"
	return geminiModel
}

//go:embed testdata/group
var groupTestData embed.FS

func TestGroupUnclosedAlerts(t *testing.T) {
	genaiModel := buildGenAIModelForTest(t)
	repo := repository.NewMemory()

	alerts := []model.Alert{
		model.NewAlert(context.Background(), "guardduty", model.PolicyAlert{
			Title: "Test Alert Title",
			Data:  loadJson(t, groupTestData, "testdata/group/alert1.json"),
		}),
		model.NewAlert(context.Background(), "guardduty", model.PolicyAlert{
			Title: "Test Alert Title 2",
			Data:  loadJson(t, groupTestData, "testdata/group/alert2.json"),
		}),
		model.NewAlert(context.Background(), "guardduty", model.PolicyAlert{
			Title: "Test Alert Title 3",
			Data:  loadJson(t, groupTestData, "testdata/group/alert3.json"),
		}),
	}

	for _, alert := range alerts {
		gt.NoError(t, repo.PutAlert(context.Background(), alert)).Must()
	}

	mockSlackThread := &mock.SlackThreadServiceMock{
		PostAlertGroupsFunc: func(ctx context.Context, alertGroups []model.AlertGroup) error {
			gt.A(t, alertGroups).Length(2).
				At(0, func(t testing.TB, v model.AlertGroup) {
					gt.A(t, v.AlertIDs).Length(2).Have(alerts[0].ID).Have(alerts[1].ID)
				}).
				At(1, func(t testing.TB, v model.AlertGroup) {
					gt.A(t, v.AlertIDs).Length(1).Have(alerts[2].ID)
				})
			return nil
		},
	}
	uc := usecase.New(func() interfaces.GenAIChatSession {
		return genaiModel.StartChat()
	}, usecase.WithRepository(repo))

	gt.NoError(t, uc.GroupUnclosedAlerts(context.Background(), mockSlackThread)).Must()
	gt.A(t, mockSlackThread.PostAlertGroupsCalls()).Length(1)
	gt.A(t, mockSlackThread.PostAlertGroupsCalls()[0].AlertGroups).Longer(0)
	gt.A(t, mockSlackThread.PostAlertGroupsCalls()[0].AlertGroups[0].Alerts).Longer(0)
}
