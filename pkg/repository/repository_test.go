package repository_test

import (
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

func TestGetLatestAlertByThread(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()
		thread := newTestThread()
		a := newTestAlert(&thread)

		// PutAlert
		gt.NoError(t, repo.PutAlert(ctx, a))

		// GetAlert
		got, err := repo.GetAlert(ctx, a.ID)
		gt.NoError(t, err)
		gt.Value(t, got.ID).Equal(a.ID)
		gt.Value(t, got.Schema).Equal(a.Schema)

		// GetLatestAlertByThread
		got, err = repo.GetLatestAlertByThread(ctx, thread)
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
		ctx := t.Context()
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
		ctx := t.Context()
		thread := newTestThread()
		alertIDs := []types.AlertID{types.NewAlertID(), types.NewAlertID()}
		list := newTestAlertList(&thread, alertIDs)

		// PutAlertList
		gt.NoError(t, repo.PutAlertList(ctx, &list))

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
		ctx := t.Context()
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
		ctx := t.Context()
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
		ctx := t.Context()
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

func TestTicketComments(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()
		thread := newTestThread()
		ticket := newTestTicket(&thread)

		// PutTicket
		gt.NoError(t, repo.PutTicket(ctx, ticket))

		// Create and put multiple comments
		comments := make([]ticketmodel.Comment, 3)
		for i := 0; i < 3; i++ {
			comment := ticket.NewComment(ctx, *slack.NewMessage(ctx, &slackevents.EventsAPIEvent{
				InnerEvent: slackevents.EventsAPIInnerEvent{
					Data: &slackevents.AppMentionEvent{
						TimeStamp: fmt.Sprintf("test-message-id-%d", i),
						Text:      fmt.Sprintf("Test Comment %d", i),
						User:      "test-user",
						Channel:   "test-channel",
					},
				},
			}))
			gt.NoError(t, repo.PutTicketComment(ctx, comment))
			comments[i] = comment
		}

		// Test GetTicketUnpromptedComments
		unpromptedComments, err := repo.GetTicketUnpromptedComments(ctx, ticket.ID)
		gt.NoError(t, err).Required()
		gt.Array(t, unpromptedComments).Length(3).Required()

		// Test PutTicketCommentsPrompted
		commentIDs := []types.CommentID{comments[0].ID, comments[1].ID}
		gt.NoError(t, repo.PutTicketCommentsPrompted(ctx, ticket.ID, commentIDs))

		// Verify prompted status
		unpromptedComments, err = repo.GetTicketUnpromptedComments(ctx, ticket.ID)
		gt.NoError(t, err).Required()
		gt.Array(t, unpromptedComments).Length(1).Required()
		gt.Value(t, unpromptedComments[0].ID).Equal(comments[2].ID)

		// Test with non-existent ticket ID
		nonExistentID := types.NewTicketID()
		unpromptedComments, err = repo.GetTicketUnpromptedComments(ctx, nonExistentID)
		gt.NoError(t, err)
		gt.Array(t, unpromptedComments).Length(0)

		err = repo.PutTicketCommentsPrompted(ctx, nonExistentID, commentIDs)
		gt.Error(t, err).Required()
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

func TestBatchPutAlerts(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()
		alerts := alert.Alerts{
			&alert.Alert{
				ID:        types.NewAlertID(),
				CreatedAt: time.Now(),
				Metadata: alert.Metadata{
					Title: "Test Alert 1",
				},
			},
			&alert.Alert{
				ID:        types.NewAlertID(),
				CreatedAt: time.Now(),
				Metadata: alert.Metadata{
					Title: "Test Alert 2",
				},
			},
		}

		err := repo.BatchPutAlerts(ctx, alerts)
		gt.NoError(t, err)

		// Verify alerts were stored
		for _, alert := range alerts {
			stored, err := repo.GetAlert(ctx, alert.ID)
			gt.NoError(t, err)
			gt.Value(t, stored.ID).Equal(alert.ID)
			gt.Value(t, stored.Title).Equal(alert.Title)
		}
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

func TestGetTicketsByStatus(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()
		thread := newTestThread()

		// Create tickets
		tickets := []*ticketmodel.Ticket{
			{
				ID:        types.NewTicketID(),
				Status:    types.TicketStatusOpen,
				CreatedAt: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				SlackThread: &slack.Thread{
					ChannelID: thread.ChannelID,
					ThreadID:  thread.ThreadID,
				},
			},
			{
				ID:        types.NewTicketID(),
				Status:    types.TicketStatusPending,
				CreatedAt: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
				SlackThread: &slack.Thread{
					ChannelID: thread.ChannelID,
					ThreadID:  thread.ThreadID,
				},
			},
			{
				ID:        types.NewTicketID(),
				Status:    types.TicketStatusResolved,
				CreatedAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				SlackThread: &slack.Thread{
					ChannelID: thread.ChannelID,
					ThreadID:  thread.ThreadID,
				},
			},
		}

		// Put tickets
		for _, ticket := range tickets {
			gt.NoError(t, repo.PutTicket(ctx, *ticket))
		}

		t.Run("investigating tickets", func(t *testing.T) {
			result, err := repo.GetTicketsByStatus(ctx, []types.TicketStatus{types.TicketStatusOpen}, 0, 0)
			gt.NoError(t, err)
			filtered := filterByIDs(result, []types.TicketID{tickets[0].ID})
			gt.Array(t, filtered).Length(1)
			gt.Value(t, filtered[0].ID).Equal(tickets[0].ID)
		})

		t.Run("pending tickets", func(t *testing.T) {
			result, err := repo.GetTicketsByStatus(ctx, []types.TicketStatus{types.TicketStatusPending}, 0, 0)
			gt.NoError(t, err)
			filtered := filterByIDs(result, []types.TicketID{tickets[1].ID})
			gt.Array(t, filtered).Length(1)
			gt.Value(t, filtered[0].ID).Equal(tickets[1].ID)
		})

		t.Run("resolved tickets", func(t *testing.T) {
			result, err := repo.GetTicketsByStatus(ctx, []types.TicketStatus{types.TicketStatusResolved}, 0, 0)
			gt.NoError(t, err)
			filtered := filterByIDs(result, []types.TicketID{tickets[2].ID})
			gt.Array(t, filtered).Length(1)
			gt.Value(t, filtered[0].ID).Equal(tickets[2].ID)
		})

		t.Run("multiple statuses", func(t *testing.T) {
			result, err := repo.GetTicketsByStatus(ctx, []types.TicketStatus{
				types.TicketStatusOpen,
				types.TicketStatusPending,
			}, 0, 0)
			gt.NoError(t, err)
			filtered := filterByIDs(result, []types.TicketID{tickets[0].ID, tickets[1].ID})
			gt.Array(t, filtered).Length(2)
		})

		t.Run("all tickets", func(t *testing.T) {
			result, err := repo.GetTicketsByStatus(ctx, nil, 0, 0)
			gt.NoError(t, err)
			filtered := filterByIDs(result, []types.TicketID{tickets[0].ID, tickets[1].ID, tickets[2].ID})
			gt.Array(t, filtered).Length(3)
		})

		t.Run("with limit", func(t *testing.T) {
			result, err := repo.GetTicketsByStatus(ctx, nil, 0, 2)
			gt.NoError(t, err)
			gt.Array(t, result).Length(2)
		})

		t.Run("with offset", func(t *testing.T) {
			result1, err := repo.GetTicketsByStatus(ctx, nil, 1, 0)
			gt.NoError(t, err)
			gt.Array(t, result1).Longer(0)
			result2, err := repo.GetTicketsByStatus(ctx, nil, 2, 0)
			gt.NoError(t, err)
			gt.Array(t, result2).Longer(0)
			gt.Array(t, result2).All(func(v *ticketmodel.Ticket) bool {
				return v.ID != result1[0].ID
			})
		})

		t.Run("with offset and limit", func(t *testing.T) {
			result, err := repo.GetTicketsByStatus(ctx, nil, 1, 1)
			gt.NoError(t, err)
			gt.Array(t, result).Length(1)
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

func TestGetTicketsBySpan(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()
		thread := newTestThread()

		// Create tickets
		tickets := []*ticketmodel.Ticket{
			{
				ID:        types.NewTicketID(),
				Status:    types.TicketStatusOpen,
				CreatedAt: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				SlackThread: &slack.Thread{
					ChannelID: thread.ChannelID,
					ThreadID:  thread.ThreadID,
				},
			},
			{
				ID:        types.NewTicketID(),
				Status:    types.TicketStatusPending,
				CreatedAt: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
				SlackThread: &slack.Thread{
					ChannelID: thread.ChannelID,
					ThreadID:  thread.ThreadID,
				},
			},
			{
				ID:        types.NewTicketID(),
				Status:    types.TicketStatusResolved,
				CreatedAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				SlackThread: &slack.Thread{
					ChannelID: thread.ChannelID,
					ThreadID:  thread.ThreadID,
				},
			},
		}

		// Put tickets
		for _, ticket := range tickets {
			gt.NoError(t, repo.PutTicket(ctx, *ticket))
		}

		t.Run("tickets in time range", func(t *testing.T) {
			start := time.Date(2024, 1, 1, 10, 30, 0, 0, time.UTC)
			end := time.Date(2024, 1, 1, 11, 30, 0, 0, time.UTC)

			result, err := repo.GetTicketsBySpan(ctx, start, end)
			gt.NoError(t, err)
			filtered := filterByIDs(result, []types.TicketID{tickets[1].ID})
			gt.Array(t, filtered).Length(1)
			gt.Value(t, filtered[0].ID).Equal(tickets[1].ID)
		})

		t.Run("tickets outside time range", func(t *testing.T) {
			start := time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC)
			end := time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC)

			result, err := repo.GetTicketsBySpan(ctx, start, end)
			gt.NoError(t, err)
			gt.Array(t, result).Length(0)
		})

		t.Run("all tickets in range", func(t *testing.T) {
			start := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
			end := time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC)

			result, err := repo.GetTicketsBySpan(ctx, start, end)
			gt.NoError(t, err)
			filtered := filterByIDs(result, []types.TicketID{tickets[0].ID, tickets[1].ID, tickets[2].ID})
			gt.Array(t, filtered).Length(3)
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

// Firestore用: 取得結果からテスト投入分のみ抽出してアサート
func filterByIDs(result []*ticketmodel.Ticket, ids []types.TicketID) []*ticketmodel.Ticket {
	idMap := make(map[types.TicketID]struct{})
	for _, id := range ids {
		idMap[id] = struct{}{}
	}
	var filtered []*ticketmodel.Ticket
	for _, t := range result {
		if _, ok := idMap[t.ID]; ok {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

func TestGetAlertWithoutEmbedding(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()
		thread := newTestThread()
		// Alert with embedding
		alertWithEmbedding := newTestAlert(&thread)
		alertWithEmbedding.Embedding = make([]float32, 256)
		// Alert without embedding
		alertWithoutEmbedding := newTestAlert(&thread)
		alertWithoutEmbedding.Embedding = nil
		// Put both alerts
		gt.NoError(t, repo.PutAlert(ctx, alertWithEmbedding))
		gt.NoError(t, repo.PutAlert(ctx, alertWithoutEmbedding))

		alerts, err := repo.GetAlertWithoutEmbedding(ctx)
		gt.NoError(t, err).Required()
		gt.Array(t, alerts).Any(func(a *alert.Alert) bool {
			return a.ID == alertWithoutEmbedding.ID
		})
		gt.Array(t, alerts).All(func(a *alert.Alert) bool {
			return len(a.Embedding) == 0
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

func TestFindNearestTicketsWithSpan(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()
		now := time.Now()

		// Generate random 256-dim embeddings
		emb1 := make([]float32, 256)
		for i := range emb1 {
			emb1[i] = rand.Float32()
		}
		emb2 := make([]float32, 256)
		copy(emb2, emb1)
		emb2[0] += 0.01 // Slightly different from emb1
		emb3 := make([]float32, 256)
		for i := range emb3 {
			emb3[i] = rand.Float32()
		}

		tickets := []ticketmodel.Ticket{
			{
				ID:        types.NewTicketID(),
				Embedding: emb1,
				CreatedAt: now.Add(-2 * time.Hour),
			},
			{
				ID:        types.NewTicketID(),
				Embedding: emb2,
				CreatedAt: now.Add(-1 * time.Hour),
			},
			{
				ID:        types.NewTicketID(),
				Embedding: emb3,
				CreatedAt: now.Add(1 * time.Hour),
			},
		}

		for _, ticket := range tickets {
			gt.NoError(t, repo.PutTicket(ctx, ticket))
		}

		begin := now.Add(-3 * time.Hour)
		end := now.Add(2 * time.Hour)
		queryEmbedding := make([]float32, 256)
		copy(queryEmbedding, emb1)
		queryEmbedding[0] += 0.005 // Slightly different from emb1, but closer to emb1 and emb2

		results, err := repo.FindNearestTicketsWithSpan(ctx, queryEmbedding, begin, end, 2)
		gt.NoError(t, err)
		gt.Array(t, results).Length(2)

		ticketIDs := make(map[types.TicketID]bool)
		for _, ticket := range results {
			ticketIDs[ticket.ID] = true
		}
		gt.Value(t, ticketIDs[tickets[0].ID]).Equal(true)
		gt.Value(t, ticketIDs[tickets[1].ID]).Equal(true)
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

func TestGetTicketsByStatusAndSpan(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()
		now := time.Now()

		// Create test tickets with different statuses and timestamps
		tickets := []ticketmodel.Ticket{
			{
				ID:        types.NewTicketID(),
				Status:    types.TicketStatusOpen,
				CreatedAt: now.Add(-2 * time.Hour),
			},
			{
				ID:        types.NewTicketID(),
				Status:    types.TicketStatusOpen,
				CreatedAt: now.Add(-1 * time.Hour),
			},
			{
				ID:        types.NewTicketID(),
				Status:    types.TicketStatusResolved,
				CreatedAt: now.Add(-1 * time.Hour),
			},
			{
				ID:        types.NewTicketID(),
				Status:    types.TicketStatusOpen,
				CreatedAt: now.Add(1 * time.Hour),
			},
		}

		// Put tickets
		for _, ticket := range tickets {
			gt.NoError(t, repo.PutTicket(ctx, ticket))
		}

		begin := now.Add(-3 * time.Hour)
		end := now

		results, err := repo.GetTicketsByStatusAndSpan(ctx, types.TicketStatusOpen, begin, end)
		gt.NoError(t, err)
		gt.Array(t, results).Longer(1)

		// Verify the results contain the expected tickets
		ticketIDs := make(map[types.TicketID]bool)
		for _, ticket := range results {
			ticketIDs[ticket.ID] = true
		}
		gt.Value(t, ticketIDs[tickets[0].ID]).Equal(true)
		gt.Value(t, ticketIDs[tickets[1].ID]).Equal(true)
		gt.Value(t, ticketIDs[tickets[2].ID]).Equal(false)
		gt.Value(t, ticketIDs[tickets[3].ID]).Equal(false)
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
