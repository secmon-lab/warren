package repository_test

import (
	"fmt"
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	ticketmodel "github.com/secmon-lab/warren/pkg/domain/model/ticket"
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
	// Generate random embedding to avoid zero vector issues
	embedding := make([]float32, 256)
	for i := range embedding {
		embedding[i] = rand.Float32()
	}

	return ticketmodel.Ticket{
		ID: types.NewTicketID(),
		Metadata: ticketmodel.Metadata{
			Title:       "Test Ticket",
			Description: "Test Description",
		},
		SlackThread: thread,
		Embedding:   embedding,
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
