package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
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

	// Create test data
	alertID := types.NewAlertID()
	thread := slack.Thread{
		ChannelID: "test-channel",
		ThreadID:  fmt.Sprintf("%d.%d", time.Now().Unix(), time.Now().Nanosecond()),
	}
	a := alert.Alert{
		ID:          alertID,
		Schema:      "test-schema",
		CreatedAt:   time.Now(),
		SlackThread: &thread,
		Metadata: alert.Metadata{
			Title:       "Test Alert",
			Description: "Test Description",
			Data:        map[string]any{"key": "value"},
			Attrs: []alert.Attribute{
				{Key: "test-key", Value: "test-value"},
			},
		},
	}

	// Alert basic operations
	t.Run("AlertBasic", func(t *testing.T) {
		// PutAlert
		gt.NoError(t, repo.PutAlert(ctx, a))

		// GetAlert
		got, err := repo.GetAlert(ctx, alertID)
		gt.NoError(t, err)
		gt.Value(t, got.ID).Equal(alertID)
		gt.Value(t, got.Schema).Equal("test-schema")

		// GetAlertByThread
		got, err = repo.GetAlertByThread(ctx, thread)
		gt.NoError(t, err)
		gt.Value(t, got.ID).Equal(alertID)

		// SearchAlerts
		gotAlerts, err := repo.SearchAlerts(ctx, "Schema", "==", "test-schema")
		gt.NoError(t, err)
		gt.Array(t, gotAlerts).Longer(0)
		gt.Value(t, gotAlerts[0].ID).Equal(alertID)

		// BatchGetAlerts
		gotAlerts, err = repo.BatchGetAlerts(ctx, []types.AlertID{alertID})
		gt.NoError(t, err)
		gt.Array(t, gotAlerts).Equal([]*alert.Alert{&a})
	})

	// Alert-Ticket binding tests
	t.Run("AlertTicketBinding", func(t *testing.T) {
		ticketID := types.NewTicketID()
		ticketObj := ticket.Ticket{
			ID:          ticketID,
			Title:       "Test Ticket",
			Description: "Test Description",
		}

		// PutTicket
		gt.NoError(t, repo.PutTicket(ctx, ticketObj))

		// GetTicket
		got, err := repo.GetTicket(ctx, ticketID)
		gt.NoError(t, err)
		gt.Value(t, got.ID).Equal(ticketID)
		gt.Value(t, got.Title).Equal("Test Ticket")

		// BindAlertToTicket
		gt.NoError(t, repo.BindAlertToTicket(ctx, alertID, ticketID))

		// GetAlertWithoutTicket
		gotAlerts, err := repo.GetAlertWithoutTicket(ctx)
		gt.NoError(t, err)
		gt.Array(t, gotAlerts).Equal([]*alert.Alert{})

		// UnbindAlertFromTicket
		gt.NoError(t, repo.UnbindAlertFromTicket(ctx, alertID))

		// GetAlertWithoutTicket again
		gotAlerts, err = repo.GetAlertWithoutTicket(ctx)
		gt.NoError(t, err)
		gt.Array(t, gotAlerts).Longer(0)
		gt.Value(t, gotAlerts[0].ID).Equal(alertID)

		// PutTicketComment
		comment := ticket.Comment{
			ID:        types.NewCommentID(),
			TicketID:  ticketID,
			Comment:   "Test Comment",
			Timestamp: time.Now(),
			User: slack.User{
				ID:   "test-user",
				Name: "Test User",
			},
		}
		gt.NoError(t, repo.PutTicketComment(ctx, comment))

		// GetTicketComments
		gotComments, err := repo.GetTicketComments(ctx, ticketID)
		gt.NoError(t, err)
		gt.Array(t, gotComments).Longer(0)
		gt.Value(t, gotComments[0].Comment).Equal("Test Comment")
	})

	// AlertList related tests
	t.Run("AlertList", func(t *testing.T) {
		list := alert.List{
			ID:          types.NewAlertListID(),
			Title:       "Test List",
			Description: "Test Description",
			AlertIDs:    []types.AlertID{types.NewAlertID(), types.NewAlertID()},
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

	// Alert search related tests
	t.Run("AlertSearch", func(t *testing.T) {
		// GetAlertsBySpan
		begin := a.CreatedAt.Add(-1 * time.Minute)
		end := a.CreatedAt.Add(1 * time.Minute)
		got, err := repo.GetAlertsBySpan(ctx, begin, end)
		gt.NoError(t, err)
		// 検索結果が空でないことのみ確認
		gt.Array(t, got).Longer(0)

		// SearchAlerts
		got, err = repo.SearchAlerts(ctx, "Schema", "==", "test-schema")
		gt.NoError(t, err)
		gt.Array(t, got).Longer(0)
		gt.Value(t, got[0].Schema).Equal("test-schema")

		// SearchAlerts with different operators
		got, err = repo.SearchAlerts(ctx, "CreatedAt", ">", begin)
		gt.NoError(t, err)
		gt.Array(t, got).Longer(0)

		got, err = repo.SearchAlerts(ctx, "CreatedAt", "<", end)
		gt.NoError(t, err)
		gt.Array(t, got).Longer(0)
	})

	// Session related tests
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
		gt.NotNil(t, got).Required()
		gt.Value(t, got.ID).Equal(sessionID)
		gt.Value(t, got.Thread.ChannelID).Equal(thread.ChannelID)
		gt.Value(t, got.Thread.ThreadID).Equal(thread.ThreadID)

		// GetSessionByThread
		got, err = repo.GetSessionByThread(ctx, thread)
		gt.NoError(t, err)
		gt.NotNil(t, got)
		gt.Value(t, got.ID).Equal(sessionID)
		gt.Value(t, got.Thread.ChannelID).Equal(thread.ChannelID)
		gt.Value(t, got.Thread.ThreadID).Equal(thread.ThreadID)

		// PutHistory
		history := session.NewHistory(ctx, sessionID)
		gt.NoError(t, repo.PutHistory(ctx, sessionID, history))

		// GetLatestHistory
		gotHistory, err := repo.GetLatestHistory(ctx, sessionID)
		gt.NoError(t, err)
		gt.NotNil(t, gotHistory)
		gt.Value(t, gotHistory.SessionID).Equal(sessionID)
	})

	// Ticket関連のテストも必要に応じて追加
}
