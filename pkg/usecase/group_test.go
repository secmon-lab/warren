package usecase_test

import (
	"context"
	"embed"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/mock"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/usecase"
)

//go:embed testdata/group
var groupTestData embed.FS

func TestGroupUnclosedAlerts(t *testing.T) {
	geminiClient := genGeminiClient(t)
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
	uc := usecase.New(usecase.WithLLMClient(geminiClient), usecase.WithRepository(repo))

	gt.NoError(t, uc.GroupUnclosedAlerts(context.Background(), mockSlackThread)).Must()
	gt.A(t, mockSlackThread.PostAlertGroupsCalls()).Length(1)
	gt.A(t, mockSlackThread.PostAlertGroupsCalls()[0].AlertGroups).Longer(0)
	gt.A(t, mockSlackThread.PostAlertGroupsCalls()[0].AlertGroups[0].Alerts).Longer(0)
}
