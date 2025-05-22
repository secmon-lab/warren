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
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	ticketmodel "github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/utils/test"
	"github.com/slack-go/slack/slackevents"
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
		Embedding: make([]float32, 256),
		Data:      map[string]any{"key": "value"},
	}
}

func newTestTicket(thread *slack.Thread) ticketmodel.Ticket {
	return ticketmodel.Ticket{
		ID: types.NewTicketID(),
		Metadata: ticketmodel.Metadata{
			Title:       "Test Ticket",
			Description: "Test Description",
		},
		SlackThread: thread,
	}
}

func newTestAlertList(thread *slack.Thread, alertIDs []types.AlertID) alert.List {
	return alert.List{
		ID: types.NewAlertListID(),
		Metadata: alert.Metadata{
			Title:       "Test List",
			Description: "Test Description",
		},
		AlertIDs:    alertIDs,
		SlackThread: thread,
		CreatedAt:   time.Now(),
		CreatedBy: &slack.User{
			ID:   "test-user",
			Name: "Test User",
		},
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

		unbindAlerts, err := repo.GetAlertWithoutTicket(ctx)
		gt.NoError(t, err).Required()
		gt.Array(t, unbindAlerts).Longer(0).Any(func(a *alert.Alert) bool {
			return a.ID == testAlert.ID
		})

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

		// BatchBindAlertsToTicket
		alert2 := newTestAlert(&thread)
		alert3 := newTestAlert(&thread)
		gt.NoError(t, repo.PutAlert(ctx, alert2))
		gt.NoError(t, repo.PutAlert(ctx, alert3))
		gt.NoError(t, repo.BatchBindAlertsToTicket(ctx, []types.AlertID{alert2.ID, alert3.ID}, ticketObj.ID))

		// Verify alerts are bound
		gotAlert2, err := repo.GetAlert(ctx, alert2.ID)
		gt.NoError(t, err)
		gt.Value(t, gotAlert2.TicketID).Equal(ticketObj.ID)

		gotAlert3, err := repo.GetAlert(ctx, alert3.ID)
		gt.NoError(t, err)
		gt.Value(t, gotAlert3.TicketID).Equal(ticketObj.ID)

		// PutTicketComment
		comment := ticketObj.NewComment(ctx, *slack.NewMessage(ctx, &slackevents.EventsAPIEvent{
			InnerEvent: slackevents.EventsAPIInnerEvent{
				Data: &slackevents.AppMentionEvent{
					TimeStamp: "test-message-id",
					Text:      "Test Comment",
					User:      "test-user",
					Channel:   "test-channel",
				},
			},
		}))
		gt.NoError(t, repo.PutTicketComment(ctx, comment))

		// GetTicketComments
		gotComments, err := repo.GetTicketComments(ctx, ticketObj.ID)
		gt.NoError(t, err)
		gt.Array(t, gotComments).Longer(0).Required()
		gt.Value(t, gotComments[0].Comment).Equal("Test Comment")
		gt.Value(t, gotComments[0].SlackMessageID).Equal("test-message-id")
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
		got, err := repo.FindNearestAlerts(ctx, target.Embedding, 3)
		gt.NoError(t, err).Required()
		gt.Array(t, got).Longer(0).Required()
		found := false
		for _, a := range got {
			if a.ID == alerts[0].ID {
				found = true
				break
			}
		}
		gt.True(t, found)
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

func TestBatchGetTickets(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := context.Background()
		thread := newTestThread()

		// Create test tickets
		tickets := make([]*ticketmodel.Ticket, 3)
		ticketIDs := make([]types.TicketID, 3)
		for i := 0; i < 3; i++ {
			ticket := newTestTicket(&thread)
			ticket.Metadata.Title = fmt.Sprintf("Test Ticket %d", i)
			gt.NoError(t, repo.PutTicket(ctx, ticket)).Required()
			tickets[i] = &ticket
			ticketIDs[i] = ticket.ID
		}

		// Test BatchGetTickets
		got, err := repo.BatchGetTickets(ctx, ticketIDs)
		gt.NoError(t, err).Required()
		gt.Array(t, got).Length(3)

		// Verify each ticket
		for i, ticket := range got {
			gt.Value(t, ticket.ID).Equal(tickets[i].ID)
			gt.Value(t, ticket.Metadata.Title).Equal(tickets[i].Metadata.Title)
		}

		// Test with non-existent ticket ID
		nonExistentID := types.NewTicketID()
		got, err = repo.BatchGetTickets(ctx, []types.TicketID{nonExistentID})
		gt.NoError(t, err)
		gt.Array(t, got).Length(0)
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

func TestFindSimilarTickets(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()
		tickets := make([]*ticketmodel.Ticket, 10)
		for i := 0; i < 10; i++ {
			// Generate random embedding array with 256 dimensions
			embeddings := make([]float32, 256)
			for i := range embeddings {
				embeddings[i] = rand.Float32()
			}
			tickets[i] = &ticketmodel.Ticket{
				ID:        types.NewTicketID(),
				Embedding: embeddings,
				CreatedAt: time.Now(),
			}
			gt.NoError(t, repo.PutTicket(ctx, *tickets[i]))
		}

		newEmbedding := make([]float32, 256)
		for i := range newEmbedding {
			newEmbedding[i] = tickets[0].Embedding[i]
		}
		newEmbedding[0] = 1.0 // Change one value to make it different

		gt.Number(t, cosineSimilarity(tickets[0].Embedding, newEmbedding)).Greater(0.99)

		target := ticketmodel.Ticket{
			ID:        types.NewTicketID(),
			Embedding: newEmbedding,
			CreatedAt: time.Now(),
		}
		gt.NoError(t, repo.PutTicket(ctx, target))
		got, err := repo.FindNearestTickets(ctx, target.Embedding, 3)
		gt.NoError(t, err).Required()
		gt.Array(t, got).Longer(0).Required().Any(func(v *ticketmodel.Ticket) bool {
			return v.ID == tickets[0].ID
		})
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

func TestFindNearestTickets(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()
		tickets := make([]*ticketmodel.Ticket, 10)
		for i := 0; i < 10; i++ {
			// Generate random embedding array with 256 dimensions
			embeddings := make([]float32, 256)
			for i := range embeddings {
				embeddings[i] = rand.Float32()
			}
			tickets[i] = &ticketmodel.Ticket{
				ID:        types.NewTicketID(),
				Embedding: embeddings,
				CreatedAt: time.Now(),
			}
			gt.NoError(t, repo.PutTicket(ctx, *tickets[i]))
		}

		// Create a target embedding that is similar to the first ticket
		targetEmbedding := make([]float32, 256)
		for i := range targetEmbedding {
			targetEmbedding[i] = tickets[0].Embedding[i]
		}
		targetEmbedding[0] = 1.0 // Change one value to make it different

		gt.Number(t, cosineSimilarity(tickets[0].Embedding, targetEmbedding)).Greater(0.99)

		// Test FindNearestTickets
		got, err := repo.FindNearestTickets(ctx, targetEmbedding, 3)
		gt.NoError(t, err).Required()
		gt.Array(t, got).Longer(0).Required()
		gt.Value(t, got[0].ID).Equal(tickets[0].ID)
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

func TestFindNearestAlerts(t *testing.T) {
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

		// Create a target embedding that is similar to the first alert
		targetEmbedding := make([]float32, 256)
		for i := range targetEmbedding {
			targetEmbedding[i] = alerts[0].Embedding[i]
		}
		targetEmbedding[0] = 1.0 // Change one value to make it different

		gt.Number(t, cosineSimilarity(alerts[0].Embedding, targetEmbedding)).Greater(0.99)

		// Test FindNearestAlerts
		got, err := repo.FindNearestAlerts(ctx, targetEmbedding, 3)
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

func TestHistory(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := context.Background()
		thread := newTestThread()
		ticket := newTestTicket(&thread)

		// PutTicket
		gt.NoError(t, repo.PutTicket(ctx, ticket))

		// Create and put multiple histories
		histories := make([]ticketmodel.History, 3)
		for i := 0; i < 3; i++ {
			history := ticketmodel.NewHistory(ctx, ticket.ID)
			history.CreatedAt = time.Now().Add(time.Duration(i) * time.Hour)
			gt.NoError(t, repo.PutHistory(ctx, ticket.ID, &history))
			histories[i] = history
		}

		// Test GetLatestHistory
		latest, err := repo.GetLatestHistory(ctx, ticket.ID)
		gt.NoError(t, err).Required()
		gt.NotNil(t, latest).Required()
		gt.Value(t, latest.ID).Equal(histories[2].ID) // 最後に追加したものが最新

		// Test with non-existent ticket ID
		nonExistentID := types.NewTicketID()
		latest, err = repo.GetLatestHistory(ctx, nonExistentID)
		gt.NoError(t, err)
		gt.Nil(t, latest)
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
