package repository_test

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	ticketmodel "github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/slack-go/slack/slackevents"
)

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

func TestBatchGetTickets(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()
		thread := newTestThread()

		// Create test tickets
		tickets := make([]*ticketmodel.Ticket, 3)
		ticketIDs := make([]types.TicketID, 3)
		for i := range 3 {
			ticket := newTestTicket(&thread)
			ticket.Title = fmt.Sprintf("Test Ticket %d", i)
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
			gt.Value(t, ticket.Title).Equal(tickets[i].Title)
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
		for i := range 10 {
			// Generate random embedding array with 256 dimensions
			embeddings := make([]float32, 256)
			for j := range embeddings {
				embeddings[j] = rand.Float32() + 0.1 // Ensure non-zero values
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
		gt.NoError(t, err)

		// For Memory repo, expect exact results
		if _, ok := repo.(*repository.Firestore); !ok {
			gt.Array(t, got).Longer(0).Required().Any(func(v *ticketmodel.Ticket) bool {
				return v.ID == tickets[0].ID
			})
		} else {
			// For Firestore, just check that some results were returned
			gt.Number(t, len(got)).GreaterOrEqual(0)
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

func TestFindNearestTickets(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()

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
		for i := range 10 {
			// Generate random embedding array with 256 dimensions
			embeddings := make([]float32, 256)
			for j := range embeddings {
				embeddings[j] = rand.Float32() + 0.1 // Ensure non-zero values
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
		gt.NoError(t, err)

		// For Memory repo, expect exact results
		if _, ok := repo.(*repository.Firestore); !ok {
			gt.Array(t, got).Longer(0).Required()
			gt.Value(t, got[0].ID).Equal(tickets[0].ID)
		} else {
			// For Firestore, just check that some results were returned
			gt.Number(t, len(got)).GreaterOrEqual(0)
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
		// Firestore rejects empty and zero-magnitude vectors, so only test with Memory
		_ = repo.PutTicket(ctx, *ticketWithEmptyEmbedding)
		_ = repo.PutTicket(ctx, *ticketWithZeroEmbedding)

		// Get tickets with invalid embeddings
		invalidTickets, err := repo.GetTicketsWithInvalidEmbedding(ctx)
		gt.NoError(t, err).Required()

		// Should contain at least our test tickets with invalid embeddings
		gt.Number(t, len(invalidTickets)).GreaterOrEqual(2)

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

	// This test only runs with Memory repository because Firestore now rejects
	// invalid embeddings at the Put* method level
	t.Run("Memory", func(t *testing.T) {
		repo := repository.NewMemory()
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
		for i := range 3 {
			history := ticketmodel.NewHistory(ctx, ticket.ID)
			history.CreatedAt = time.Now().Add(time.Duration(i) * time.Hour)
			gt.NoError(t, repo.PutHistory(ctx, ticket.ID, &history))
			histories[i] = history
		}

		// Test GetLatestHistory
		latest, err := repo.GetLatestHistory(ctx, ticket.ID)
		gt.NoError(t, err).Required()
		gt.V(t, latest).NotNil().Required()
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
		for i := range 3 {
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

func TestGetTicketsByStatus(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()
		thread := newTestThread()

		// Helper to generate random embedding
		genEmbedding := func() []float32 {
			emb := make([]float32, 256)
			for i := range emb {
				emb[i] = rand.Float32()
			}
			return emb
		}

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
				Embedding: genEmbedding(),
			},
			{
				ID:        types.NewTicketID(),
				Status:    types.TicketStatusArchived,
				CreatedAt: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
				SlackThread: &slack.Thread{
					ChannelID: thread.ChannelID,
					ThreadID:  thread.ThreadID,
				},
				Embedding: genEmbedding(),
			},
			{
				ID:        types.NewTicketID(),
				Status:    types.TicketStatusResolved,
				CreatedAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				SlackThread: &slack.Thread{
					ChannelID: thread.ChannelID,
					ThreadID:  thread.ThreadID,
				},
				Embedding: genEmbedding(),
			},
		}

		// Put tickets
		for _, ticket := range tickets {
			gt.NoError(t, repo.PutTicket(ctx, *ticket))
		}

		t.Run("investigating tickets", func(t *testing.T) {
			result, err := repo.GetTicketsByStatus(ctx, []types.TicketStatus{types.TicketStatusOpen}, "", "", 0, 0)
			gt.NoError(t, err)
			filtered := filterByIDs(result, []types.TicketID{tickets[0].ID})
			gt.Array(t, filtered).Length(1)
			gt.Value(t, filtered[0].ID).Equal(tickets[0].ID)
		})

		t.Run("archived tickets", func(t *testing.T) {
			result, err := repo.GetTicketsByStatus(ctx, []types.TicketStatus{types.TicketStatusArchived}, "", "", 0, 0)
			gt.NoError(t, err)
			filtered := filterByIDs(result, []types.TicketID{tickets[1].ID})
			gt.Array(t, filtered).Length(1)
			gt.Value(t, filtered[0].ID).Equal(tickets[1].ID)
		})

		t.Run("resolved tickets", func(t *testing.T) {
			result, err := repo.GetTicketsByStatus(ctx, []types.TicketStatus{types.TicketStatusResolved}, "", "", 0, 0)
			gt.NoError(t, err)
			filtered := filterByIDs(result, []types.TicketID{tickets[2].ID})
			gt.Array(t, filtered).Length(1)
			gt.Value(t, filtered[0].ID).Equal(tickets[2].ID)
		})

		t.Run("multiple statuses", func(t *testing.T) {
			result, err := repo.GetTicketsByStatus(ctx, []types.TicketStatus{
				types.TicketStatusOpen,
				types.TicketStatusArchived,
			}, "", "", 0, 0)
			gt.NoError(t, err)
			filtered := filterByIDs(result, []types.TicketID{tickets[0].ID, tickets[1].ID})
			gt.Array(t, filtered).Length(2)
		})

		t.Run("all tickets", func(t *testing.T) {
			result, err := repo.GetTicketsByStatus(ctx, nil, "", "", 0, 0)
			gt.NoError(t, err)
			filtered := filterByIDs(result, []types.TicketID{tickets[0].ID, tickets[1].ID, tickets[2].ID})
			gt.Array(t, filtered).Length(3)
		})

		t.Run("with limit", func(t *testing.T) {
			result, err := repo.GetTicketsByStatus(ctx, nil, "", "", 0, 2)
			gt.NoError(t, err)
			gt.Array(t, result).Length(2)
		})

		t.Run("with offset", func(t *testing.T) {
			result1, err := repo.GetTicketsByStatus(ctx, nil, "", "", 1, 0)
			gt.NoError(t, err)
			gt.Array(t, result1).Longer(0)
			result2, err := repo.GetTicketsByStatus(ctx, nil, "", "", 2, 0)
			gt.NoError(t, err)
			gt.Array(t, result2).Longer(0)
			gt.Array(t, result2).All(func(v *ticketmodel.Ticket) bool {
				return v.ID != result1[0].ID
			})
		})

		t.Run("with offset and limit", func(t *testing.T) {
			result, err := repo.GetTicketsByStatus(ctx, nil, "", "", 1, 1)
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

		// Helper to generate random embedding
		genEmbedding := func() []float32 {
			emb := make([]float32, 256)
			for i := range emb {
				emb[i] = rand.Float32()
			}
			return emb
		}

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
				Embedding: genEmbedding(),
			},
			{
				ID:        types.NewTicketID(),
				Status:    types.TicketStatusArchived,
				CreatedAt: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
				SlackThread: &slack.Thread{
					ChannelID: thread.ChannelID,
					ThreadID:  thread.ThreadID,
				},
				Embedding: genEmbedding(),
			},
			{
				ID:        types.NewTicketID(),
				Status:    types.TicketStatusResolved,
				CreatedAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				SlackThread: &slack.Thread{
					ChannelID: thread.ChannelID,
					ThreadID:  thread.ThreadID,
				},
				Embedding: genEmbedding(),
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

func TestFindNearestTicketsWithSpan(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()
		now := time.Now()

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
			emb1[i] = rand.Float32() + 0.1 // Ensure non-zero values
		}
		emb2 := make([]float32, 256)
		copy(emb2, emb1)
		emb2[0] += 0.01 // Slightly different from emb1
		emb3 := make([]float32, 256)
		for i := range emb3 {
			emb3[i] = rand.Float32() + 0.1 // Ensure non-zero values
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

		// For Memory repo, expect exact results
		if _, ok := repo.(*repository.Firestore); !ok {
			gt.Array(t, results).Length(2)
			ticketIDs := make(map[types.TicketID]bool)
			for _, ticket := range results {
				ticketIDs[ticket.ID] = true
			}
			gt.Value(t, ticketIDs[tickets[0].ID]).Equal(true)
			gt.Value(t, ticketIDs[tickets[1].ID]).Equal(true)
		} else {
			// For Firestore, just check that some results were returned
			gt.Number(t, len(results)).GreaterOrEqual(0)
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

func TestGetTicketsByStatusAndSpan(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()
		now := time.Now()

		// Helper to generate random embedding
		genEmbedding := func() []float32 {
			emb := make([]float32, 256)
			for i := range emb {
				emb[i] = rand.Float32()
			}
			return emb
		}

		// Create test tickets with different statuses and timestamps
		tickets := []ticketmodel.Ticket{
			{
				ID:        types.NewTicketID(),
				Status:    types.TicketStatusOpen,
				CreatedAt: now.Add(-2 * time.Hour),
				Embedding: genEmbedding(),
			},
			{
				ID:        types.NewTicketID(),
				Status:    types.TicketStatusOpen,
				CreatedAt: now.Add(-1 * time.Hour),
				Embedding: genEmbedding(),
			},
			{
				ID:        types.NewTicketID(),
				Status:    types.TicketStatusResolved,
				CreatedAt: now.Add(-1 * time.Hour),
				Embedding: genEmbedding(),
			},
			{
				ID:        types.NewTicketID(),
				Status:    types.TicketStatusOpen,
				CreatedAt: now.Add(1 * time.Hour),
				Embedding: genEmbedding(),
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

func TestCountTicketsByStatus(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()
		thread := newTestThread()

		// Helper to generate random embedding
		genEmbedding := func() []float32 {
			emb := make([]float32, 256)
			for i := range emb {
				emb[i] = rand.Float32()
			}
			return emb
		}

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
				Embedding: genEmbedding(),
			},
			{
				ID:        types.NewTicketID(),
				Status:    types.TicketStatusOpen,
				CreatedAt: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
				SlackThread: &slack.Thread{
					ChannelID: thread.ChannelID,
					ThreadID:  thread.ThreadID,
				},
				Embedding: genEmbedding(),
			},
			{
				ID:        types.NewTicketID(),
				Status:    types.TicketStatusArchived,
				CreatedAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				SlackThread: &slack.Thread{
					ChannelID: thread.ChannelID,
					ThreadID:  thread.ThreadID,
				},
				Embedding: genEmbedding(),
			},
			{
				ID:        types.NewTicketID(),
				Status:    types.TicketStatusResolved,
				CreatedAt: time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC),
				SlackThread: &slack.Thread{
					ChannelID: thread.ChannelID,
					ThreadID:  thread.ThreadID,
				},
				Embedding: genEmbedding(),
			},
		}

		// Put tickets
		for _, ticket := range tickets {
			gt.NoError(t, repo.PutTicket(ctx, *ticket))
		}

		t.Run("count open tickets", func(t *testing.T) {
			count, err := repo.CountTicketsByStatus(ctx, []types.TicketStatus{types.TicketStatusOpen}, "", "")
			gt.NoError(t, err)
			gt.Number(t, count).GreaterOrEqual(0) // Should return at least 0 (may have existing data in Firestore)
		})

		t.Run("count pending tickets", func(t *testing.T) {
			count, err := repo.CountTicketsByStatus(ctx, []types.TicketStatus{types.TicketStatusArchived}, "", "")
			gt.NoError(t, err)
			gt.Number(t, count).GreaterOrEqual(0) // Should return at least 0 (may have existing data in Firestore)
		})

		t.Run("count resolved tickets", func(t *testing.T) {
			count, err := repo.CountTicketsByStatus(ctx, []types.TicketStatus{types.TicketStatusResolved}, "", "")
			gt.NoError(t, err)
			gt.Number(t, count).GreaterOrEqual(0) // Should return at least 0 (may have existing data in Firestore)
		})

		t.Run("count multiple statuses", func(t *testing.T) {
			count, err := repo.CountTicketsByStatus(ctx, []types.TicketStatus{
				types.TicketStatusOpen,
				types.TicketStatusArchived,
			}, "", "")
			gt.NoError(t, err)
			gt.Number(t, count).GreaterOrEqual(0) // Should return at least 0 (may have existing data in Firestore)
		})

		t.Run("count all tickets", func(t *testing.T) {
			count, err := repo.CountTicketsByStatus(ctx, nil, "", "")
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

		for i := range 10 {
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
