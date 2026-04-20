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

				// Now update the ticket
				ticket.Title = "Updated Title"
				ticket.UpdatedAt = time.Now()

				err = repo.PutTicket(ctx, ticket)
				gt.NoError(t, err).Required()

				// Wait a bit for Firestore eventual consistency

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

			// chat-session-redesign Phase 7 (confinement): the legacy
			// "comment addition" and "agent comment suppression" activity
			// cases were removed along with Repository.PutTicketComment.
			// session.Message(type=user) is persisted via the Session
			// Notifier path and does not emit activity rows (by design:
			// thread messages are always user-visible on Slack already).

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
