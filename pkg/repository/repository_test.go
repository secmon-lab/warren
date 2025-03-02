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

	t.Run("GetLatestAlerts", func(t *testing.T) {
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

		got, err := repo.GetLatestAlerts(ctx, now.Add(-24*time.Hour), 5)
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
		gt.NoError(t, err)
		gt.Nil(t, got)
	})

	t.Run("InsertAlertComment_and_GetAlertComments", func(t *testing.T) {
		alert := model.NewAlert(ctx, "test", model.PolicyAlert{
			Title: "test",
			Attrs: []model.Attribute{
				{Key: "test", Value: "test"},
			},
		})
		gt.NoError(t, repo.PutAlert(ctx, alert))

		comment1 := model.AlertComment{
			AlertID:   alert.ID,
			Comment:   "test1",
			Timestamp: time.Now().Format(time.RFC3339),
			UserID:    "orange",
		}
		gt.NoError(t, repo.InsertAlertComment(ctx, comment1))

		comment2 := model.AlertComment{
			AlertID:   alert.ID,
			Comment:   "test2",
			Timestamp: time.Now().Add(time.Second).Format(time.RFC3339),
			UserID:    "blue",
		}
		gt.NoError(t, repo.InsertAlertComment(ctx, comment2))

		got, err := repo.GetAlertComments(ctx, alert.ID)
		gt.NoError(t, err)
		gt.Equal(t, len(got), 2)
		gt.Equal(t, got[0].AlertID, alert.ID)
		gt.Equal(t, got[0].Comment, comment2.Comment)
		gt.Equal(t, got[0].Timestamp, comment2.Timestamp)
		gt.Equal(t, got[0].UserID, comment2.UserID)
		gt.Equal(t, got[1].AlertID, alert.ID)
		gt.Equal(t, got[1].Comment, comment1.Comment)
		gt.Equal(t, got[1].Timestamp, comment1.Timestamp)
		gt.Equal(t, got[1].UserID, comment1.UserID)
	})

	t.Run("GetPolicy_and_SavePolicy", func(t *testing.T) {
		policy := model.PolicyData{
			Hash:      uuid.New().String(),
			Data:      map[string]string{"test": "test"},
			CreatedAt: time.Now(),
		}

		gt.NoError(t, repo.SavePolicy(ctx, &policy))

		got, err := repo.GetPolicy(ctx, policy.Hash)
		gt.NoError(t, err).Must()
		gt.NotNil(t, got)
		gt.Equal(t, policy.Hash, got.Hash)
		gt.Equal(t, policy.Data, got.Data)
		// NOTE: Firestore returns time in UTC and has microseconds precision
		gt.Equal(t, policy.CreatedAt.Unix(), got.CreatedAt.Unix())
	})

	t.Run("GetAlertsByStatus", func(t *testing.T) {
		alert1 := model.NewAlert(ctx, "test", model.PolicyAlert{
			Title: "test",
			Attrs: []model.Attribute{
				{Key: "test", Value: "test"},
			},
		})
		alert1.Status = model.AlertStatusNew
		alert2 := model.NewAlert(ctx, "test", model.PolicyAlert{
			Title: "test",
			Attrs: []model.Attribute{
				{Key: "test", Value: "test"},
			},
		})
		alert2.Status = model.AlertStatusClosed
		gt.NoError(t, repo.PutAlert(ctx, alert1))
		gt.NoError(t, repo.PutAlert(ctx, alert2))

		got, err := repo.GetAlertsByStatus(ctx, model.AlertStatusNew)
		gt.NoError(t, err)
		gt.A(t, got).Longer(0).Any(func(v model.Alert) bool {
			return v.ID == alert1.ID
		}).All(func(v model.Alert) bool {
			return v.ID != alert2.ID
		}).All(func(v model.Alert) bool {
			return v.Status == model.AlertStatusNew
		})
	})

	t.Run("BatchGetAlerts", func(t *testing.T) {
		alert1 := model.NewAlert(ctx, "test", model.PolicyAlert{
			Title: "test",
			Attrs: []model.Attribute{
				{Key: "test", Value: "test"},
			},
		})
		alert2 := model.NewAlert(ctx, "test", model.PolicyAlert{
			Title: "test",
			Attrs: []model.Attribute{
				{Key: "test", Value: "test"},
			},
		})
		alert2.Status = model.AlertStatusClosed
		gt.NoError(t, repo.PutAlert(ctx, alert1))
		gt.NoError(t, repo.PutAlert(ctx, alert2))

		got, err := repo.BatchGetAlerts(ctx, []model.AlertID{alert1.ID, alert2.ID})
		gt.NoError(t, err)
		gt.A(t, got).
			Length(2).
			Any(func(v model.Alert) bool {
				return v.ID == alert1.ID
			}).
			Any(func(v model.Alert) bool {
				return v.ID == alert2.ID
			})
	})

	t.Run("PutAlertGroups", func(t *testing.T) {
		alert1 := model.NewAlert(ctx, "test", model.PolicyAlert{
			Title: "test",
			Attrs: []model.Attribute{
				{Key: "test", Value: "test"},
			},
		})
		alert2 := model.NewAlert(ctx, "test", model.PolicyAlert{
			Title: "test",
			Attrs: []model.Attribute{
				{Key: "test", Value: "test"},
			},
		})

		group1 := model.AlertGroup{
			ID: model.AlertGroupID(uuid.New().String()),
			AlertGroupMetadata: model.AlertGroupMetadata{
				AlertIDs: []model.AlertID{alert1.ID, alert2.ID},
			},
		}
		group2 := model.AlertGroup{
			ID: model.AlertGroupID(uuid.New().String()),
			AlertGroupMetadata: model.AlertGroupMetadata{
				AlertIDs: []model.AlertID{alert1.ID, alert2.ID},
			},
		}
		gt.NoError(t, repo.PutAlertGroups(ctx, []model.AlertGroup{group1, group2}))

		got, err := repo.GetAlertGroup(ctx, group1.ID)
		gt.NoError(t, err)
		gt.Equal(t, group1.ID, got.ID)

		got, err = repo.GetAlertGroup(ctx, group2.ID)
		gt.NoError(t, err)
		gt.Equal(t, group2.ID, got.ID)
	})

	t.Run("GetAlertGroup_NotFound", func(t *testing.T) {
		got, err := repo.GetAlertGroup(ctx, model.AlertGroupID(uuid.New().String()))
		gt.NoError(t, err)
		gt.Nil(t, got)
	})

	t.Run("GetAlertsByParentID", func(t *testing.T) {
		alert1 := model.NewAlert(ctx, "test", model.PolicyAlert{
			Title: "GetAlerts test 1",
			Attrs: []model.Attribute{
				{Key: "test", Value: "test"},
			},
		})
		alert2 := model.NewAlert(ctx, "test", model.PolicyAlert{
			Title: "GetAlerts test 2",
			Attrs: []model.Attribute{
				{Key: "test", Value: "test"},
			},
		})
		alert3 := model.NewAlert(ctx, "test", model.PolicyAlert{
			Title: "GetAlerts test 3",
			Attrs: []model.Attribute{
				{Key: "test", Value: "test"},
			},
		})

		alert2.ParentID = alert1.ID
		alert3.ParentID = alert1.ID
		gt.NoError(t, repo.PutAlert(ctx, alert1))
		gt.NoError(t, repo.PutAlert(ctx, alert2))
		gt.NoError(t, repo.PutAlert(ctx, alert3))

		got, err := repo.GetAlertsByParentID(ctx, alert1.ID)
		gt.NoError(t, err)
		gt.Equal(t, len(got), 2)
		gt.A(t, got).
			Length(2).
			Any(func(v model.Alert) bool {
				return v.ID == alert2.ID
			}).
			Any(func(v model.Alert) bool {
				return v.ID == alert3.ID
			})
	})

	t.Run("GetPolicyDiff", func(t *testing.T) {
		diff := model.NewPolicyDiff(ctx, model.NewPolicyDiffID(), "test", "test", map[string]string{"test": "test"}, map[string]string{}, model.NewTestDataSet())
		gt.NoError(t, repo.PutPolicyDiff(ctx, diff))

		got, err := repo.GetPolicyDiff(ctx, diff.ID)
		gt.NoError(t, err)
		gt.Equal(t, diff.ID, got.ID)
	})

	t.Run("GetPolicyDiff_NotFound", func(t *testing.T) {
		got, err := repo.GetPolicyDiff(ctx, model.PolicyDiffID(uuid.New().String()))
		gt.NoError(t, err)
		gt.Nil(t, got)
	})
}
