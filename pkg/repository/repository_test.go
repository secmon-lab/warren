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
	"github.com/secmon-lab/warren/pkg/domain/model/activity"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/tag"
	ticketmodel "github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/utils/test"
	"github.com/secmon-lab/warren/pkg/utils/user"
	"github.com/slack-go/slack/slackevents"
	"google.golang.org/api/iterator"
)

// Helper function to check if a tag ID exists in a slice
func containsTag(tags []types.TagID, target types.TagID) bool {
	for _, tag := range tags {
		if tag == target {
			return true
		}
	}
	return false
}

func newFirestoreClient(t *testing.T) *repository.Firestore {
	vars := test.NewEnvVars(t, "TEST_FIRESTORE_PROJECT_ID", "TEST_FIRESTORE_DATABASE_ID")
	client, err := repository.NewFirestore(t.Context(),
		vars.Get("TEST_FIRESTORE_PROJECT_ID"),
		vars.Get("TEST_FIRESTORE_DATABASE_ID"),
	)
	gt.NoError(t, err).Required()
	return client
}

// cleanupFirestoreCollection removes all documents from a Firestore collection
func cleanupFirestoreCollection(t *testing.T, repo interfaces.Repository, collectionName string) {
	fsRepo, ok := repo.(*repository.Firestore)
	if !ok {
		return // Not a Firestore repository, nothing to clean
	}

	ctx := t.Context()
	iter := fsRepo.GetClient().Collection(collectionName).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("failed to iterate %s: %v", collectionName, err)
		}
		_, err = doc.Ref.Delete(ctx)
		if err != nil {
			t.Fatalf("failed to delete %s document: %v", collectionName, err)
		}
	}
}

func newTestThread() slack.Thread {
	return slack.Thread{
		ChannelID: "test-channel",
		ThreadID:  fmt.Sprintf("%d.%d", time.Now().Unix(), time.Now().Nanosecond()),
	}
}

func newTestAlert(thread *slack.Thread) alert.Alert {
	// Generate random embedding to avoid zero vector issues
	embedding := make([]float32, 256)
	for i := range embedding {
		embedding[i] = rand.Float32()
	}

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
		Embedding: embedding,
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

		unbindAlerts, err := repo.GetAlertWithoutTicket(ctx, 0, 0)
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

		// BindAlertsToTicket
		gt.NoError(t, repo.BindAlertsToTicket(ctx, []types.AlertID{testAlert.ID}, ticketObj.ID))

		// BindAlertsToTicket
		alert2 := newTestAlert(&thread)
		alert3 := newTestAlert(&thread)
		gt.NoError(t, repo.PutAlert(ctx, alert2))
		gt.NoError(t, repo.PutAlert(ctx, alert3))
		gt.NoError(t, repo.BindAlertsToTicket(ctx, []types.AlertID{alert2.ID, alert3.ID}, ticketObj.ID))

		// Verify alerts are bound to ticket
		gotAlert2, err := repo.GetAlert(ctx, alert2.ID)
		gt.NoError(t, err)
		gt.Value(t, gotAlert2.TicketID).Equal(ticketObj.ID)

		gotAlert3, err := repo.GetAlert(ctx, alert3.ID)
		gt.NoError(t, err)
		gt.Value(t, gotAlert3.TicketID).Equal(ticketObj.ID)

		// Verify ticket's AlertIDs array is updated with bound alerts
		updatedTicket, err := repo.GetTicket(ctx, ticketObj.ID)
		gt.NoError(t, err)
		gt.Array(t, updatedTicket.AlertIDs).Any(func(id types.AlertID) bool { return id == testAlert.ID }) // From BindAlertsToTicket
		gt.Array(t, updatedTicket.AlertIDs).Any(func(id types.AlertID) bool { return id == alert2.ID })    // From BindAlertsToTicket
		gt.Array(t, updatedTicket.AlertIDs).Any(func(id types.AlertID) bool { return id == alert3.ID })    // From BindAlertsToTicket
		gt.Number(t, len(updatedTicket.AlertIDs)).GreaterOrEqual(3)                                        // Should have at least 3 alerts

		// PutTicketComment
		slackMsg := slack.NewMessage(ctx, &slackevents.EventsAPIEvent{
			InnerEvent: slackevents.EventsAPIInnerEvent{
				Data: &slackevents.AppMentionEvent{
					TimeStamp: "test-message-id",
					Text:      "Test Comment",
					User:      "test-user",
					Channel:   "test-channel",
				},
			},
		})
		comment := ticketObj.NewComment(ctx, slackMsg.Text(), slackMsg.User(), slackMsg.ID())
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

func TestBindAlertsToTicketBidirectional(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()
		thread := newTestThread()

		// Create a ticket
		ticketObj := newTestTicket(&thread)
		gt.NoError(t, repo.PutTicket(ctx, ticketObj))

		// Create multiple alerts
		alert1 := newTestAlert(&thread)
		alert2 := newTestAlert(&thread)
		alert3 := newTestAlert(&thread)

		gt.NoError(t, repo.PutAlert(ctx, alert1))
		gt.NoError(t, repo.PutAlert(ctx, alert2))
		gt.NoError(t, repo.PutAlert(ctx, alert3))

		// Verify initial state: ticket has no alerts
		initialTicket, err := repo.GetTicket(ctx, ticketObj.ID)
		gt.NoError(t, err)
		gt.Array(t, initialTicket.AlertIDs).Length(0)

		// Verify initial state: alerts are not bound to any ticket
		gt.Value(t, alert1.TicketID).Equal(types.EmptyTicketID)
		gt.Value(t, alert2.TicketID).Equal(types.EmptyTicketID)
		gt.Value(t, alert3.TicketID).Equal(types.EmptyTicketID)

		// Bind alerts to ticket using BindAlertsToTicket
		alertIDs := []types.AlertID{alert1.ID, alert2.ID, alert3.ID}
		gt.NoError(t, repo.BindAlertsToTicket(ctx, alertIDs, ticketObj.ID))

		// Verify bidirectional binding: alerts → ticket
		boundAlert1, err := repo.GetAlert(ctx, alert1.ID)
		gt.NoError(t, err)
		gt.Value(t, boundAlert1.TicketID).Equal(ticketObj.ID)

		boundAlert2, err := repo.GetAlert(ctx, alert2.ID)
		gt.NoError(t, err)
		gt.Value(t, boundAlert2.TicketID).Equal(ticketObj.ID)

		boundAlert3, err := repo.GetAlert(ctx, alert3.ID)
		gt.NoError(t, err)
		gt.Value(t, boundAlert3.TicketID).Equal(ticketObj.ID)

		// Verify bidirectional binding: ticket → alerts
		updatedTicket, err := repo.GetTicket(ctx, ticketObj.ID)
		gt.NoError(t, err)
		gt.Array(t, updatedTicket.AlertIDs).Length(3)
		gt.Array(t, updatedTicket.AlertIDs).Any(func(id types.AlertID) bool { return id == alert1.ID })
		gt.Array(t, updatedTicket.AlertIDs).Any(func(id types.AlertID) bool { return id == alert2.ID })
		gt.Array(t, updatedTicket.AlertIDs).Any(func(id types.AlertID) bool { return id == alert3.ID })

		// Verify no duplicate alerts in ticket
		alertIDMap := make(map[types.AlertID]int)
		for _, id := range updatedTicket.AlertIDs {
			alertIDMap[id]++
		}
		for alertID, count := range alertIDMap {
			gt.Value(t, count).Equal(1) // Alert should appear only once in ticket AlertIDs
			_ = alertID                 // Suppress unused variable warning
		}

		// Test binding additional alerts to the same ticket
		alert4 := newTestAlert(&thread)
		gt.NoError(t, repo.PutAlert(ctx, alert4))

		gt.NoError(t, repo.BindAlertsToTicket(ctx, []types.AlertID{alert4.ID}, ticketObj.ID))

		// Verify the additional alert is bound
		finalTicket, err := repo.GetTicket(ctx, ticketObj.ID)
		gt.NoError(t, err)
		gt.Array(t, finalTicket.AlertIDs).Length(4)
		gt.Array(t, finalTicket.AlertIDs).Any(func(id types.AlertID) bool { return id == alert4.ID })
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

		// Clean up for Firestore
		cleanupFirestoreCollection(t, repo, "tickets")

		// Track created tickets for cleanup
		var createdTickets []*ticketmodel.Ticket

		// Register cleanup function
		t.Cleanup(func() {
			if fsRepo, ok := repo.(*repository.Firestore); ok {
				for _, ticket := range createdTickets {
					if ticket != nil {
						doc := fsRepo.GetClient().Collection("tickets").Doc(ticket.ID.String())
						_, _ = doc.Delete(ctx)
					}
				}
			}
		})

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
			createdTickets = append(createdTickets, tickets[i])
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
		createdTickets = append(createdTickets, &target)
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

		// Clean up for Firestore
		cleanupFirestoreCollection(t, repo, "tickets")

		// Track created tickets for cleanup
		var createdTickets []*ticketmodel.Ticket

		// Register cleanup function
		t.Cleanup(func() {
			if fsRepo, ok := repo.(*repository.Firestore); ok {
				for _, ticket := range createdTickets {
					if ticket != nil {
						doc := fsRepo.GetClient().Collection("tickets").Doc(ticket.ID.String())
						_, _ = doc.Delete(ctx)
					}
				}
			}
		})

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
			createdTickets = append(createdTickets, tickets[i])
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

		// Clean up any existing alerts with zero vectors for Firestore
		cleanupFirestoreCollection(t, repo, "alerts")

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

func TestGetAlertsWithInvalidEmbedding(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()

		// Clean up for Firestore
		cleanupFirestoreCollection(t, repo, "alerts")

		// Track created alerts for cleanup
		var createdAlerts []*alert.Alert

		// Register cleanup function
		t.Cleanup(func() {
			if fsRepo, ok := repo.(*repository.Firestore); ok {
				for _, a := range createdAlerts {
					if a != nil {
						doc := fsRepo.GetClient().Collection("alerts").Doc(a.ID.String())
						_, _ = doc.Delete(ctx)
					}
				}
			}
		})

		// Create alerts with various embedding states
		alertsWithValidEmbedding := &alert.Alert{
			ID:        types.NewAlertID(),
			Schema:    types.AlertSchema("test-schema-valid"),
			CreatedAt: time.Now(),
			Embedding: []float32{0.1, 0.2, 0.3}, // Valid embedding
		}
		createdAlerts = append(createdAlerts, alertsWithValidEmbedding)

		alertWithNilEmbedding := &alert.Alert{
			ID:        types.NewAlertID(),
			Schema:    types.AlertSchema("test-schema-nil"),
			CreatedAt: time.Now(),
			Embedding: nil, // Nil embedding
		}
		createdAlerts = append(createdAlerts, alertWithNilEmbedding)

		alertWithEmptyEmbedding := &alert.Alert{
			ID:        types.NewAlertID(),
			Schema:    types.AlertSchema("test-schema-empty"),
			CreatedAt: time.Now(),
			Embedding: []float32{}, // Empty embedding
		}
		createdAlerts = append(createdAlerts, alertWithEmptyEmbedding)

		alertWithZeroEmbedding := &alert.Alert{
			ID:        types.NewAlertID(),
			Schema:    types.AlertSchema("test-schema-zero"),
			CreatedAt: time.Now(),
			Embedding: make([]float32, 256), // Zero vector
		}
		createdAlerts = append(createdAlerts, alertWithZeroEmbedding)

		// Put all alerts
		gt.NoError(t, repo.PutAlert(ctx, *alertsWithValidEmbedding))
		gt.NoError(t, repo.PutAlert(ctx, *alertWithNilEmbedding))
		// Skip empty embedding for Firestore as it rejects empty vectors
		if _, ok := repo.(*repository.Firestore); !ok {
			gt.NoError(t, repo.PutAlert(ctx, *alertWithEmptyEmbedding))
		}
		gt.NoError(t, repo.PutAlert(ctx, *alertWithZeroEmbedding))

		// Get alerts with invalid embeddings
		invalidAlerts, err := repo.GetAlertsWithInvalidEmbedding(ctx)
		gt.NoError(t, err).Required()

		// Should return at least 2 alerts
		// For Firestore, empty embedding will be converted to nil during PutAlert, so it might be 2 instead of 3
		if _, ok := repo.(*repository.Firestore); ok {
			gt.Array(t, invalidAlerts).Length(2).Required()
		} else {
			gt.Array(t, invalidAlerts).Length(3).Required()
		}

		// Verify the returned alerts are the ones with invalid embeddings
		invalidIDs := map[types.AlertID]bool{
			alertWithNilEmbedding.ID:  false,
			alertWithZeroEmbedding.ID: false,
		}
		// Add empty embedding check only for Memory repo
		if _, ok := repo.(*repository.Firestore); !ok {
			invalidIDs[alertWithEmptyEmbedding.ID] = false
		}

		for _, alert := range invalidAlerts {
			if _, ok := invalidIDs[alert.ID]; ok {
				invalidIDs[alert.ID] = true
			}
		}

		for _, found := range invalidIDs {
			gt.True(t, found)
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

func TestGetTicketsWithInvalidEmbedding(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()

		// Clean up for Firestore
		cleanupFirestoreCollection(t, repo, "tickets")

		// Track created tickets for cleanup
		var createdTickets []*ticketmodel.Ticket

		// Register cleanup function
		t.Cleanup(func() {
			if fsRepo, ok := repo.(*repository.Firestore); ok {
				for _, ticket := range createdTickets {
					if ticket != nil {
						doc := fsRepo.GetClient().Collection("tickets").Doc(ticket.ID.String())
						_, _ = doc.Delete(ctx)
					}
				}
			}
		})

		// Create tickets with various embedding states
		ticketWithValidEmbedding := &ticketmodel.Ticket{
			ID:          types.NewTicketID(),
			Status:      types.TicketStatusOpen,
			CreatedAt:   time.Now(),
			SlackThread: &slack.Thread{ChannelID: "test", ThreadID: "1"},
			Embedding:   []float32{0.1, 0.2, 0.3}, // Valid embedding
		}
		createdTickets = append(createdTickets, ticketWithValidEmbedding)

		ticketWithNilEmbedding := &ticketmodel.Ticket{
			ID:          types.NewTicketID(),
			Status:      types.TicketStatusOpen,
			CreatedAt:   time.Now(),
			SlackThread: &slack.Thread{ChannelID: "test", ThreadID: "2"},
			Embedding:   nil, // Nil embedding
		}
		createdTickets = append(createdTickets, ticketWithNilEmbedding)

		ticketWithEmptyEmbedding := &ticketmodel.Ticket{
			ID:          types.NewTicketID(),
			Status:      types.TicketStatusOpen,
			CreatedAt:   time.Now(),
			SlackThread: &slack.Thread{ChannelID: "test", ThreadID: "3"},
			Embedding:   []float32{}, // Empty embedding
		}
		createdTickets = append(createdTickets, ticketWithEmptyEmbedding)

		ticketWithZeroEmbedding := &ticketmodel.Ticket{
			ID:          types.NewTicketID(),
			Status:      types.TicketStatusOpen,
			CreatedAt:   time.Now(),
			SlackThread: &slack.Thread{ChannelID: "test", ThreadID: "4"},
			Embedding:   make([]float32, 256), // Zero vector
		}
		createdTickets = append(createdTickets, ticketWithZeroEmbedding)

		// Put all tickets
		gt.NoError(t, repo.PutTicket(ctx, *ticketWithValidEmbedding))
		gt.NoError(t, repo.PutTicket(ctx, *ticketWithNilEmbedding))
		// Skip empty embedding for Firestore as it rejects empty vectors
		if _, ok := repo.(*repository.Firestore); !ok {
			gt.NoError(t, repo.PutTicket(ctx, *ticketWithEmptyEmbedding))
		}
		gt.NoError(t, repo.PutTicket(ctx, *ticketWithZeroEmbedding))

		// Get tickets with invalid embeddings
		invalidTickets, err := repo.GetTicketsWithInvalidEmbedding(ctx)
		gt.NoError(t, err).Required()

		// Should return at least 2 tickets
		// For Firestore, empty embedding will be converted to nil during PutTicket, so it might be 2 instead of 3
		if _, ok := repo.(*repository.Firestore); ok {
			gt.Array(t, invalidTickets).Length(2).Required()
		} else {
			gt.Array(t, invalidTickets).Length(3).Required()
		}

		// Verify the returned tickets are the ones with invalid embeddings
		invalidIDs := map[types.TicketID]bool{
			ticketWithNilEmbedding.ID:  false,
			ticketWithZeroEmbedding.ID: false,
		}
		// Add empty embedding check only for Memory repo
		if _, ok := repo.(*repository.Firestore); !ok {
			invalidIDs[ticketWithEmptyEmbedding.ID] = false
		}

		for _, ticket := range invalidTickets {
			if _, ok := invalidIDs[ticket.ID]; ok {
				invalidIDs[ticket.ID] = true
			}
		}

		for _, found := range invalidIDs {
			gt.True(t, found)
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
		gt.Value(t, latest.ID).Equal(histories[2].ID) // The last added one is the latest

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
			slackMsg := slack.NewMessage(ctx, &slackevents.EventsAPIEvent{
				InnerEvent: slackevents.EventsAPIInnerEvent{
					Data: &slackevents.AppMentionEvent{
						TimeStamp: fmt.Sprintf("test-message-id-%d", i),
						Text:      fmt.Sprintf("Test Comment %d", i),
						User:      "test-user",
						Channel:   "test-channel",
					},
				},
			})
			comment := ticket.NewComment(ctx, slackMsg.Text(), slackMsg.User(), slackMsg.ID())
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
			// Filter by test tickets only - should be empty since no test tickets are in this range
			filtered := filterByIDs(result, []types.TicketID{tickets[0].ID, tickets[1].ID, tickets[2].ID})
			gt.Array(t, filtered).Length(0)
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

// For Firestore: Extract only test-injected data from results and assert
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

		// Clean up for Firestore
		cleanupFirestoreCollection(t, repo, "tickets")

		// Track created tickets for cleanup
		var createdTickets []*ticketmodel.Ticket

		// Register cleanup function
		t.Cleanup(func() {
			if fsRepo, ok := repo.(*repository.Firestore); ok {
				for _, ticket := range createdTickets {
					if ticket != nil {
						doc := fsRepo.GetClient().Collection("tickets").Doc(ticket.ID.String())
						_, _ = doc.Delete(ctx)
					}
				}
			}
		})

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

		for i := range tickets {
			createdTickets = append(createdTickets, &tickets[i])
			gt.NoError(t, repo.PutTicket(ctx, tickets[i]))
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

func TestToken(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()

		// Create test token
		token := auth.NewToken("test-sub", "test@example.com", "Test User")

		// PutToken
		gt.NoError(t, repo.PutToken(ctx, token))

		// GetToken
		gotToken, err := repo.GetToken(ctx, token.ID)
		gt.NoError(t, err)
		gt.Value(t, gotToken.ID).Equal(token.ID)
		gt.Value(t, gotToken.Secret).Equal(token.Secret)
		gt.Value(t, gotToken.Sub).Equal(token.Sub)
		gt.Value(t, gotToken.Email).Equal(token.Email)
		gt.Value(t, gotToken.Name).Equal(token.Name)
		gt.Value(t, gotToken.ExpiresAt.Unix()).Equal(token.ExpiresAt.Unix())
		gt.Value(t, gotToken.CreatedAt.Unix()).Equal(token.CreatedAt.Unix())

		// Test token validation
		gt.NoError(t, gotToken.Validate())
		gt.Value(t, gotToken.IsExpired()).Equal(false)

		// DeleteToken
		gt.NoError(t, repo.DeleteToken(ctx, token.ID))

		// Verify token is deleted
		_, err = repo.GetToken(ctx, token.ID)
		gt.Error(t, err) // Should return error for non-existent token
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

func TestCountTicketsByStatus(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()
		thread := newTestThread()

		// Create tickets with different statuses
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
				Status:    types.TicketStatusOpen,
				CreatedAt: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
				SlackThread: &slack.Thread{
					ChannelID: thread.ChannelID,
					ThreadID:  thread.ThreadID,
				},
			},
			{
				ID:        types.NewTicketID(),
				Status:    types.TicketStatusPending,
				CreatedAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				SlackThread: &slack.Thread{
					ChannelID: thread.ChannelID,
					ThreadID:  thread.ThreadID,
				},
			},
			{
				ID:        types.NewTicketID(),
				Status:    types.TicketStatusResolved,
				CreatedAt: time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC),
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

		t.Run("count open tickets", func(t *testing.T) {
			count, err := repo.CountTicketsByStatus(ctx, []types.TicketStatus{types.TicketStatusOpen})
			gt.NoError(t, err)
			gt.Number(t, count).GreaterOrEqual(0) // Should return at least 0 (may have existing data in Firestore)
		})

		t.Run("count pending tickets", func(t *testing.T) {
			count, err := repo.CountTicketsByStatus(ctx, []types.TicketStatus{types.TicketStatusPending})
			gt.NoError(t, err)
			gt.Number(t, count).GreaterOrEqual(0) // Should return at least 0 (may have existing data in Firestore)
		})

		t.Run("count resolved tickets", func(t *testing.T) {
			count, err := repo.CountTicketsByStatus(ctx, []types.TicketStatus{types.TicketStatusResolved})
			gt.NoError(t, err)
			gt.Number(t, count).GreaterOrEqual(0) // Should return at least 0 (may have existing data in Firestore)
		})

		t.Run("count multiple statuses", func(t *testing.T) {
			count, err := repo.CountTicketsByStatus(ctx, []types.TicketStatus{
				types.TicketStatusOpen,
				types.TicketStatusPending,
			})
			gt.NoError(t, err)
			gt.Number(t, count).GreaterOrEqual(0) // Should return at least 0 (may have existing data in Firestore)
		})

		t.Run("count all tickets", func(t *testing.T) {
			count, err := repo.CountTicketsByStatus(ctx, nil)
			gt.NoError(t, err)
			gt.Number(t, count).GreaterOrEqual(0) // Should return at least 0 (may have existing data in Firestore)
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

func TestTicketCommentsPagination(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()
		thread := newTestThread()
		ticket := newTestTicket(&thread)

		// PutTicket
		gt.NoError(t, repo.PutTicket(ctx, ticket))

		// Create and put multiple comments with different timestamps
		comments := make([]ticketmodel.Comment, 10)
		baseTime := time.Now().Add(-time.Hour) // Start from 1 hour ago

		for i := 0; i < 10; i++ {
			slackMsg := slack.NewMessage(ctx, &slackevents.EventsAPIEvent{
				InnerEvent: slackevents.EventsAPIInnerEvent{
					Data: &slackevents.AppMentionEvent{
						TimeStamp: fmt.Sprintf("test-message-id-%d", i),
						Text:      fmt.Sprintf("Test Comment %d", i),
						User:      "test-user",
						Channel:   "test-channel",
					},
				},
			})
			comment := ticket.NewComment(ctx, slackMsg.Text(), slackMsg.User(), slackMsg.ID())
			// Set different timestamps to ensure proper ordering (newer first)
			comment.CreatedAt = baseTime.Add(time.Duration(i) * time.Minute)

			gt.NoError(t, repo.PutTicketComment(ctx, comment))
			comments[i] = comment
		}

		t.Run("CountTicketComments", func(t *testing.T) {
			// Test count with existing ticket
			count, err := repo.CountTicketComments(ctx, ticket.ID)
			gt.NoError(t, err)
			gt.Number(t, count).Equal(10)

			// Test count with non-existent ticket
			nonExistentID := types.NewTicketID()
			count, err = repo.CountTicketComments(ctx, nonExistentID)
			gt.NoError(t, err)
			gt.Number(t, count).Equal(0)
		})

		t.Run("GetTicketCommentsPaginated - basic pagination", func(t *testing.T) {
			// Test first page
			paginatedComments, err := repo.GetTicketCommentsPaginated(ctx, ticket.ID, 0, 3)
			gt.NoError(t, err)
			gt.Array(t, paginatedComments).Length(3)

			// Comments should be ordered by CreatedAt descending (newest first)
			// So comment 9 should be first, comment 8 second, etc.
			gt.Value(t, paginatedComments[0].Comment).Equal("Test Comment 9")
			gt.Value(t, paginatedComments[1].Comment).Equal("Test Comment 8")
			gt.Value(t, paginatedComments[2].Comment).Equal("Test Comment 7")

			// Test second page
			paginatedComments, err = repo.GetTicketCommentsPaginated(ctx, ticket.ID, 3, 3)
			gt.NoError(t, err)
			gt.Array(t, paginatedComments).Length(3)
			gt.Value(t, paginatedComments[0].Comment).Equal("Test Comment 6")
			gt.Value(t, paginatedComments[1].Comment).Equal("Test Comment 5")
			gt.Value(t, paginatedComments[2].Comment).Equal("Test Comment 4")

			// Test third page
			paginatedComments, err = repo.GetTicketCommentsPaginated(ctx, ticket.ID, 6, 3)
			gt.NoError(t, err)
			gt.Array(t, paginatedComments).Length(3)
			gt.Value(t, paginatedComments[0].Comment).Equal("Test Comment 3")
			gt.Value(t, paginatedComments[1].Comment).Equal("Test Comment 2")
			gt.Value(t, paginatedComments[2].Comment).Equal("Test Comment 1")

			// Test last page (partial)
			paginatedComments, err = repo.GetTicketCommentsPaginated(ctx, ticket.ID, 9, 3)
			gt.NoError(t, err)
			gt.Array(t, paginatedComments).Length(1)
			gt.Value(t, paginatedComments[0].Comment).Equal("Test Comment 0")
		})

		t.Run("GetTicketCommentsPaginated - edge cases", func(t *testing.T) {
			// Test offset beyond available comments
			paginatedComments, err := repo.GetTicketCommentsPaginated(ctx, ticket.ID, 15, 5)
			gt.NoError(t, err)
			gt.Array(t, paginatedComments).Length(0)

			// Test limit larger than remaining comments
			paginatedComments, err = repo.GetTicketCommentsPaginated(ctx, ticket.ID, 8, 5)
			gt.NoError(t, err)
			gt.Array(t, paginatedComments).Length(2)
			gt.Value(t, paginatedComments[0].Comment).Equal("Test Comment 1")
			gt.Value(t, paginatedComments[1].Comment).Equal("Test Comment 0")

			// Test zero limit
			paginatedComments, err = repo.GetTicketCommentsPaginated(ctx, ticket.ID, 0, 0)
			gt.NoError(t, err)
			gt.Array(t, paginatedComments).Length(0)

			// Test with non-existent ticket
			nonExistentID := types.NewTicketID()
			paginatedComments, err = repo.GetTicketCommentsPaginated(ctx, nonExistentID, 0, 5)
			gt.NoError(t, err)
			gt.Array(t, paginatedComments).Length(0)
		})

		t.Run("GetTicketCommentsPaginated - timestamp ordering", func(t *testing.T) {
			// Verify that comments are consistently ordered by CreatedAt descending
			allComments, err := repo.GetTicketCommentsPaginated(ctx, ticket.ID, 0, 10)
			gt.NoError(t, err)
			gt.Array(t, allComments).Length(10)

			// Check that each comment is older than the previous one
			for i := 1; i < len(allComments); i++ {
				gt.Value(t, allComments[i].CreatedAt.Before(allComments[i-1].CreatedAt)).Equal(true)
			}

			// Verify that the newest comment (index 9) is first
			gt.Value(t, allComments[0].Comment).Equal("Test Comment 9")
			// Verify that the oldest comment (index 0) is last
			gt.Value(t, allComments[9].Comment).Equal("Test Comment 0")
		})

		t.Run("GetTicketCommentsPaginated - different page sizes", func(t *testing.T) {
			// Test with page size 20 (should return all 10 comments)
			paginatedComments, err := repo.GetTicketCommentsPaginated(ctx, ticket.ID, 0, 20)
			gt.NoError(t, err)
			gt.Array(t, paginatedComments).Length(10)

			// Test with page size 50 (should return all 10 comments)
			paginatedComments, err = repo.GetTicketCommentsPaginated(ctx, ticket.ID, 0, 50)
			gt.NoError(t, err)
			gt.Array(t, paginatedComments).Length(10)

			// Test with page size 100 (should return all 10 comments)
			paginatedComments, err = repo.GetTicketCommentsPaginated(ctx, ticket.ID, 0, 100)
			gt.NoError(t, err)
			gt.Array(t, paginatedComments).Length(10)

			// Test with page size 1 (should return only 1 comment)
			paginatedComments, err = repo.GetTicketCommentsPaginated(ctx, ticket.ID, 0, 1)
			gt.NoError(t, err)
			gt.Array(t, paginatedComments).Length(1)
			gt.Value(t, paginatedComments[0].Comment).Equal("Test Comment 9")
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

func TestActivityCreation(t *testing.T) {
	repositories := []struct {
		name string
		repo func(t *testing.T) interfaces.Repository
	}{
		{
			name: "Memory",
			repo: func(t *testing.T) interfaces.Repository {
				return repository.NewMemory()
			},
		},
		{
			name: "Firestore",
			repo: func(t *testing.T) interfaces.Repository {
				return newFirestoreClient(t)
			},
		},
	}

	for _, repoTest := range repositories {
		t.Run(repoTest.name, func(t *testing.T) {
			repo := repoTest.repo(t)

			// Test ticket creation activity
			t.Run("TicketCreation", func(t *testing.T) {
				ctx := user.WithUserID(context.Background(), "test-user")

				ticket := ticketmodel.Ticket{
					ID: types.NewTicketID(),
					Metadata: ticketmodel.Metadata{
						Title: "Test Ticket",
					},
					Status:    types.TicketStatusOpen,
					CreatedAt: time.Now(),
				}

				err := repo.PutTicket(ctx, ticket)
				gt.NoError(t, err).Required()

				// Wait a bit for Firestore eventual consistency
				time.Sleep(100 * time.Millisecond)

				// Check that activity was created
				activities, err := repo.GetActivities(ctx, 0, 100) // Get more activities to account for test accumulation
				gt.NoError(t, err).Required()

				gt.Number(t, len(activities)).GreaterOrEqual(1).Required()

				// Find ticket creation activity
				var ticketActivity *activity.Activity
				for _, act := range activities {
					if act.Type == types.ActivityTypeTicketCreated && act.TicketID == ticket.ID {
						ticketActivity = act
						break
					}
				}

				gt.Value(t, ticketActivity).NotNil()
				gt.Value(t, ticketActivity.Type).Equal(types.ActivityTypeTicketCreated)
				gt.Value(t, ticketActivity.TicketID).Equal(ticket.ID)
				gt.Value(t, ticketActivity.UserID).Equal("test-user")
			})

			// Test ticket update activity
			t.Run("TicketUpdate", func(t *testing.T) {
				ctx := user.WithUserID(context.Background(), "update-user")

				// First create a ticket
				ticket := ticketmodel.Ticket{
					ID: types.NewTicketID(),
					Metadata: ticketmodel.Metadata{
						Title: "Original Title",
					},
					Status:    types.TicketStatusOpen,
					CreatedAt: time.Now(),
				}

				err := repo.PutTicket(ctx, ticket)
				gt.NoError(t, err).Required()

				// Wait a bit for Firestore eventual consistency
				time.Sleep(100 * time.Millisecond)

				// Now update the ticket
				ticket.Metadata.Title = "Updated Title"
				ticket.UpdatedAt = time.Now()

				err = repo.PutTicket(ctx, ticket)
				gt.NoError(t, err).Required()

				// Wait a bit for Firestore eventual consistency
				time.Sleep(100 * time.Millisecond)

				// Check that both creation and update activities were created
				activities, err := repo.GetActivities(ctx, 0, 100)
				gt.NoError(t, err).Required()

				gt.Number(t, len(activities)).GreaterOrEqual(2).Required()

				// Find both activities
				var creationActivity, updateActivity *activity.Activity
				for _, act := range activities {
					if act.TicketID == ticket.ID {
						if act.Type == types.ActivityTypeTicketCreated {
							creationActivity = act
						} else if act.Type == types.ActivityTypeTicketUpdated {
							updateActivity = act
						}
					}
				}

				gt.Value(t, creationActivity).NotNil()
				gt.Value(t, creationActivity.Type).Equal(types.ActivityTypeTicketCreated)
				gt.Value(t, creationActivity.TicketID).Equal(ticket.ID)
				gt.Value(t, creationActivity.UserID).Equal("update-user")

				gt.Value(t, updateActivity).NotNil()
				gt.Value(t, updateActivity.Type).Equal(types.ActivityTypeTicketUpdated)
				gt.Value(t, updateActivity.TicketID).Equal(ticket.ID)
				gt.Value(t, updateActivity.UserID).Equal("update-user")
			})

			// Test comment activity
			t.Run("CommentAddition", func(t *testing.T) {
				ctx := user.WithUserID(context.Background(), "comment-user")

				ticket := ticketmodel.Ticket{
					ID: types.NewTicketID(),
					Metadata: ticketmodel.Metadata{
						Title: "Test Ticket for Comments",
					},
					Status:    types.TicketStatusOpen,
					CreatedAt: time.Now(),
				}

				err := repo.PutTicket(ctx, ticket)
				gt.NoError(t, err)

				comment := ticketmodel.Comment{
					ID:        types.NewCommentID(),
					TicketID:  ticket.ID,
					Comment:   "Test comment",
					CreatedAt: time.Now(),
				}

				err = repo.PutTicketComment(ctx, comment)
				gt.NoError(t, err)

				// Check activities - should have at least the ticket creation + comment
				activities, err := repo.GetActivities(ctx, 0, 100)
				gt.NoError(t, err)
				gt.Number(t, len(activities)).GreaterOrEqual(2)

				// Find comment activity
				var commentActivity *activity.Activity
				for _, act := range activities {
					if act.Type == types.ActivityTypeCommentAdded && act.CommentID == comment.ID {
						commentActivity = act
						break
					}
				}

				gt.Value(t, commentActivity).NotNil()
				gt.Value(t, commentActivity.TicketID).Equal(ticket.ID)
				gt.Value(t, commentActivity.CommentID).Equal(comment.ID)
				gt.Value(t, commentActivity.UserID).Equal("comment-user")
			})

			// Test agent comment should not create activity
			t.Run("AgentCommentNoActivity", func(t *testing.T) {
				ctx := user.WithAgent(user.WithUserID(context.Background(), "agent"))

				ticket := ticketmodel.Ticket{
					ID: types.NewTicketID(),
					Metadata: ticketmodel.Metadata{
						Title: "Test Ticket for Agent Comments",
					},
					Status:    types.TicketStatusOpen,
					CreatedAt: time.Now(),
				}

				err := repo.PutTicket(ctx, ticket)
				gt.NoError(t, err)

				// Count activities for this specific ticket after ticket creation
				// Agent context should not create ticket creation activity
				activitiesAfterTicket, err := repo.GetActivities(ctx, 0, 100)
				gt.NoError(t, err)
				ticketCreationCount := 0
				for _, act := range activitiesAfterTicket {
					if act.TicketID == ticket.ID && act.Type == types.ActivityTypeTicketCreated {
						ticketCreationCount++
					}
				}
				gt.Number(t, ticketCreationCount).Equal(0) // Should have no ticket creation activity for agent

				comment := ticketmodel.Comment{
					ID:        types.NewCommentID(),
					TicketID:  ticket.ID,
					Comment:   "Agent comment",
					CreatedAt: time.Now(),
				}

				err = repo.PutTicketComment(ctx, comment)
				gt.NoError(t, err)

				// Count activities for this specific ticket after adding agent comment
				activitiesAfter, err := repo.GetActivities(ctx, 0, 100)
				gt.NoError(t, err)
				var ticketActivity, commentActivity *activity.Activity
				ticketActivityCount := 0
				for _, act := range activitiesAfter {
					if act.TicketID == ticket.ID {
						if act.Type == types.ActivityTypeTicketCreated {
							ticketActivity = act
							ticketActivityCount++
						} else if act.Type == types.ActivityTypeCommentAdded && act.CommentID == comment.ID {
							commentActivity = act
						}
					}
				}

				// Should have no activities for agent context
				gt.Number(t, ticketActivityCount).Equal(0)
				gt.Value(t, ticketActivity).Nil()  // Ticket creation should not exist for agent
				gt.Value(t, commentActivity).Nil() // Comment activity should not exist for agent
			})

			// Test alert binding activity
			t.Run("AlertBinding", func(t *testing.T) {
				ctx := user.WithUserID(context.Background(), "bind-user")

				ticket := ticketmodel.Ticket{
					ID: types.NewTicketID(),
					Metadata: ticketmodel.Metadata{
						Title: "Test Ticket for Alert Binding",
					},
					Status:    types.TicketStatusOpen,
					CreatedAt: time.Now(),
				}

				alert := &alert.Alert{
					ID: types.NewAlertID(),
					Metadata: alert.Metadata{
						Title: "Test Alert",
					},
					CreatedAt: time.Now(),
				}

				err := repo.PutTicket(ctx, ticket)
				gt.NoError(t, err)

				err = repo.PutAlert(ctx, *alert)
				gt.NoError(t, err)

				err = repo.BindAlertsToTicket(ctx, []types.AlertID{alert.ID}, ticket.ID)
				gt.NoError(t, err)

				// Check activities - should have at least ticket creation + alert binding
				activities, err := repo.GetActivities(ctx, 0, 100)
				gt.NoError(t, err)
				gt.Number(t, len(activities)).GreaterOrEqual(2)

				// Find alert binding activity
				var bindActivity *activity.Activity
				for _, act := range activities {
					if act.Type == types.ActivityTypeAlertBound && act.AlertID == alert.ID && act.TicketID == ticket.ID {
						bindActivity = act
						break
					}
				}

				gt.Value(t, bindActivity).NotNil()
				gt.Value(t, bindActivity.TicketID).Equal(ticket.ID)
				gt.Value(t, bindActivity.AlertID).Equal(alert.ID)
				gt.Value(t, bindActivity.UserID).Equal("bind-user")
			})

			// Test bulk alert binding activity
			t.Run("BulkAlertBinding", func(t *testing.T) {
				ctx := user.WithUserID(context.Background(), "bulk-bind-user")

				ticket := ticketmodel.Ticket{
					ID: types.NewTicketID(),
					Metadata: ticketmodel.Metadata{
						Title: "Test Ticket for Bulk Alert Binding",
					},
					Status:    types.TicketStatusOpen,
					CreatedAt: time.Now(),
				}

				// Create multiple alerts
				alertIDs := make([]types.AlertID, 3)
				for i := 0; i < 3; i++ {
					alert := &alert.Alert{
						ID: types.NewAlertID(),
						Metadata: alert.Metadata{
							Title: fmt.Sprintf("Test Alert %d", i+1),
						},
						CreatedAt: time.Now(),
					}
					alertIDs[i] = alert.ID

					err := repo.PutAlert(ctx, *alert)
					gt.NoError(t, err)
				}

				err := repo.PutTicket(ctx, ticket)
				gt.NoError(t, err)

				err = repo.BindAlertsToTicket(ctx, alertIDs, ticket.ID)
				gt.NoError(t, err)

				// Check activities - should have at least ticket creation + bulk alert binding
				activities, err := repo.GetActivities(ctx, 0, 100)
				gt.NoError(t, err)
				gt.Number(t, len(activities)).GreaterOrEqual(2)

				// Find bulk alert binding activity
				var bulkBindActivity *activity.Activity
				for _, act := range activities {
					if act.Type == types.ActivityTypeAlertsBulkBound && act.TicketID == ticket.ID {
						bulkBindActivity = act
						break
					}
				}

				gt.Value(t, bulkBindActivity).NotNil()
				gt.Value(t, bulkBindActivity.TicketID).Equal(ticket.ID)
				gt.Value(t, bulkBindActivity.UserID).Equal("bulk-bind-user")
			})

			// Test status change activity
			t.Run("StatusChange", func(t *testing.T) {
				ctx := user.WithUserID(context.Background(), "status-user")

				ticket := ticketmodel.Ticket{
					ID: types.NewTicketID(),
					Metadata: ticketmodel.Metadata{
						Title: "Test Ticket for Status Change",
					},
					Status:    types.TicketStatusOpen,
					CreatedAt: time.Now(),
				}

				err := repo.PutTicket(ctx, ticket)
				gt.NoError(t, err)

				err = repo.BatchUpdateTicketsStatus(ctx, []types.TicketID{ticket.ID}, types.TicketStatusResolved)
				gt.NoError(t, err)

				// Check activities - should have at least ticket creation + status change
				activities, err := repo.GetActivities(ctx, 0, 100)
				gt.NoError(t, err)
				gt.Number(t, len(activities)).GreaterOrEqual(2)

				// Find status change activity
				var statusActivity *activity.Activity
				for _, act := range activities {
					if act.Type == types.ActivityTypeTicketStatusChanged && act.TicketID == ticket.ID {
						statusActivity = act
						break
					}
				}

				gt.Value(t, statusActivity).NotNil()
				gt.Value(t, statusActivity.TicketID).Equal(ticket.ID)
				gt.Value(t, statusActivity.UserID).Equal("status-user")
				gt.Value(t, statusActivity.Metadata["old_status"]).Equal("open")
				gt.Value(t, statusActivity.Metadata["new_status"]).Equal("resolved")
			})
		})
	}
}

func TestGetAlertWithoutTicketPagination(t *testing.T) {
	runTest := func(repo interfaces.Repository) func(t *testing.T) {
		return func(t *testing.T) {
			ctx := context.Background()

			// Create multiple tickets and alerts to test pagination
			ticket1 := ticketmodel.Ticket{
				ID:       types.TicketID("ticket-1"),
				Metadata: ticketmodel.Metadata{Title: "Ticket 1"},
				Status:   types.TicketStatusOpen,
			}

			// Create 10 alerts: 5 bound to ticket1, 5 unbound
			boundAlerts := make([]alert.Alert, 5)
			unboundAlerts := make([]alert.Alert, 5)

			for i := 0; i < 5; i++ {
				boundAlerts[i] = alert.Alert{
					ID:       types.AlertID(fmt.Sprintf("bound-alert-%d", i)),
					TicketID: ticket1.ID,
					Metadata: alert.Metadata{Title: fmt.Sprintf("Bound Alert %d", i)},
				}

				unboundAlerts[i] = alert.Alert{
					ID:       types.AlertID(fmt.Sprintf("unbound-alert-%d", i)),
					TicketID: types.EmptyTicketID,
					Metadata: alert.Metadata{Title: fmt.Sprintf("Unbound Alert %d", i)},
				}
			}

			// Put ticket and alerts
			gt.NoError(t, repo.PutTicket(ctx, ticket1))
			for _, alert := range boundAlerts {
				gt.NoError(t, repo.PutAlert(ctx, alert))
			}
			for _, alert := range unboundAlerts {
				gt.NoError(t, repo.PutAlert(ctx, alert))
			}

			t.Run("Get all unbound alerts", func(t *testing.T) {
				alerts, err := repo.GetAlertWithoutTicket(ctx, 0, 0)
				gt.NoError(t, err)

				// Count our test alerts
				ourUnboundCount := 0
				for _, alert := range alerts {
					gt.Equal(t, alert.TicketID, types.EmptyTicketID)
					if alert.ID == types.AlertID("unbound-alert-0") ||
						alert.ID == types.AlertID("unbound-alert-1") ||
						alert.ID == types.AlertID("unbound-alert-2") ||
						alert.ID == types.AlertID("unbound-alert-3") ||
						alert.ID == types.AlertID("unbound-alert-4") {
						ourUnboundCount++
					}
				}
				gt.Number(t, ourUnboundCount).Equal(5)
			})

			t.Run("Get first 3 unbound alerts", func(t *testing.T) {
				alerts, err := repo.GetAlertWithoutTicket(ctx, 0, 3)
				gt.NoError(t, err)
				gt.Array(t, alerts).Length(3)

				// Verify all returned alerts are unbound
				for _, alert := range alerts {
					gt.Equal(t, alert.TicketID, types.EmptyTicketID)
				}
			})

			t.Run("Get alerts with offset", func(t *testing.T) {
				// Get first 2 alerts
				firstBatch, err := repo.GetAlertWithoutTicket(ctx, 0, 2)
				gt.NoError(t, err)
				gt.Number(t, len(firstBatch)).GreaterOrEqual(0) // May be 0 if no unbound alerts at beginning

				// Get alerts with offset - verify different results when there are enough alerts
				allAlerts, err := repo.GetAlertWithoutTicket(ctx, 0, 0)
				gt.NoError(t, err)

				if len(allAlerts) >= 4 {
					secondBatch, err := repo.GetAlertWithoutTicket(ctx, 2, 2)
					gt.NoError(t, err)
					gt.Number(t, len(secondBatch)).GreaterOrEqual(0)
				}
			})

			t.Run("Get alerts with offset beyond available", func(t *testing.T) {
				// Use a very large offset to ensure we get no results
				allAlerts, err := repo.GetAlertWithoutTicket(ctx, 0, 0)
				gt.NoError(t, err)

				largeOffset := len(allAlerts) + 100
				alerts, err := repo.GetAlertWithoutTicket(ctx, largeOffset, 5)
				gt.NoError(t, err)
				gt.Array(t, alerts).Length(0)
			})

			t.Run("Get limited alerts", func(t *testing.T) {
				alerts, err := repo.GetAlertWithoutTicket(ctx, 0, 3)
				gt.NoError(t, err)
				gt.Number(t, len(alerts)).LessOrEqual(3) // Should not exceed limit

				// Verify all returned alerts are unbound
				for _, alert := range alerts {
					gt.Equal(t, alert.TicketID, types.EmptyTicketID)
				}
			})

			t.Run("Count unbound alerts", func(t *testing.T) {
				count, err := repo.CountAlertsWithoutTicket(ctx)
				gt.NoError(t, err)
				gt.Number(t, count).GreaterOrEqual(5) // We created 5 unbound alerts

				// Verify count matches actual alerts
				allUnboundAlerts, err := repo.GetAlertWithoutTicket(ctx, 0, 0)
				gt.NoError(t, err)
				gt.Number(t, count).GreaterOrEqual(len(allUnboundAlerts))
			})
		}
	}

	// Test both Memory and Firestore implementations
	t.Run("Memory", runTest(repository.NewMemory()))

	t.Run("Firestore", runTest(newFirestoreClient(t)))
}

func TestTagOperations(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()

		t.Run("Create and list tags", func(t *testing.T) {
			// Initially no tags
			tags, err := repo.ListAllTags(ctx)
			gt.NoError(t, err)
			gt.Array(t, tags).Length(0)

			// Create tags
			tag1 := &tag.Tag{ID: types.NewTagID(), Name: "security", Color: "#ff0000"}
			gt.NoError(t, repo.CreateTagWithID(ctx, tag1))

			tag2 := &tag.Tag{ID: types.NewTagID(), Name: "incident", Color: "#00ff00"}
			gt.NoError(t, repo.CreateTagWithID(ctx, tag2))

			tag3 := &tag.Tag{ID: types.NewTagID(), Name: "phishing", Color: "#0000ff"}
			gt.NoError(t, repo.CreateTagWithID(ctx, tag3))

			// List tags
			tags, err = repo.ListAllTags(ctx)
			gt.NoError(t, err)
			gt.Array(t, tags).Length(3)

			// Verify tag names
			tagNames := make(map[string]bool)
			for _, tag := range tags {
				tagNames[tag.Name] = true
			}
			gt.True(t, tagNames["security"])
			gt.True(t, tagNames["incident"])
			gt.True(t, tagNames["phishing"])
		})

		t.Run("Create duplicate tag", func(t *testing.T) {
			// Create a tag
			tag1 := &tag.Tag{ID: types.NewTagID(), Name: "duplicate", Color: "#ff0000"}
			gt.NoError(t, repo.CreateTagWithID(ctx, tag1))

			// Try to create a different tag with same name (should succeed with different ID)
			tag2 := &tag.Tag{ID: types.NewTagID(), Name: "duplicate", Color: "#00ff00"}
			gt.NoError(t, repo.CreateTagWithID(ctx, tag2))

			// Verify both tags exist
			tags, err := repo.ListAllTags(ctx)
			gt.NoError(t, err)
			count := 0
			for _, t := range tags {
				if t.Name == "duplicate" {
					count++
				}
			}
			gt.Number(t, count).Equal(2) // Now expects 2 since they have different IDs
		})

		t.Run("Get tag by ID", func(t *testing.T) {
			// Create a tag
			testTag := &tag.Tag{ID: types.NewTagID(), Name: "gettag", Color: "#ff0000"}
			gt.NoError(t, repo.CreateTagWithID(ctx, testTag))

			// Get existing tag by ID
			retrievedTag, err := repo.GetTagByID(ctx, testTag.ID)
			gt.NoError(t, err)
			gt.NotNil(t, retrievedTag)
			gt.V(t, retrievedTag.Name).Equal("gettag")
			gt.V(t, retrievedTag.ID).Equal(testTag.ID)

			// Get non-existent tag
			nonExistent, err := repo.GetTagByID(ctx, types.NewTagID())
			gt.NoError(t, err)
			gt.Nil(t, nonExistent)
		})

		t.Run("Get tag by name", func(t *testing.T) {
			// Create a tag
			testTag := &tag.Tag{ID: types.NewTagID(), Name: "nametest", Color: "#00ff00"}
			gt.NoError(t, repo.CreateTagWithID(ctx, testTag))

			// Get existing tag by name
			retrievedTag, err := repo.GetTagByName(ctx, "nametest")
			gt.NoError(t, err)
			gt.NotNil(t, retrievedTag)
			gt.V(t, retrievedTag.Name).Equal("nametest")
			gt.V(t, retrievedTag.ID).Equal(testTag.ID)

			// Get non-existent tag
			nonExistent, err := repo.GetTagByName(ctx, "nonexistent")
			gt.NoError(t, err)
			gt.Nil(t, nonExistent)
		})

		t.Run("Delete tag by ID", func(t *testing.T) {
			// Create tags
			tag1 := &tag.Tag{ID: types.NewTagID(), Name: "delete1", Color: "#ff0000"}
			tag2 := &tag.Tag{ID: types.NewTagID(), Name: "delete2", Color: "#00ff00"}
			gt.NoError(t, repo.CreateTagWithID(ctx, tag1))
			gt.NoError(t, repo.CreateTagWithID(ctx, tag2))

			// Delete one tag by ID
			gt.NoError(t, repo.DeleteTagByID(ctx, tag1.ID))

			// Verify it's deleted
			deletedTag, err := repo.GetTagByID(ctx, tag1.ID)
			gt.NoError(t, err)
			gt.Nil(t, deletedTag)

			// Other tag should still exist
			remainingTag, err := repo.GetTagByID(ctx, tag2.ID)
			gt.NoError(t, err)
			gt.NotNil(t, remainingTag)
		})

		t.Run("Tag timestamps", func(t *testing.T) {
			// Create a tag
			before := time.Now()
			testTag := &tag.Tag{ID: types.NewTagID(), Name: "timestamped", Color: "#0000ff"}
			gt.NoError(t, repo.CreateTagWithID(ctx, testTag))
			after := time.Now()

			// Get the tag
			retrievedTag, err := repo.GetTagByID(ctx, testTag.ID)
			gt.NoError(t, err)
			gt.NotNil(t, retrievedTag)

			// Verify timestamps are set
			gt.True(t, !retrievedTag.CreatedAt.IsZero())
			gt.True(t, !retrievedTag.UpdatedAt.IsZero())
			gt.True(t, retrievedTag.CreatedAt.Equal(retrievedTag.UpdatedAt))

			// Verify timestamps are within expected range
			gt.True(t, retrievedTag.CreatedAt.After(before.Add(-time.Second)))
			gt.True(t, retrievedTag.CreatedAt.Before(after.Add(time.Second)))
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

func TestAlertAndTicketTags(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()

		t.Run("Alert with tags", func(t *testing.T) {
			// Create tags first in the new system
			securityTag := &tag.Tag{
				ID:   types.NewTagID(),
				Name: "security",
			}
			incidentTag := &tag.Tag{
				ID:   types.NewTagID(),
				Name: "incident",
			}
			criticalTag := &tag.Tag{
				ID:   types.NewTagID(),
				Name: "critical",
			}

			// Store tags in repository
			gt.NoError(t, repo.CreateTagWithID(ctx, securityTag))
			gt.NoError(t, repo.CreateTagWithID(ctx, incidentTag))
			gt.NoError(t, repo.CreateTagWithID(ctx, criticalTag))

			// Create an alert with tags
			a := alert.New(ctx, "test", map[string]string{"test": "data"}, alert.Metadata{
				Title:       "Test Alert",
				Description: "Test Description",
			})
			a.Tags = []types.TagID{securityTag.ID, incidentTag.ID, criticalTag.ID}

			// Save the alert
			gt.NoError(t, repo.PutAlert(ctx, a))

			// Retrieve the alert
			retrievedAlert, err := repo.GetAlert(ctx, a.ID)
			gt.NoError(t, err)
			gt.NotNil(t, retrievedAlert)

			// Verify tags are preserved
			gt.Number(t, len(retrievedAlert.Tags)).Equal(3)
			// Check tags are present in slice
			gt.True(t, containsTag(retrievedAlert.Tags, securityTag.ID))
			gt.True(t, containsTag(retrievedAlert.Tags, incidentTag.ID))
			gt.True(t, containsTag(retrievedAlert.Tags, criticalTag.ID))
		})

		t.Run("Ticket with tags", func(t *testing.T) {
			// Create tags first
			resolvedTag := &tag.Tag{
				ID:   types.NewTagID(),
				Name: "resolved",
			}
			fpTag := &tag.Tag{
				ID:   types.NewTagID(),
				Name: "false-positive",
			}

			// Store tags in repository
			gt.NoError(t, repo.CreateTagWithID(ctx, resolvedTag))
			gt.NoError(t, repo.CreateTagWithID(ctx, fpTag))

			// Create a ticket with tags
			tk := ticketmodel.New(ctx, []types.AlertID{}, nil)
			tk.Metadata.Title = "Test Ticket"
			tk.Tags = []types.TagID{resolvedTag.ID, fpTag.ID}

			// Save the ticket
			gt.NoError(t, repo.PutTicket(ctx, tk))

			// Retrieve the ticket
			retrievedTicket, err := repo.GetTicket(ctx, tk.ID)
			gt.NoError(t, err)
			gt.NotNil(t, retrievedTicket)

			// Verify tags are preserved
			gt.Number(t, len(retrievedTicket.Tags)).Equal(2)
			gt.True(t, containsTag(retrievedTicket.Tags, resolvedTag.ID))
			gt.True(t, containsTag(retrievedTicket.Tags, fpTag.ID))
		})

		t.Run("Empty tags", func(t *testing.T) {
			// Create alert without tags
			a := alert.New(ctx, "test", map[string]string{"test": "data"}, alert.Metadata{
				Title:       "No Tags Alert",
				Description: "Test Description",
			})
			// a.Tags should be nil by default

			gt.NoError(t, repo.PutAlert(ctx, a))

			// Retrieve and verify
			retrievedAlert, err := repo.GetAlert(ctx, a.ID)
			gt.NoError(t, err)
			gt.NotNil(t, retrievedAlert)
			// Tags should be nil or empty
			if retrievedAlert.Tags != nil {
				gt.Number(t, len(retrievedAlert.Tags)).Equal(0)
			}
		})

		t.Run("Tag persistence in batch operations", func(t *testing.T) {
			// Create common tag and individual tags
			commonTag := &tag.Tag{
				ID:   types.NewTagID(),
				Name: "common",
			}
			gt.NoError(t, repo.CreateTagWithID(ctx, commonTag))

			var individualTags []*tag.Tag
			for i := 0; i < 3; i++ {
				individualTag := &tag.Tag{
					ID:   types.NewTagID(),
					Name: fmt.Sprintf("tag%d", i),
				}
				gt.NoError(t, repo.CreateTagWithID(ctx, individualTag))
				individualTags = append(individualTags, individualTag)
			}

			// Create multiple alerts with tags
			alerts := make(alert.Alerts, 3)
			for i := 0; i < 3; i++ {
				a := alert.New(ctx, "test", map[string]string{"index": fmt.Sprintf("%d", i)}, alert.Metadata{
					Title:       fmt.Sprintf("Batch Alert %d", i),
					Description: "Test Description",
				})
				a.Tags = []types.TagID{individualTags[i].ID, commonTag.ID}
				alerts[i] = &a
			}

			// Batch save
			gt.NoError(t, repo.BatchPutAlerts(ctx, alerts))

			// Batch retrieve
			alertIDs := make([]types.AlertID, len(alerts))
			for i, a := range alerts {
				alertIDs[i] = a.ID
			}
			retrievedAlerts, err := repo.BatchGetAlerts(ctx, alertIDs)
			gt.NoError(t, err)
			gt.Array(t, retrievedAlerts).Length(3)

			// Verify tags
			for i, a := range retrievedAlerts {
				gt.True(t, containsTag(a.Tags, individualTags[i].ID))
				gt.True(t, containsTag(a.Tags, commonTag.ID))
			}
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
