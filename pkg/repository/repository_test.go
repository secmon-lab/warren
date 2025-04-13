package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/policy"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
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
		Status:      types.AlertStatusNew,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Data:        map[string]any{"key": "value"},
		Attributes: []alert.Attribute{
			{Key: "test-key", Value: "test-value"},
		},
	}

	thread := slack.Thread{
		ChannelID: "test-channel",
		ThreadID:  fmt.Sprintf("%d.%d", time.Now().Unix(), time.Now().Nanosecond()),
	}

	// Put
	gt.NoError(t, repo.PutAlert(ctx, a))

	// Alert関連のテスト
	t.Run("PutAndGetAlert", func(t *testing.T) {
		// Get
		got, err := repo.GetAlert(ctx, alertID)
		gt.NoError(t, err)
		gt.Value(t, got.ID).Equal(a.ID)
		gt.Value(t, got.Schema).Equal(a.Schema)
		gt.Value(t, got.Title).Equal(a.Title)
		gt.Value(t, got.Description).Equal(a.Description)
		gt.Value(t, got.Status).Equal(a.Status)
		gotData := gt.Cast[map[string]any](t, got.Data)
		wantData := gt.Cast[map[string]any](t, a.Data)
		gt.Value(t, gotData["key"]).Equal(wantData["key"])
		gt.Array(t, got.Attributes).Equal(a.Attributes)
	})

	t.Run("GetAlertByThread", func(t *testing.T) {
		// スレッドを設定
		a.SlackThread = &thread
		gt.NoError(t, repo.PutAlert(ctx, a))

		// GetByThread
		got, err := repo.GetAlertByThread(ctx, thread)
		gt.NoError(t, err).Required()
		gt.NotNil(t, got)
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
		gt.Array(t, got).Has(comment)
	})

	// Chat関連のテスト
	t.Run("History", func(t *testing.T) {
		sessionID := types.NewSessionID()
		histories := []*session.History{
			{
				ID:        types.NewHistoryID(),
				CreatedAt: time.Now(),
				Contents: session.Contents{
					&session.Content{
						Role: "user",
						Parts: []session.Part{
							{
								Text: "test message 1",
							},
						},
					},
				},
			},
			{
				ID:        types.NewHistoryID(),
				CreatedAt: time.Now(),
				Contents: session.Contents{
					&session.Content{
						Role: "assistant",
						Parts: []session.Part{
							{
								Text: "test response 1",
							},
						},
					},
				},
			},
		}

		// PutHistory
		for _, h := range histories {
			gt.NoError(t, repo.PutHistory(ctx, sessionID, h))
		}

		// GetLatestHistory
		got, err := repo.GetLatestHistory(ctx, sessionID)
		gt.NoError(t, err)
		gt.Value(t, got).NotNil()
		gt.Value(t, got.Contents[0].Role).Equal("assistant")
		gt.Value(t, got.Contents[0].Parts[0].Text).Equal("test response 1")
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
		gt.Array(t, got).Any(func(v *alert.Alert) bool { return v.ID == alertID })

		// GetAlertsBySpan
		begin := a.CreatedAt.Add(-1 * time.Minute)
		end := a.CreatedAt.Add(1 * time.Minute)
		got, err = repo.GetAlertsBySpan(ctx, begin, end)
		gt.NoError(t, err)
		gt.Array(t, got).Any(func(v *alert.Alert) bool { return v.ID == alertID })

		// BatchGetAlerts
		got, err = repo.BatchGetAlerts(ctx, []types.AlertID{alertID})
		gt.NoError(t, err)
		gt.Array(t, got).Any(func(v *alert.Alert) bool { return v.ID == alertID })

		// BatchUpdateAlertStatus
		gt.NoError(t, repo.BatchUpdateAlertStatus(ctx, []types.AlertID{alertID}, types.AlertStatusResolved, "Test reason"))
		gotAlert, err := repo.GetAlert(ctx, alertID)
		gt.NoError(t, err)
		gt.Value(t, gotAlert.Status).Equal(types.AlertStatusResolved)
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
			Status:      types.AlertStatusNew,
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
		alerts, err := repo.GetAlertsByStatus(ctx, types.AlertStatusNew)
		gt.NoError(t, err)
		gt.Array(t, alerts).Any(func(v *alert.Alert) bool { return v.ID == a.ID })

		// GetAlertsWithoutStatus
		alerts, err = repo.GetAlertsWithoutStatus(ctx, types.AlertStatusResolved)
		gt.NoError(t, err)
		gt.Array(t, alerts).Any(func(v *alert.Alert) bool { return v.ID == a.ID })

		// BatchUpdateAlertStatus
		gt.NoError(t, repo.BatchUpdateAlertStatus(ctx, []types.AlertID{a.ID}, types.AlertStatusAcknowledged, "test reason"))

		got, err = repo.GetAlert(ctx, a.ID)
		gt.NoError(t, err)
		gt.Value(t, got.Status).Equal(types.AlertStatusAcknowledged)
	})

	// Session関連のテスト
	t.Run("Session", func(t *testing.T) {
		sessionID := types.NewSessionID()
		thread := slack.Thread{
			ChannelID: "test-channel",
			ThreadID:  fmt.Sprintf("%d.%d", time.Now().Unix(), time.Now().Nanosecond()),
		}
		s := session.Session{
			ID:     sessionID,
			Thread: &thread,
		}

		// PutSession
		gt.NoError(t, repo.PutSession(ctx, s))

		// GetSession
		got, err := repo.GetSession(ctx, sessionID)
		gt.NoError(t, err)
		gt.Value(t, got.ID).Equal(sessionID)
		gt.Value(t, got.Thread.ChannelID).Equal(thread.ChannelID)
		gt.Value(t, got.Thread.ThreadID).Equal(thread.ThreadID)

		// GetSessionByThread
		got, err = repo.GetSessionByThread(ctx, thread)
		gt.NoError(t, err)
		gt.Value(t, got.ID).Equal(sessionID)
		gt.Value(t, got.Thread.ChannelID).Equal(thread.ChannelID)
		gt.Value(t, got.Thread.ThreadID).Equal(thread.ThreadID)
	})

	// Test notes
	t.Run("notes", func(t *testing.T) {
		sessionID := types.NewSessionID()
		note1 := session.NewNote(sessionID, "test note 1")
		note2 := session.NewNote(sessionID, "test note 2")

		// Put notes
		err := repo.PutNote(ctx, note1)
		gt.NoError(t, err)
		err = repo.PutNote(ctx, note2)
		gt.NoError(t, err)

		// Get notes
		notes, err := repo.GetNotes(ctx, sessionID)
		gt.NoError(t, err)
		gt.Array(t, notes).Length(2)
		gt.Value(t, notes[0].Note).Equal("test note 1")
		gt.Value(t, notes[1].Note).Equal("test note 2")
	})

	// Test SearchAlerts
	t.Run("SearchAlerts", func(t *testing.T) {
		// テストデータの準備
		alerts := []alert.Alert{
			{
				ID:        types.AlertID(uuid.New().String()),
				Title:     "Test Alert 1",
				Status:    types.AlertStatus("open"),
				CreatedAt: time.Now(),
			},
			{
				ID:        types.AlertID(uuid.New().String()),
				Title:     "Test Alert 2",
				Status:    types.AlertStatus("closed"),
				CreatedAt: time.Now(),
			},
			{
				ID:        types.AlertID(uuid.New().String()),
				Title:     "Test Alert 3",
				Status:    types.AlertStatus("open"),
				CreatedAt: time.Now(),
			},
		}

		for _, a := range alerts {
			err := repo.PutAlert(ctx, a)
			gt.NoError(t, err)
		}

		// Search for alerts with status "open"
		result, err := repo.SearchAlerts(ctx, "Status", "==", types.AlertStatus("open"))
		gt.NoError(t, err)
		gt.Array(t, result).Longer(0)
		gt.Array(t, result).Any(func(v *alert.Alert) bool { return v.ID == alerts[0].ID })
		gt.Array(t, result).Any(func(v *alert.Alert) bool { return v.ID == alerts[2].ID })

		// Search for alerts with status not "open"
		result, err = repo.SearchAlerts(ctx, "Status", "!=", types.AlertStatus("open"))
		gt.NoError(t, err)
		gt.Array(t, result).Longer(0)
		gt.Array(t, result).Any(func(v *alert.Alert) bool { return v.ID == alerts[1].ID })

		// Specify invalid field path
		result, err = repo.SearchAlerts(ctx, "InvalidField", "==", "test")
		gt.NoError(t, err)
		gt.Array(t, result).Length(0)
	})
}
