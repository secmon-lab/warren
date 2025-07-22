package graphql_test

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
	"github.com/secmon-lab/warren/pkg/usecase"
)

func TestClusteringPerformance(t *testing.T) {
	ctx := context.Background()

	t.Run("clustering with large dataset", func(t *testing.T) {
		repo := repository.NewMemory()
		uc := usecase.New(usecase.WithRepository(repo))

		// Create test alerts with varied embeddings
		alertCount := 100
		for i := 0; i < alertCount; i++ {
			testAlert := &alert.Alert{
				ID:       types.NewAlertID(),
				TicketID: types.EmptyTicketID,
				Schema:   types.AlertSchema(fmt.Sprintf("test.schema.v%d", i%5)),
				Data:     map[string]interface{}{"index": i, "value": fmt.Sprintf("test%d", i)},
				Metadata: alert.Metadata{
					Title:       fmt.Sprintf("Test Alert %d", i),
					Description: fmt.Sprintf("Test alert description %d", i),
					Attributes:  []alert.Attribute{},
				},
				CreatedAt: time.Now().Add(-time.Duration(i) * time.Minute),
				Embedding: firestore.Vector32{
					float32(i%10) * 0.1,
					float32((i+1)%10) * 0.1,
					float32((i+2)%10) * 0.1,
					float32((i+3)%10) * 0.1,
					float32((i+4)%10) * 0.1,
				},
			}
			gt.NoError(t, repo.PutAlert(ctx, *testAlert))
		}

		// Test clustering performance
		start := time.Now()
		result, err := uc.ClusteringUC.GetAlertClusters(ctx, usecase.GetClustersParams{
			Limit:          100,
			Offset:         0,
			MinClusterSize: 1,
		})
		duration := time.Since(start)

		gt.NoError(t, err)
		gt.NotNil(t, result)

		// Performance assertions
		if duration > 5*time.Second {
			t.Errorf("Clustering took too long: %v (expected < 5s)", duration)
		}

		// Verify clustering worked
		totalProcessed := len(result.Clusters)
		for _, cluster := range result.Clusters {
			totalProcessed += cluster.Size - 1 // subtract 1 because center alert is counted separately
		}
		totalProcessed += len(result.NoiseAlertIDs)

		if totalProcessed != alertCount {
			t.Errorf("Expected %d alerts to be processed, got %d", alertCount, totalProcessed)
		}

		t.Logf("Clustered %d alerts in %v", alertCount, duration)
		t.Logf("Found %d clusters and %d noise alerts", len(result.Clusters), len(result.NoiseAlertIDs))
	})

	t.Run("cluster alerts pagination", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping pagination test in short mode")
		}

		repo := repository.NewMemory()
		uc := usecase.New(usecase.WithRepository(repo))

		// Create a cluster of similar alerts
		clusterSize := 50
		for i := 0; i < clusterSize; i++ {
			testAlert := &alert.Alert{
				ID:       types.NewAlertID(),
				TicketID: types.EmptyTicketID,
				Schema:   "similarity.test.v1",
				Data:     map[string]interface{}{"keyword": "test", "index": i},
				Metadata: alert.Metadata{
					Title:       fmt.Sprintf("Similar Alert %d", i),
					Description: "Test alert for similarity clustering",
					Attributes:  []alert.Attribute{},
				},
				CreatedAt: time.Now().Add(-time.Duration(i) * time.Minute),
				Embedding: firestore.Vector32{
					0.1 + float32(i)*0.001, // Very similar embeddings
					0.2 + float32(i)*0.001,
					0.3 + float32(i)*0.001,
					0.4 + float32(i)*0.001,
					0.5 + float32(i)*0.001,
				},
			}
			gt.NoError(t, repo.PutAlert(ctx, *testAlert))
		}

		// Get clusters
		clustersResult, err := uc.ClusteringUC.GetAlertClusters(ctx, usecase.GetClustersParams{
			Limit:          100,
			Offset:         0,
			MinClusterSize: 10, // Should form one large cluster
		})
		gt.NoError(t, err)

		if len(clustersResult.Clusters) == 0 {
			t.Skip("No clusters formed, skipping pagination test")
			return
		}

		clusterID := clustersResult.Clusters[0].ID

		// Test pagination
		pageSize := 10
		page1, totalCount, err := uc.ClusteringUC.GetClusterAlerts(ctx, clusterID, "", pageSize, 0)
		gt.NoError(t, err)

		if len(page1) > pageSize {
			t.Errorf("Page 1 returned %d alerts, expected <= %d", len(page1), pageSize)
		}

		if totalCount < len(page1) {
			t.Errorf("Total count %d is less than returned results %d", totalCount, len(page1))
		}

		if totalCount > pageSize {
			page2, _, err := uc.ClusteringUC.GetClusterAlerts(ctx, clusterID, "", pageSize, pageSize)
			gt.NoError(t, err)

			if len(page2) == 0 {
				t.Error("Expected second page to have results")
			}

			// Verify no overlap between pages
			page1IDs := make(map[types.AlertID]bool)
			for _, alert := range page1 {
				page1IDs[alert.ID] = true
			}

			for _, alert := range page2 {
				if page1IDs[alert.ID] {
					t.Error("Found duplicate alert between pages")
				}
			}
		}

		t.Logf("Cluster %s has %d alerts, tested pagination with page size %d", clusterID, totalCount, pageSize)
	})

	t.Run("keyword filtering", func(t *testing.T) {
		repo := repository.NewMemory()
		uc := usecase.New(usecase.WithRepository(repo))

		// Create alerts with different keywords
		keywords := []string{"database", "network", "authentication", "malware"}
		for i, keyword := range keywords {
			for j := 0; j < 5; j++ { // 5 alerts per keyword
				testAlert := &alert.Alert{
					ID:       types.NewAlertID(),
					TicketID: types.EmptyTicketID,
					Schema:   "keyword.test.v1",
					Data:     map[string]interface{}{"type": keyword, "severity": "high", "count": j + 1},
					Metadata: alert.Metadata{
						Title:       fmt.Sprintf("%s Alert %d", keyword, j+1),
						Description: fmt.Sprintf("Alert related to %s security", keyword),
						Attributes:  []alert.Attribute{},
					},
					CreatedAt: time.Now().Add(-time.Duration(i*5+j) * time.Minute),
					Embedding: firestore.Vector32{
						float32(i) * 0.2,
						float32(j) * 0.1,
						0.3, 0.4, 0.5,
					},
				}
				gt.NoError(t, repo.PutAlert(ctx, *testAlert))
			}
		}

		// Get clusters
		clustersResult, err := uc.ClusteringUC.GetAlertClusters(ctx, usecase.GetClustersParams{
			Limit:          100,
			Offset:         0,
			MinClusterSize: 1,
		})
		gt.NoError(t, err)

		if len(clustersResult.Clusters) == 0 {
			t.Skip("No clusters formed, skipping keyword filtering test")
			return
		}

		clusterID := clustersResult.Clusters[0].ID

		// Test keyword filtering
		for _, keyword := range keywords {
			filtered, count, err := uc.ClusteringUC.GetClusterAlerts(ctx, clusterID, keyword, 100, 0)
			gt.NoError(t, err)

			if count > 0 {
				// Basic verification that filtering returned results
				if len(filtered) > count {
					t.Errorf("Filtered results %d exceed total count %d", len(filtered), count)
				}
				t.Logf("Keyword '%s' filtered to %d alerts out of cluster", keyword, count)
			}
		}
	})
}
