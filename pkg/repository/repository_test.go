package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/utils/test"
)

func TestMemory(t *testing.T) {
	repo := repository.NewMemory()
	testRepository(t, repo)
}

func TestFirestore(t *testing.T) {
	vars := test.NewEnvVars(t, "TEST_FIRESTORE_PROJECT_ID", "TEST_FIRESTORE_DATABASE_ID")
	repo, err := repository.NewFirestore(context.Background(),
		vars.Get("TEST_FIRESTORE_PROJECT_ID"),
		vars.Get("TEST_FIRESTORE_DATABASE_ID"),
	)
	gt.NoError(t, err)
	testRepository(t, repo)
}

func testRepository(t *testing.T, repo interfaces.Repository) {
	ctx := context.Background()

	t.Run("PutAlert", func(t *testing.T) {
		alert := model.NewAlert(ctx, "test", model.PolicyAlert{
			Title: "test",
			Attrs: []model.Attribute{
				{
					Key:   "test",
					Value: "test",
				},
			},
			Data: map[string]any{
				"test": "test",
			},
		})
		gt.NoError(t, repo.PutAlert(ctx, alert))

		got, err := repo.GetAlert(ctx, alert.ID)
		gt.NoError(t, err)
		gt.Equal(t, alert.ID, got.ID)
		gt.Equal(t, alert.Title, got.Title)
		gt.Equal(t, alert.Attributes, got.Attributes)
		gt.Equal(t, alert.Data, got.Data)
	})

	t.Run("FetchLatestAlerts", func(t *testing.T) {
		var alerts []model.Alert
		now := time.Now()
		for i := 0; i < 10; i++ {
			newAlert := model.NewAlert(ctx, "test", model.PolicyAlert{
				Title: "test",
				Attrs: []model.Attribute{
					{Key: "test", Value: "test"},
				},
				Data: map[string]any{
					"test": "test",
				},
			})
			newAlert.CreatedAt = now.Add(time.Duration(i) * time.Second)
			alerts = append(alerts, newAlert)
		}
		for _, alert := range alerts {
			gt.NoError(t, repo.PutAlert(ctx, alert))
		}

		got, err := repo.FetchLatestAlerts(ctx, now.Add(-24*time.Hour), 5)
		gt.NoError(t, err)
		gt.Equal(t, len(got), 5)
		for i, alert := range got {
			gt.True(t, alert.CreatedAt.After(now.Add(-24*time.Hour)))
			gt.Equal(t, alert.ID, alerts[len(alerts)-i-1].ID)
		}
	})

	t.Run("GetAlertBySlackMessageID", func(t *testing.T) {
		alert := model.NewAlert(ctx, "test", model.PolicyAlert{
			Title: "test",
			Attrs: []model.Attribute{
				{Key: "test", Value: "test"},
			},
			Data: map[string]any{
				"test": "test",
			},
		})
		alert.SlackThread = &model.SlackThread{
			ChannelID: "test",
			ThreadID:  uuid.New().String(),
		}
		gt.NoError(t, repo.PutAlert(ctx, alert))

		got, err := repo.GetAlertBySlackThread(ctx, *alert.SlackThread)
		gt.NoError(t, err)
		gt.Equal(t, alert.ID, got.ID)
	})

	t.Run("GetAlertBySlackMessageID_NotFound", func(t *testing.T) {
		got, err := repo.GetAlertBySlackThread(ctx, model.SlackThread{
			ChannelID: "test",
			ThreadID:  uuid.New().String(),
		})
		gt.Error(t, err)
		gt.Nil(t, got)
	})
}
