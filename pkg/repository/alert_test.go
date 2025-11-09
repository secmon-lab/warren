package repository_test

import (
	"context"
	"fmt"
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
	"github.com/slack-go/slack/slackevents"
)

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
		for i := range 10 {
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

func TestFindNearestAlerts(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()

		alerts := alert.Alerts{}
		for i := range 10 {
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
		// Firestore rejects empty and zero-magnitude vectors, so only test with Memory
		_ = repo.PutAlert(ctx, *alertWithEmptyEmbedding)
		_ = repo.PutAlert(ctx, *alertWithZeroEmbedding)

		// Get alerts with invalid embeddings
		invalidAlerts, err := repo.GetAlertsWithInvalidEmbedding(ctx)
		gt.NoError(t, err).Required()

		// Should contain at least our test alerts with invalid embeddings
		gt.Number(t, len(invalidAlerts)).GreaterOrEqual(2)

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

	// This test only runs with Memory repository because Firestore now rejects
	// invalid embeddings at the Put* method level
	t.Run("Memory", func(t *testing.T) {
		repo := repository.NewMemory()
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

	// This test only runs with Memory repository because Firestore now rejects
	// invalid embeddings at the Put* method level
	t.Run("Memory", func(t *testing.T) {
		repo := repository.NewMemory()
		testFn(t, repo)
	})
}

func TestGetAlertWithoutTicketPagination(t *testing.T) {
	runTest := func(repo interfaces.Repository) func(t *testing.T) {
		return func(t *testing.T) {
			ctx := context.Background()

			// Generate random embedding for ticket
			ticketEmb := make([]float32, 256)
			for i := range ticketEmb {
				ticketEmb[i] = rand.Float32()
			}

			// Create multiple tickets and alerts to test pagination
			ticket1 := ticketmodel.Ticket{
				ID:        types.TicketID("ticket-1"),
				Metadata:  ticketmodel.Metadata{Title: "Ticket 1"},
				Status:    types.TicketStatusOpen,
				Embedding: ticketEmb,
			}

			// Create 10 alerts: 5 bound to ticket1, 5 unbound
			boundAlerts := make([]alert.Alert, 5)
			unboundAlerts := make([]alert.Alert, 5)

			for i := range 5 {
				// Generate random embedding for bound alert
				boundEmb := make([]float32, 256)
				for j := range boundEmb {
					boundEmb[j] = rand.Float32()
				}

				boundAlerts[i] = alert.Alert{
					ID:        types.AlertID(fmt.Sprintf("bound-alert-%d", i)),
					TicketID:  ticket1.ID,
					Metadata:  alert.Metadata{Title: fmt.Sprintf("Bound Alert %d", i)},
					Embedding: boundEmb,
				}

				// Generate random embedding for unbound alert
				unboundEmb := make([]float32, 256)
				for j := range unboundEmb {
					unboundEmb[j] = rand.Float32()
				}

				unboundAlerts[i] = alert.Alert{
					ID:        types.AlertID(fmt.Sprintf("unbound-alert-%d", i)),
					TicketID:  types.EmptyTicketID,
					Metadata:  alert.Metadata{Title: fmt.Sprintf("Unbound Alert %d", i)},
					Embedding: unboundEmb,
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
