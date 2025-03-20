package repository_test

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/chat"
	"github.com/secmon-lab/warren/pkg/domain/model/policy"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
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

	// テスト用のデータを作成
	alertID := types.NewAlertID()
	a := alert.Alert{
		ID:          alertID,
		Schema:      "test-schema",
		Title:       "Test Alert",
		Description: "Test Description",
		Status:      alert.StatusNew,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Data:        map[string]string{"key": "value"},
		Attributes: []alert.Attribute{
			{Key: "test-key", Value: "test-value"},
		},
	}

	thread := slack.Thread{
		ChannelID: "test-channel",
		ThreadID:  "test-thread",
	}

	// Alert関連のテスト
	t.Run("PutAndGetAlert", func(t *testing.T) {
		// Put
		gt.NoError(t, repo.PutAlert(ctx, a))

		// Get
		got, err := repo.GetAlert(ctx, alertID)
		gt.NoError(t, err)
		gt.Value(t, got.ID).Equal(a.ID)
		gt.Value(t, got.Schema).Equal(a.Schema)
		gt.Value(t, got.Title).Equal(a.Title)
		gt.Value(t, got.Description).Equal(a.Description)
		gt.Value(t, got.Status).Equal(a.Status)
		gt.Value(t, got.Data).Equal(a.Data)
		gt.Array(t, got.Attributes).Equal(a.Attributes)
	})

	t.Run("GetAlertByThread", func(t *testing.T) {
		// スレッドを設定
		a.SlackThread = &thread
		gt.NoError(t, repo.PutAlert(ctx, a))

		// GetByThread
		got, err := repo.GetAlertByThread(ctx, thread)
		gt.NoError(t, err)
		gt.Value(t, got.ID).Equal(a.ID)
		gt.Value(t, got.SlackThread.ChannelID).Equal(thread.ChannelID)
		gt.Value(t, got.SlackThread.ThreadID).Equal(thread.ThreadID)
	})

	t.Run("AlertComments", func(t *testing.T) {
		comment := alert.AlertComment{
			AlertID:   alertID,
			Timestamp: time.Now().Format(time.RFC3339),
			Comment:   "Test comment",
			User: slack.User{
				ID:   "test-user",
				Name: "Test User",
			},
		}

		// PutComment
		gt.NoError(t, repo.PutAlertComment(ctx, comment))

		// GetComments
		got, err := repo.GetAlertComments(ctx, alertID)
		gt.NoError(t, err)
		gt.Array(t, got).Have(comment)
	})

	// Chat関連のテスト
	t.Run("History", func(t *testing.T) {
		history := chat.History{
			ID:        types.HistoryID("test-history-id"),
			Thread:    thread,
			CreatedBy: slack.User{ID: "test-user", Name: "Test User"},
			CreatedAt: time.Now(),
			Contents:  []*genai.Content{{Role: "user", Parts: []genai.Part{genai.Text("test message")}}},
		}

		// PutHistory
		gt.NoError(t, repo.PutHistory(ctx, history))

		// GetHistory
		got, err := repo.GetHistory(ctx, thread)
		gt.NoError(t, err)
		gt.Value(t, got.ID).Equal(history.ID)
		gt.Value(t, got.Thread.ChannelID).Equal(thread.ChannelID)
		gt.Value(t, got.Thread.ThreadID).Equal(thread.ThreadID)
		gt.Value(t, got.Contents[0].Role).Equal("user")
	})

	// AlertList関連のテスト
	t.Run("AlertList", func(t *testing.T) {
		list := alert.List{
			ID:          types.NewAlertListID(),
			Title:       "Test List",
			Description: "Test Description",
			AlertIDs:    []types.AlertID{a.ID},
			SlackThread: &thread,
			CreatedAt:   time.Now(),
			CreatedBy: &slack.User{
				ID:   "test-user",
				Name: "Test User",
			},
		}

		// PutAlertList
		gt.NoError(t, repo.PutAlertList(ctx, list))

		// GetAlertList
		got, err := repo.GetAlertList(ctx, list.ID)
		gt.NoError(t, err)
		gt.Value(t, got.ID).Equal(list.ID)
		gt.Value(t, got.Title).Equal(list.Title)
		gt.Value(t, got.Description).Equal(list.Description)
		gt.Array(t, got.AlertIDs).Equal(list.AlertIDs)

		// GetAlertListByThread
		got, err = repo.GetAlertListByThread(ctx, thread)
		gt.NoError(t, err)
		gt.Value(t, got.ID).Equal(list.ID)
		gt.Value(t, got.SlackThread.ChannelID).Equal(thread.ChannelID)
		gt.Value(t, got.SlackThread.ThreadID).Equal(thread.ThreadID)

		// GetLatestAlertListInThread
		got, err = repo.GetLatestAlertListInThread(ctx, thread)
		gt.NoError(t, err)
		gt.Value(t, got.ID).Equal(list.ID)
		gt.Value(t, got.SlackThread.ChannelID).Equal(thread.ChannelID)
		gt.Value(t, got.SlackThread.ThreadID).Equal(thread.ThreadID)
	})

	// Alert検索関連のテスト
	t.Run("AlertSearch", func(t *testing.T) {
		// GetAlertsByStatus
		got, err := repo.GetAlertsByStatus(ctx, a.Status)
		gt.NoError(t, err)
		gt.Array(t, got).Have(a)

		// GetAlertsBySpan
		begin := time.Now().Add(-1 * time.Hour)
		end := time.Now().Add(1 * time.Hour)
		got, err = repo.GetAlertsBySpan(ctx, begin, end)
		gt.NoError(t, err)
		gt.Array(t, got).Have(a)

		// BatchGetAlerts
		got, err = repo.BatchGetAlerts(ctx, []types.AlertID{alertID})
		gt.NoError(t, err)
		gt.Array(t, got).Have(a)

		// BatchUpdateAlertStatus
		gt.NoError(t, repo.BatchUpdateAlertStatus(ctx, []types.AlertID{alertID}, a.Status, "Test reason"))
		gotAlert, err := repo.GetAlert(ctx, alertID)
		gt.NoError(t, err)
		gt.Value(t, gotAlert.Status).Equal(a.Status)
		gt.Value(t, gotAlert.Reason).Equal("Test reason")
	})

	// Policy関連のテスト
	t.Run("Policy", func(t *testing.T) {
		diffID := policy.PolicyDiffID("test-diff-id")
		diff := &policy.Diff{
			ID:          diffID,
			Title:       "Test Title",
			Description: "Test Description",
			CreatedAt:   time.Now(),
			New:         map[string]string{"key": "value"},
			Old:         map[string]string{},
		}

		// PutPolicyDiff
		gt.NoError(t, repo.PutPolicyDiff(ctx, diff))

		// GetPolicyDiff
		got, err := repo.GetPolicyDiff(ctx, types.PolicyDiffID(diffID))
		gt.NoError(t, err)
		gt.Value(t, got.ID).Equal(diff.ID)
		gt.Value(t, got.New).Equal(diff.New)
	})

	t.Run("Alert", func(t *testing.T) {
		a := alert.Alert{
			ID:          types.AlertID("test-alert-id"),
			Title:       "Test Title",
			Description: "Test Description",
			Status:      alert.StatusNew,
			CreatedAt:   time.Now(),
		}

		// PutAlert
		gt.NoError(t, repo.PutAlert(ctx, a))

		// GetAlert
		got, err := repo.GetAlert(ctx, a.ID)
		gt.NoError(t, err)
		gt.Value(t, got.ID).Equal(a.ID)
		gt.Value(t, got.Title).Equal(a.Title)

		// GetAlertsByStatus
		alerts, err := repo.GetAlertsByStatus(ctx, alert.StatusNew)
		gt.NoError(t, err)
		gt.Value(t, len(alerts)).Equal(1)
		gt.Value(t, alerts[0].ID).Equal(a.ID)

		// BatchUpdateAlertStatus
		gt.NoError(t, repo.BatchUpdateAlertStatus(ctx, []types.AlertID{a.ID}, alert.StatusAcknowledged, "test reason"))

		got, err = repo.GetAlert(ctx, a.ID)
		gt.NoError(t, err)
		gt.Value(t, got.Status).Equal(alert.StatusAcknowledged)
	})
}
