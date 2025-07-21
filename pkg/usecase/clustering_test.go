package usecase_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service/clustering"
	"github.com/secmon-lab/warren/pkg/usecase"
)

func TestClusteringUseCase_GetAlertClusters(t *testing.T) {
	ctx := context.Background()

	t.Run("basic clustering", func(t *testing.T) {
		repo := repository.NewMemory()
		uc := usecase.NewClusteringUseCase(repo)

		// Create test alerts with embeddings
		alerts := []alert.Alert{
			{
				ID:        types.AlertID("alert1"),
				Embedding: firestore.Vector32{1.0, 0.0, 0.0},
				TicketID:  types.EmptyTicketID, // unbound
			},
			{
				ID:        types.AlertID("alert2"),
				Embedding: firestore.Vector32{0.99, 0.01, 0.0}, // very similar
				TicketID:  types.EmptyTicketID,                 // unbound
			},
			{
				ID:        types.AlertID("alert3"),
				Embedding: firestore.Vector32{0.0, 1.0, 0.0},
				TicketID:  types.EmptyTicketID, // unbound
			},
		}

		// Save alerts
		for _, a := range alerts {
			alertCopy := a
			gt.NoError(t, repo.PutAlert(ctx, alertCopy))
		}

		// Get clusters
		params := usecase.GetClustersParams{
			MinClusterSize: 1,
			DBSCANParams: clustering.DBSCANParams{
				Eps:        0.15,
				MinSamples: 2,
			},
		}

		summary, err := uc.GetAlertClusters(ctx, params)
		gt.NoError(t, err)
		gt.NotEqual(t, summary, nil)
		gt.True(t, len(summary.Clusters) > 0)
	})

	t.Run("excludes bound alerts", func(t *testing.T) {
		repo := repository.NewMemory()
		uc := usecase.NewClusteringUseCase(repo)

		ticketID := types.TicketID("ticket1")
		alerts := []alert.Alert{
			{
				ID:        types.AlertID("alert1"),
				Embedding: firestore.Vector32{1.0, 0.0, 0.0},
				TicketID:  types.EmptyTicketID, // unbound
			},
			{
				ID:        types.AlertID("alert2"),
				Embedding: firestore.Vector32{0.99, 0.01, 0.0}, // very similar
				TicketID:  ticketID,                            // bound
			},
		}

		// Save alerts
		for _, a := range alerts {
			alertCopy := a
			gt.NoError(t, repo.PutAlert(ctx, alertCopy))
		}

		// Get clusters
		params := usecase.GetClustersParams{
			MinClusterSize: 1,
			DBSCANParams: clustering.DBSCANParams{
				Eps:        0.15,
				MinSamples: 1,
			},
		}

		summary, err := uc.GetAlertClusters(ctx, params)
		gt.NoError(t, err)
		// Only unbound alert should be processed
		totalAlerts := 0
		for _, cluster := range summary.Clusters {
			totalAlerts += cluster.Size
		}
		totalAlerts += len(summary.NoiseAlertIDs)
		gt.Equal(t, totalAlerts, 1)
	})

	t.Run("minimum cluster size filter", func(t *testing.T) {
		repo := repository.NewMemory()
		uc := usecase.NewClusteringUseCase(repo)

		// Create alerts that will form different sized clusters
		alerts := []alert.Alert{
			// Small cluster (2 alerts)
			{ID: types.AlertID("alert1"), Embedding: firestore.Vector32{1.0, 0.0, 0.0}, TicketID: types.EmptyTicketID},
			{ID: types.AlertID("alert2"), Embedding: firestore.Vector32{0.95, 0.05, 0.0}, TicketID: types.EmptyTicketID},
			// Large cluster (3 alerts)
			{ID: types.AlertID("alert3"), Embedding: firestore.Vector32{0.0, 1.0, 0.0}, TicketID: types.EmptyTicketID},
			{ID: types.AlertID("alert4"), Embedding: firestore.Vector32{0.05, 0.95, 0.0}, TicketID: types.EmptyTicketID},
			{ID: types.AlertID("alert5"), Embedding: firestore.Vector32{0.0, 0.9, 0.1}, TicketID: types.EmptyTicketID},
		}

		for _, a := range alerts {
			alertCopy := a
			gt.NoError(t, repo.PutAlert(ctx, alertCopy))
		}

		// Request only clusters with 3 or more alerts
		params := usecase.GetClustersParams{
			MinClusterSize: 3,
			DBSCANParams: clustering.DBSCANParams{
				Eps:        0.15,
				MinSamples: 2,
			},
		}

		summary, err := uc.GetAlertClusters(ctx, params)
		gt.NoError(t, err)
		// Should only return the large cluster
		gt.Equal(t, len(summary.Clusters), 1)
		gt.True(t, summary.Clusters[0].Size >= 3)
	})

	t.Run("caching", func(t *testing.T) {
		repo := repository.NewMemory()
		uc := usecase.NewClusteringUseCase(repo)

		// Create test alerts
		alerts := []alert.Alert{
			{ID: types.AlertID("alert1"), Embedding: firestore.Vector32{1.0, 0.0, 0.0}, TicketID: types.EmptyTicketID},
			{ID: types.AlertID("alert2"), Embedding: firestore.Vector32{0.9, 0.1, 0.0}, TicketID: types.EmptyTicketID},
		}

		for _, a := range alerts {
			alertCopy := a
			gt.NoError(t, repo.PutAlert(ctx, alertCopy))
		}

		params := usecase.GetClustersParams{
			DBSCANParams: clustering.DBSCANParams{
				Eps:        0.15,
				MinSamples: 2,
			},
		}

		// First call - computes clustering
		summary1, err := uc.GetAlertClusters(ctx, params)
		gt.NoError(t, err)
		computedAt1 := summary1.ComputedAt

		// Small delay
		time.Sleep(10 * time.Millisecond)

		// Second call - should return cached result
		summary2, err := uc.GetAlertClusters(ctx, params)
		gt.NoError(t, err)
		gt.Equal(t, summary2.ComputedAt, computedAt1) // Same timestamp means cached
	})
}

func TestClusteringUseCase_GetClusterAlerts(t *testing.T) {
	ctx := context.Background()

	t.Run("get alerts with keyword filter", func(t *testing.T) {
		repo := repository.NewMemory()
		uc := usecase.NewClusteringUseCase(repo)

		// Create test alerts
		alerts := []alert.Alert{
			{
				ID:        types.AlertID("alert1"),
				Data:      map[string]interface{}{"message": "error in system"},
				Embedding: firestore.Vector32{1.0, 0.0, 0.0},
				TicketID:  types.EmptyTicketID,
			},
			{
				ID:        types.AlertID("alert2"),
				Data:      map[string]interface{}{"message": "warning in database"},
				Embedding: firestore.Vector32{0.99, 0.01, 0.0}, // very similar
				TicketID:  types.EmptyTicketID,
			},
			{
				ID:        types.AlertID("alert3"),
				Data:      map[string]interface{}{"message": "info log"},
				Embedding: firestore.Vector32{0.95, 0.05, 0.0},
				TicketID:  types.EmptyTicketID,
			},
		}

		for _, a := range alerts {
			alertCopy := a
			gt.NoError(t, repo.PutAlert(ctx, alertCopy))
		}

		// Get clusters first to populate cache
		params := usecase.GetClustersParams{
			DBSCANParams: clustering.DBSCANParams{
				Eps:        0.15,
				MinSamples: 2,
			},
		}
		summary, err := uc.GetAlertClusters(ctx, params)
		gt.NoError(t, err)
		gt.True(t, len(summary.Clusters) > 0)

		clusterID := summary.Clusters[0].ID

		// Get cluster alerts with keyword filter
		filteredAlerts, total, err := uc.GetClusterAlerts(ctx, clusterID, "error", 10, 0)
		gt.NoError(t, err)
		gt.Equal(t, len(filteredAlerts), 1)
		gt.Equal(t, total, 1)
		gt.Equal(t, filteredAlerts[0].ID, types.AlertID("alert1"))
	})

	t.Run("pagination", func(t *testing.T) {
		repo := repository.NewMemory()
		uc := usecase.NewClusteringUseCase(repo)

		// Create many alerts
		var alerts []alert.Alert
		for i := 0; i < 5; i++ {
			alerts = append(alerts, alert.Alert{
				ID:        types.AlertID(fmt.Sprintf("alert-%d", i)),
				Data:      map[string]interface{}{"index": i},
				Embedding: firestore.Vector32{1.0, float32(i) * 0.01, 0.0}, // Similar embeddings
				TicketID:  types.EmptyTicketID,
			})
		}

		for _, a := range alerts {
			alertCopy := a
			gt.NoError(t, repo.PutAlert(ctx, alertCopy))
		}

		// Get clusters
		params := usecase.GetClustersParams{
			DBSCANParams: clustering.DBSCANParams{
				Eps:        0.15,
				MinSamples: 2,
			},
		}
		summary, err := uc.GetAlertClusters(ctx, params)
		gt.NoError(t, err)
		gt.True(t, len(summary.Clusters) > 0)

		clusterID := summary.Clusters[0].ID

		// Test pagination
		page1, total1, err := uc.GetClusterAlerts(ctx, clusterID, "", 2, 0)
		gt.NoError(t, err)
		gt.Equal(t, len(page1), 2)
		gt.Equal(t, total1, 5)

		page2, total2, err := uc.GetClusterAlerts(ctx, clusterID, "", 2, 2)
		gt.NoError(t, err)
		gt.Equal(t, len(page2), 2)
		gt.Equal(t, total2, 5)

		// Ensure different alerts
		gt.NotEqual(t, page1[0].ID, page2[0].ID)
	})
}
