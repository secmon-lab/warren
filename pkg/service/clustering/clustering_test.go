package clustering_test

import (
	"context"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/clustering"
)

func TestDBSCANClustering(t *testing.T) {
	ctx := context.Background()
	service := clustering.NewService()

	t.Run("empty alerts", func(t *testing.T) {
		result, err := service.ClusterAlerts(ctx, []*alert.Alert{}, clustering.DBSCANParams{
			Eps:        0.3,
			MinSamples: 2,
		})
		gt.NoError(t, err)
		gt.Equal(t, len(result.Clusters), 0)
		gt.Equal(t, len(result.NoiseAlertIDs), 0)
	})

	t.Run("alerts without embeddings", func(t *testing.T) {
		alerts := []*alert.Alert{
			{ID: types.AlertID("alert1")},
			{ID: types.AlertID("alert2")},
		}
		result, err := service.ClusterAlerts(ctx, alerts, clustering.DBSCANParams{
			Eps:        0.3,
			MinSamples: 2,
		})
		gt.NoError(t, err)
		gt.Equal(t, len(result.Clusters), 0)
		gt.Equal(t, len(result.NoiseAlertIDs), 0)
	})

	t.Run("single cluster", func(t *testing.T) {
		// Create alerts with similar embeddings
		alerts := []*alert.Alert{
			{ID: types.AlertID("alert1"), Embedding: firestore.Vector32{1.0, 0.0, 0.0}},
			{ID: types.AlertID("alert2"), Embedding: firestore.Vector32{0.9, 0.1, 0.0}},
			{ID: types.AlertID("alert3"), Embedding: firestore.Vector32{0.95, 0.05, 0.0}},
		}

		result, err := service.ClusterAlerts(ctx, alerts, clustering.DBSCANParams{
			Eps:        0.2, // Small distance threshold
			MinSamples: 2,
		})
		gt.NoError(t, err)
		gt.Equal(t, len(result.Clusters), 1)
		gt.Equal(t, result.Clusters[0].Size, 3)
		gt.Equal(t, len(result.NoiseAlertIDs), 0)
	})

	t.Run("multiple clusters", func(t *testing.T) {
		// Create alerts with two distinct groups
		alerts := []*alert.Alert{
			// Cluster 1 - very similar vectors
			{ID: types.AlertID("alert1"), Embedding: firestore.Vector32{1.0, 0.0, 0.0}},
			{ID: types.AlertID("alert2"), Embedding: firestore.Vector32{0.99, 0.01, 0.0}},
			// Cluster 2 - very similar vectors
			{ID: types.AlertID("alert3"), Embedding: firestore.Vector32{0.0, 1.0, 0.0}},
			{ID: types.AlertID("alert4"), Embedding: firestore.Vector32{0.01, 0.99, 0.0}},
		}

		result, err := service.ClusterAlerts(ctx, alerts, clustering.DBSCANParams{
			Eps:        0.15,
			MinSamples: 2,
		})
		gt.NoError(t, err)
		gt.Equal(t, len(result.Clusters), 2)
		gt.Equal(t, len(result.NoiseAlertIDs), 0)
	})

	t.Run("noise points", func(t *testing.T) {
		// Create alerts with outliers
		alerts := []*alert.Alert{
			// Cluster - very similar vectors
			{ID: types.AlertID("alert1"), Embedding: firestore.Vector32{1.0, 0.0, 0.0}},
			{ID: types.AlertID("alert2"), Embedding: firestore.Vector32{0.99, 0.01, 0.0}},
			// Noise points (isolated)
			{ID: types.AlertID("alert3"), Embedding: firestore.Vector32{0.0, 0.0, 1.0}},
			{ID: types.AlertID("alert4"), Embedding: firestore.Vector32{-1.0, 0.0, 0.0}},
		}

		result, err := service.ClusterAlerts(ctx, alerts, clustering.DBSCANParams{
			Eps:        0.15,
			MinSamples: 2,
		})
		gt.NoError(t, err)
		gt.True(t, len(result.Clusters) >= 1)
		if len(result.Clusters) > 0 {
			gt.Equal(t, result.Clusters[0].Size, 2)
		}
		gt.Equal(t, len(result.NoiseAlertIDs), 2)
		// Check that noise alerts contain the expected IDs
		hasAlert3 := false
		hasAlert4 := false
		for _, id := range result.NoiseAlertIDs {
			if id == "alert3" {
				hasAlert3 = true
			}
			if id == "alert4" {
				hasAlert4 = true
			}
		}
		gt.True(t, hasAlert3)
		gt.True(t, hasAlert4)
	})

	t.Run("clusters sorted by size", func(t *testing.T) {
		// Create alerts with different cluster sizes
		alerts := []*alert.Alert{
			// Small cluster - very similar vectors
			{ID: types.AlertID("alert1"), Embedding: firestore.Vector32{1.0, 0.0, 0.0}},
			{ID: types.AlertID("alert2"), Embedding: firestore.Vector32{0.99, 0.01, 0.0}},
			// Large cluster - very similar vectors
			{ID: types.AlertID("alert3"), Embedding: firestore.Vector32{0.0, 1.0, 0.0}},
			{ID: types.AlertID("alert4"), Embedding: firestore.Vector32{0.01, 0.99, 0.0}},
			{ID: types.AlertID("alert5"), Embedding: firestore.Vector32{0.0, 0.99, 0.01}},
			{ID: types.AlertID("alert6"), Embedding: firestore.Vector32{0.01, 0.98, 0.01}},
		}

		result, err := service.ClusterAlerts(ctx, alerts, clustering.DBSCANParams{
			Eps:        0.15,
			MinSamples: 2,
		})
		gt.NoError(t, err)
		gt.Equal(t, len(result.Clusters), 2)
		// Check that clusters are sorted by size (descending)
		gt.True(t, result.Clusters[0].Size > result.Clusters[1].Size)
	})
}

func TestFindCenterAlert(t *testing.T) {
	ctx := context.Background()
	service := clustering.NewService()

	t.Run("single alert", func(t *testing.T) {
		alerts := []*alert.Alert{
			{ID: types.AlertID("alert1"), Embedding: firestore.Vector32{1.0, 0.0, 0.0}},
		}
		center, err := service.FindCenterAlert(ctx, alerts)
		gt.NoError(t, err)
		gt.Equal(t, center.ID, types.AlertID("alert1"))
	})

	t.Run("multiple alerts", func(t *testing.T) {
		// alert2 should be closest to the centroid
		alerts := []*alert.Alert{
			{ID: types.AlertID("alert1"), Embedding: firestore.Vector32{1.0, 0.0, 0.0}},
			{ID: types.AlertID("alert2"), Embedding: firestore.Vector32{0.5, 0.5, 0.0}}, // This should be the center
			{ID: types.AlertID("alert3"), Embedding: firestore.Vector32{0.0, 1.0, 0.0}},
		}
		center, err := service.FindCenterAlert(ctx, alerts)
		gt.NoError(t, err)
		gt.Equal(t, center.ID, types.AlertID("alert2"))
	})

	t.Run("empty alerts", func(t *testing.T) {
		_, err := service.FindCenterAlert(ctx, []*alert.Alert{})
		gt.Error(t, err)
	})
}

func TestCosineDistance(t *testing.T) {
	// Test the cosine distance calculation indirectly through clustering
	ctx := context.Background()
	service := clustering.NewService()

	t.Run("identical vectors should cluster together", func(t *testing.T) {
		alerts := []*alert.Alert{
			{ID: types.AlertID("alert1"), Embedding: firestore.Vector32{1.0, 0.0, 0.0}},
			{ID: types.AlertID("alert2"), Embedding: firestore.Vector32{1.0, 0.0, 0.0}},
		}

		result, err := service.ClusterAlerts(ctx, alerts, clustering.DBSCANParams{
			Eps:        0.1, // Very small threshold
			MinSamples: 2,
		})
		gt.NoError(t, err)
		gt.Equal(t, len(result.Clusters), 1)
		gt.Equal(t, result.Clusters[0].Size, 2)
	})

	t.Run("orthogonal vectors should not cluster", func(t *testing.T) {
		alerts := []*alert.Alert{
			{ID: types.AlertID("alert1"), Embedding: firestore.Vector32{1.0, 0.0, 0.0}},
			{ID: types.AlertID("alert2"), Embedding: firestore.Vector32{0.0, 1.0, 0.0}},
		}

		result, err := service.ClusterAlerts(ctx, alerts, clustering.DBSCANParams{
			Eps:        0.15, // Cosine distance between orthogonal vectors is 1.0
			MinSamples: 2,
		})
		gt.NoError(t, err)
		gt.Equal(t, len(result.Clusters), 0)
		gt.Equal(t, len(result.NoiseAlertIDs), 2)
	})
}
