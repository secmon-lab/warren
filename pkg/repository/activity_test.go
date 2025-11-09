package repository_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/activity"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	ticketmodel "github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/utils/user"
)

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

				// Generate random embedding
				emb := make([]float32, 256)
				for i := range emb {
					emb[i] = rand.Float32()
				}

				ticket := ticketmodel.Ticket{
					ID: types.NewTicketID(),
					Metadata: ticketmodel.Metadata{
						Title: "Test Ticket",
					},
					Status:    types.TicketStatusOpen,
					CreatedAt: time.Now(),
					Embedding: emb,
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

				// Generate random embedding
				emb := make([]float32, 256)
				for i := range emb {
					emb[i] = rand.Float32()
				}

				// First create a ticket
				ticket := ticketmodel.Ticket{
					ID: types.NewTicketID(),
					Metadata: ticketmodel.Metadata{
						Title: "Original Title",
					},
					Status:    types.TicketStatusOpen,
					CreatedAt: time.Now(),
					Embedding: emb,
				}

				err := repo.PutTicket(ctx, ticket)
				gt.NoError(t, err).Required()

				// Wait a bit for Firestore eventual consistency
				time.Sleep(100 * time.Millisecond)

				// Now update the ticket
				ticket.Title = "Updated Title"
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
						switch act.Type {
						case types.ActivityTypeTicketCreated:
							creationActivity = act
						case types.ActivityTypeTicketUpdated:
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

				// Generate random embedding
				emb := make([]float32, 256)
				for i := range emb {
					emb[i] = rand.Float32()
				}

				ticket := ticketmodel.Ticket{
					ID: types.NewTicketID(),
					Metadata: ticketmodel.Metadata{
						Title: "Test Ticket for Comments",
					},
					Status:    types.TicketStatusOpen,
					CreatedAt: time.Now(),
					Embedding: emb,
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

				// Generate random embedding
				emb := make([]float32, 256)
				for i := range emb {
					emb[i] = rand.Float32()
				}

				ticket := ticketmodel.Ticket{
					ID: types.NewTicketID(),
					Metadata: ticketmodel.Metadata{
						Title: "Test Ticket for Agent Comments",
					},
					Status:    types.TicketStatusOpen,
					CreatedAt: time.Now(),
					Embedding: emb,
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

				// Generate random embedding
				emb := make([]float32, 256)
				for i := range emb {
					emb[i] = rand.Float32()
				}

				ticket := ticketmodel.Ticket{
					ID: types.NewTicketID(),
					Metadata: ticketmodel.Metadata{
						Title: "Test Ticket for Alert Binding",
					},
					Status:    types.TicketStatusOpen,
					CreatedAt: time.Now(),
					Embedding: emb,
				}

				// Generate random embedding for alert
				alertEmb := make([]float32, 256)
				for i := range alertEmb {
					alertEmb[i] = rand.Float32()
				}

				alert := &alert.Alert{
					ID: types.NewAlertID(),
					Metadata: alert.Metadata{
						Title: "Test Alert",
					},
					CreatedAt: time.Now(),
					Embedding: alertEmb,
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

				// Generate random embedding
				emb := make([]float32, 256)
				for i := range emb {
					emb[i] = rand.Float32()
				}

				ticket := ticketmodel.Ticket{
					ID: types.NewTicketID(),
					Metadata: ticketmodel.Metadata{
						Title: "Test Ticket for Bulk Alert Binding",
					},
					Status:    types.TicketStatusOpen,
					CreatedAt: time.Now(),
					Embedding: emb,
				}

				// Create multiple alerts
				alertIDs := make([]types.AlertID, 3)
				for i := range 3 {
					// Generate random embedding for each alert
					alertEmb := make([]float32, 256)
					for j := range alertEmb {
						alertEmb[j] = rand.Float32()
					}

					alert := &alert.Alert{
						ID: types.NewAlertID(),
						Metadata: alert.Metadata{
							Title: fmt.Sprintf("Test Alert %d", i+1),
						},
						CreatedAt: time.Now(),
						Embedding: alertEmb,
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

				// Generate random embedding
				emb := make([]float32, 256)
				for i := range emb {
					emb[i] = rand.Float32()
				}

				ticket := ticketmodel.Ticket{
					ID: types.NewTicketID(),
					Metadata: ticketmodel.Metadata{
						Title: "Test Ticket for Status Change",
					},
					Status:    types.TicketStatusOpen,
					CreatedAt: time.Now(),
					Embedding: emb,
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
