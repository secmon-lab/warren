package repository_test

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/google/uuid"
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

func newFirestoreClient(t *testing.T) *repository.Firestore {
	vars := test.NewEnvVars(t, "TEST_FIRESTORE_PROJECT_ID", "TEST_FIRESTORE_DATABASE_ID")
	client, err := repository.NewFirestore(t.Context(),
		vars.Get("TEST_FIRESTORE_PROJECT_ID"),
		vars.Get("TEST_FIRESTORE_DATABASE_ID"),
	)
	gt.NoError(t, err).Required()
	return client
}

func newTestThread() slack.Thread {
	return slack.Thread{
		ChannelID: "test-channel",
		ThreadID:  fmt.Sprintf("%d.%d", time.Now().Unix(), time.Now().Nanosecond()),
	}
}

func newTestAlert(thread *slack.Thread) alert.Alert {
	return alert.Alert{
		ID:          types.NewAlertID(),
		Schema:      types.AlertSchema("test-schema." + uuid.New().String()),
		CreatedAt:   time.Now(),
		SlackThread: thread,
		Metadata: alert.Metadata{
			Title:       "Test Alert",
			Description: "Test Description",
			Attributes: []alert.Attribute{
				{Key: "test-key", Value: "test-value"},
			},
		},
		Data: map[string]any{"key": "value"},
		// Embedding: make([]float32, 256),
	}
}

func newTestTicket(thread *slack.Thread) ticket.Ticket {
	return ticket.Ticket{
		ID: types.NewTicketID(),
		Metadata: ticket.Metadata{
			Title:       "Test Ticket",
			Description: "Test Description",
		},
		SlackThread: thread,
	}
}

func newTestAlertList(thread *slack.Thread, alertIDs []types.AlertID) alert.List {
	return alert.List{
		ID:          types.NewAlertListID(),
		Title:       "Test List",
		Description: "Test Description",
		AlertIDs:    alertIDs,
		SlackThread: thread,
		CreatedAt:   time.Now(),
		CreatedBy: &slack.User{
			ID:   "test-user",
			Name: "Test User",
		},
	}
}

func newTestSession(thread *slack.Thread) session.Session {
	return session.Session{
		ID:     types.NewSessionID(),
		Thread: thread,
	}
}

func TestAlertBasic(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := context.Background()
		thread := newTestThread()
		a := newTestAlert(&thread)

		// PutAlert
		gt.NoError(t, repo.PutAlert(ctx, a))

		// GetAlert
		got, err := repo.GetAlert(ctx, a.ID)
		gt.NoError(t, err)
		gt.Value(t, got.ID).Equal(a.ID)
		gt.Value(t, got.Schema).Equal(a.Schema)

		// GetAlertByThread
		got, err = repo.GetAlertByThread(ctx, thread)
		gt.NoError(t, err)
		gt.Value(t, got.ID).Equal(a.ID)

		// SearchAlerts
		gotAlerts, err := repo.SearchAlerts(ctx, "Schema", "==", a.Schema, 3)
		gt.NoError(t, err)
		gt.Array(t, gotAlerts).Length(1)
		gt.Value(t, gotAlerts[0].ID).Equal(a.ID)

		// BatchGetAlerts
		gotAlerts, err = repo.BatchGetAlerts(ctx, []types.AlertID{a.ID})
		gt.NoError(t, err)
		gt.Array(t, gotAlerts).Length(1).Required()
		gt.Equal(t, gotAlerts[0].ID, a.ID)
		gt.Equal(t, gotAlerts[0].Schema, a.Schema)
		gt.Equal(t, gotAlerts[0].SlackThread, a.SlackThread)
		gt.Equal(t, gotAlerts[0].Metadata, a.Metadata)
		gt.Equal(t, gotAlerts[0].Data, a.Data)
		gt.Equal(t, gotAlerts[0].Embedding, a.Embedding)
		gt.Equal(t, gotAlerts[0].CreatedAt.Unix(), a.CreatedAt.Unix())
	}

	t.Run("Memory", func(t *testing.T) {
		repo := repository.NewMemory()
		testFn(t, repo)
	})

	t.Run("Firestore", func(t *testing.T) {
		repo := newFirestoreClient(t)
		testFn(t, repo)
	})
}

func TestAlertTicketBinding(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := context.Background()
		thread := newTestThread()
		testAlert := newTestAlert(&thread)
		ticketObj := newTestTicket(&thread)

		// PutAlert and PutTicket
		gt.NoError(t, repo.PutAlert(ctx, testAlert))
		gt.NoError(t, repo.PutTicket(ctx, ticketObj))

		// GetTicket
		got, err := repo.GetTicket(ctx, ticketObj.ID)
		gt.NoError(t, err)
		gt.Value(t, got.ID).Equal(ticketObj.ID)
		gt.Value(t, got.Title).Equal(ticketObj.Title)

		// GetTicketByThread
		got, err = repo.GetTicketByThread(ctx, thread)
		gt.NoError(t, err)
		gt.NotNil(t, got)
		gt.Value(t, got.ID).Equal(ticketObj.ID)
		gt.Value(t, got.SlackThread.ChannelID).Equal(thread.ChannelID)
		gt.Value(t, got.SlackThread.ThreadID).Equal(thread.ThreadID)

		// BindAlertToTicket
		gt.NoError(t, repo.BindAlertToTicket(ctx, testAlert.ID, ticketObj.ID))

		// UnbindAlertFromTicket
		gt.NoError(t, repo.UnbindAlertFromTicket(ctx, testAlert.ID))

		// GetAlertWithoutTicket again
		gotAlerts, err := repo.GetAlertWithoutTicket(ctx)
		gt.NoError(t, err)
		gt.Array(t, gotAlerts).Longer(0)
		gt.Array(t, gotAlerts).Any(func(a *alert.Alert) bool {
			return a.ID == testAlert.ID
		})

		// PutTicketComment
		comment := ticketObj.NewComment(ctx, "Test Comment", slack.User{
			ID:   "test-user",
			Name: "Test User",
		})
		gt.NoError(t, repo.PutTicketComment(ctx, comment))

		// GetTicketComments
		gotComments, err := repo.GetTicketComments(ctx, ticketObj.ID)
		gt.NoError(t, err)
		gt.Array(t, gotComments).Longer(0).Required()
		gt.Value(t, gotComments[0].Comment).Equal("Test Comment")
	}

	t.Run("Memory", func(t *testing.T) {
		repo := repository.NewMemory()
		testFn(t, repo)
	})

	t.Run("Firestore", func(t *testing.T) {
		repo := newFirestoreClient(t)
		testFn(t, repo)
	})
}

func TestAlertList(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := context.Background()
		thread := newTestThread()
		alertIDs := []types.AlertID{types.NewAlertID(), types.NewAlertID()}
		list := newTestAlertList(&thread, alertIDs)

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
	}

	t.Run("Memory", func(t *testing.T) {
		repo := repository.NewMemory()
		testFn(t, repo)
	})

	t.Run("Firestore", func(t *testing.T) {
		repo := newFirestoreClient(t)
		testFn(t, repo)
	})
}

func TestAlertSearch(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := context.Background()
		alert := newTestAlert(nil)

		gt.NoError(t, repo.PutAlert(ctx, alert))

		// GetAlertsBySpan
		begin := alert.CreatedAt.Add(-1 * time.Minute)
		end := alert.CreatedAt.Add(1 * time.Minute)
		got, err := repo.GetAlertsBySpan(ctx, begin, end)
		gt.NoError(t, err)
		gt.Array(t, got).Longer(0)

		// SearchAlerts
		got, err = repo.SearchAlerts(ctx, "Schema", "==", alert.Schema, 3)
		gt.NoError(t, err)
		gt.Array(t, got).Longer(0)
		gt.Value(t, got[0].Schema).Equal(alert.Schema)

		// SearchAlerts with different operators
		got, err = repo.SearchAlerts(ctx, "CreatedAt", ">", begin, 3)
		gt.NoError(t, err)
		gt.Array(t, got).Longer(0)

		got, err = repo.SearchAlerts(ctx, "CreatedAt", "<", end, 3)
		gt.NoError(t, err)
		gt.Array(t, got).Longer(0)
	}

	t.Run("Memory", func(t *testing.T) {
		repo := repository.NewMemory()
		testFn(t, repo)
	})

	t.Run("Firestore", func(t *testing.T) {
		repo := newFirestoreClient(t)
		testFn(t, repo)
	})
}

func TestSession(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := context.Background()
		thread := newTestThread()
		s := newTestSession(&thread)

		// PutSession
		gt.NoError(t, repo.PutSession(ctx, s))

		// GetSession
		got, err := repo.GetSession(ctx, s.ID)
		gt.NoError(t, err)
		gt.NotNil(t, got).Required()
		gt.Value(t, got.ID).Equal(s.ID)
		gt.Value(t, got.Thread.ChannelID).Equal(thread.ChannelID)
		gt.Value(t, got.Thread.ThreadID).Equal(thread.ThreadID)

		// GetSessionByThread
		got, err = repo.GetSessionByThread(ctx, thread)
		gt.NoError(t, err)
		gt.NotNil(t, got)
		gt.Value(t, got.ID).Equal(s.ID)
		gt.Value(t, got.Thread.ChannelID).Equal(thread.ChannelID)
		gt.Value(t, got.Thread.ThreadID).Equal(thread.ThreadID)

		// PutHistory
		history := session.NewHistory(ctx, s.ID)
		gt.NoError(t, repo.PutHistory(ctx, s.ID, history))

		// GetLatestHistory
		gotHistory, err := repo.GetLatestHistory(ctx, s.ID)
		gt.NoError(t, err)
		gt.NotNil(t, gotHistory)
		gt.Value(t, gotHistory.SessionID).Equal(s.ID)
	}

	t.Run("Memory", func(t *testing.T) {
		repo := repository.NewMemory()
		testFn(t, repo)
	})

	t.Run("Firestore", func(t *testing.T) {
		repo := newFirestoreClient(t)
		testFn(t, repo)
	})
}

func TestFindSimilarAlerts(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()
		alerts := alert.Alerts{}
		for i := 0; i < 10; i++ {
			// Generate random embedding array with 256 dimensions
			embeddings := make([]float32, 256)
			for i := range embeddings {
				embeddings[i] = rand.Float32()
			}
			alerts = append(alerts, &alert.Alert{
				ID:        types.NewAlertID(),
				Schema:    types.AlertSchema("test-schema." + uuid.New().String()),
				Embedding: embeddings,
				CreatedAt: time.Now(),
			})
			gt.NoError(t, repo.PutAlert(ctx, *alerts[i]))
		}

		newEmbedding := make([]float32, 256)
		for i := range newEmbedding {
			newEmbedding[i] = alerts[0].Embedding[i]
		}
		newEmbedding[0] = 1.0 // Change one value to make it different

		gt.Number(t, cosineSimilarity(alerts[0].Embedding, newEmbedding)).Greater(0.99)

		target := alert.Alert{
			ID:        types.NewAlertID(),
			Schema:    types.AlertSchema("test-schema." + uuid.New().String()),
			Embedding: newEmbedding,
			CreatedAt: time.Now(),
		}
		gt.NoError(t, repo.PutAlert(ctx, target))
		got, err := repo.FindSimilarAlerts(ctx, target, 3)
		gt.NoError(t, err).Required()
		gt.Array(t, got).Longer(0).Required()
		gt.Value(t, got[0].ID).Equal(alerts[0].ID)
	}

	t.Run("Memory", func(t *testing.T) {
		repo := repository.NewMemory()
		testFn(t, repo)
	})

	t.Run("Firestore", func(t *testing.T) {
		repo := newFirestoreClient(t)
		testFn(t, repo)
	})
}

func cosineSimilarity(a, b []float32) float32 {
	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}
