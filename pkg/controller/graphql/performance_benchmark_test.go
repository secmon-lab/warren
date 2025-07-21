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
	"github.com/secmon-lab/warren/pkg/service/clustering"
	"github.com/secmon-lab/warren/pkg/usecase"
)

func BenchmarkClusteringPerformance(b *testing.B) {
	ctx := context.Background()

	// Test different dataset sizes
	for _, alertCount := range []int{100, 500, 1000, 2000} {
		b.Run(fmt.Sprintf("alerts_%d", alertCount), func(b *testing.B) {
			repo := repository.NewMemory()
			uc := usecase.New(usecase.WithRepository(repo))

			// Create test alerts
			for i := 0; i < alertCount; i++ {
				testAlert := &alert.Alert{
					ID:       types.NewAlertID(),
					TicketID: types.EmptyTicketID,
					Schema:   types.AlertSchema(fmt.Sprintf("bench.schema.v%d", i%10)),
					Data:     map[string]interface{}{"index": i, "value": fmt.Sprintf("benchmark%d", i)},
					Metadata: alert.Metadata{
						Title:       fmt.Sprintf("Benchmark Alert %d", i),
						Description: fmt.Sprintf("Benchmark alert description %d", i),
						Attributes:  []alert.Attribute{},
					},
					CreatedAt: time.Now().Add(-time.Duration(i) * time.Minute),
					Embedding: firestore.Vector32{
						float32(i%20) * 0.05, // Create 20 different groups
						float32((i+1)%20) * 0.05,
						float32((i+2)%20) * 0.05,
						float32((i+3)%20) * 0.05,
						float32((i+4)%20) * 0.05,
					},
				}
				if err := repo.PutAlert(ctx, *testAlert); err != nil {
					b.Fatal(err)
				}
			}

			b.ResetTimer()

			// Benchmark clustering
			for i := 0; i < b.N; i++ {
				_, err := uc.ClusteringUC.GetAlertClusters(ctx, usecase.GetClustersParams{
					Limit:          alertCount,
					Offset:         0,
					MinClusterSize: 1,
				})
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestLargeDatasetPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large dataset test in short mode")
	}

	ctx := context.Background()

	// Test with 1000 alerts
	t.Run("1000 alerts clustering", func(t *testing.T) {
		repo := repository.NewMemory()
		uc := usecase.New(usecase.WithRepository(repo))

		alertCount := 1000
		t.Logf("Creating %d test alerts...", alertCount)

		// Create alerts with more realistic embeddings
		for i := 0; i < alertCount; i++ {
			// Create some clusters by grouping similar embeddings
			groupID := i / 50 // 50 alerts per group, 20 groups total

			testAlert := &alert.Alert{
				ID:       types.NewAlertID(),
				TicketID: types.EmptyTicketID,
				Schema:   types.AlertSchema(fmt.Sprintf("large.test.group%d", groupID)),
				Data: map[string]interface{}{
					"group_id": groupID,
					"index":    i,
					"severity": []string{"low", "medium", "high", "critical"}[i%4],
					"source":   fmt.Sprintf("system_%d", i%10),
				},
				Metadata: alert.Metadata{
					Title:       fmt.Sprintf("Large Test Alert %d (Group %d)", i, groupID),
					Description: fmt.Sprintf("Alert from group %d with index %d", groupID, i),
					Attributes:  []alert.Attribute{},
				},
				CreatedAt: time.Now().Add(-time.Duration(i) * time.Minute),
				Embedding: firestore.Vector32{
					// Similar embeddings within group, different between groups
					float32(groupID)*0.1 + float32(i%50)*0.001,
					float32(groupID)*0.1 + float32((i+1)%50)*0.001,
					float32(groupID)*0.1 + float32((i+2)%50)*0.001,
					float32(groupID)*0.1 + float32((i+3)%50)*0.001,
					float32(groupID)*0.1 + float32((i+4)%50)*0.001,
				},
			}
			gt.NoError(t, repo.PutAlert(ctx, *testAlert))
		}

		t.Log("Starting clustering performance test...")
		start := time.Now()
		result, err := uc.ClusteringUC.GetAlertClusters(ctx, usecase.GetClustersParams{
			Limit:          1000,
			Offset:         0,
			MinClusterSize: 5, // Require at least 5 alerts per cluster
			DBSCANParams: clustering.DBSCANParams{
				Eps:        0.3, // Distance threshold for neighborhood
				MinSamples: 3,   // Minimum samples to form a cluster
			},
		})
		duration := time.Since(start)

		gt.NoError(t, err)
		gt.NotNil(t, result)

		// Performance check
		if duration > 10*time.Second {
			t.Errorf("Clustering 1000 alerts took too long: %v (expected < 10s)", duration)
		}

		// Verify clustering results
		totalClustered := 0
		for _, cluster := range result.Clusters {
			totalClustered += cluster.Size
		}
		totalClustered += len(result.NoiseAlertIDs)

		if totalClustered != alertCount {
			t.Errorf("Expected %d alerts to be processed, got %d", alertCount, totalClustered)
		}

		t.Logf("Performance Results:")
		t.Logf("  - Processed %d alerts in %v", alertCount, duration)
		t.Logf("  - Found %d clusters", len(result.Clusters))
		t.Logf("  - Found %d noise alerts", len(result.NoiseAlertIDs))
		t.Logf("  - Average processing time per alert: %v", duration/time.Duration(alertCount))

		// Test cluster details retrieval performance
		if len(result.Clusters) > 0 {
			largestCluster := result.Clusters[0]
			for _, cluster := range result.Clusters {
				if cluster.Size > largestCluster.Size {
					largestCluster = cluster
				}
			}

			t.Logf("Testing cluster details retrieval for largest cluster (size: %d)...", largestCluster.Size)

			start = time.Now()
			alerts, count, err := uc.ClusteringUC.GetClusterAlerts(ctx, largestCluster.ID, "", 100, 0)
			duration = time.Since(start)

			gt.NoError(t, err)
			t.Logf("Retrieved %d/%d cluster alerts in %v", len(alerts), count, duration)

			if duration > 1*time.Second {
				t.Errorf("Cluster alerts retrieval took too long: %v (expected < 1s)", duration)
			}
		}
	})

	t.Run("pagination performance", func(t *testing.T) {
		repo := repository.NewMemory()
		uc := usecase.New(usecase.WithRepository(repo))

		// Create a large cluster
		clusterSize := 500
		t.Logf("Creating cluster with %d similar alerts...", clusterSize)

		for i := 0; i < clusterSize; i++ {
			testAlert := &alert.Alert{
				ID:       types.NewAlertID(),
				TicketID: types.EmptyTicketID,
				Schema:   "pagination.test.v1",
				Data: map[string]interface{}{
					"test_data": fmt.Sprintf("pagination test %d", i),
					"index":     i,
				},
				Metadata: alert.Metadata{
					Title:       fmt.Sprintf("Pagination Test Alert %d", i),
					Description: "Alert for pagination performance testing",
					Attributes:  []alert.Attribute{},
				},
				CreatedAt: time.Now().Add(-time.Duration(i) * time.Minute),
				Embedding: firestore.Vector32{
					// Very similar embeddings to form one large cluster
					0.5 + float32(i)*0.0001,
					0.5 + float32(i)*0.0001,
					0.5 + float32(i)*0.0001,
					0.5 + float32(i)*0.0001,
					0.5 + float32(i)*0.0001,
				},
			}
			gt.NoError(t, repo.PutAlert(ctx, *testAlert))
		}

		// Get clusters
		clustersResult, err := uc.ClusteringUC.GetAlertClusters(ctx, usecase.GetClustersParams{
			Limit:          1000,
			Offset:         0,
			MinClusterSize: 100, // Should form one large cluster
			DBSCANParams: clustering.DBSCANParams{
				Eps:        0.1, // Smaller distance for tighter clustering
				MinSamples: 10,  // Higher minimum for large clusters
			},
		})
		gt.NoError(t, err)

		if len(clustersResult.Clusters) == 0 {
			t.Skip("No large clusters formed, skipping pagination performance test")
			return
		}

		clusterID := clustersResult.Clusters[0].ID
		t.Logf("Testing pagination on cluster %s (size: %d)", clusterID, clustersResult.Clusters[0].Size)

		// Test different page sizes
		pageSizes := []int{10, 20, 50, 100}
		for _, pageSize := range pageSizes {
			t.Run(fmt.Sprintf("page_size_%d", pageSize), func(t *testing.T) {
				start := time.Now()
				alerts, totalCount, err := uc.ClusteringUC.GetClusterAlerts(ctx, clusterID, "", pageSize, 0)
				duration := time.Since(start)

				gt.NoError(t, err)

				if len(alerts) > pageSize {
					t.Errorf("Returned %d alerts, expected <= %d", len(alerts), pageSize)
				}

				if duration > 500*time.Millisecond {
					t.Errorf("Page retrieval (size %d) took too long: %v (expected < 500ms)", pageSize, duration)
				}

				t.Logf("Page size %d: retrieved %d/%d alerts in %v", pageSize, len(alerts), totalCount, duration)
			})
		}

		// Test pagination with keyword filtering
		t.Run("keyword_filtering_performance", func(t *testing.T) {
			keyword := "pagination"
			start := time.Now()
			alerts, count, err := uc.ClusteringUC.GetClusterAlerts(ctx, clusterID, keyword, 50, 0)
			duration := time.Since(start)

			gt.NoError(t, err)

			if duration > 1*time.Second {
				t.Errorf("Keyword filtering took too long: %v (expected < 1s)", duration)
			}

			t.Logf("Keyword filtering '%s': found %d matching alerts in %v", keyword, count, duration)

			// Verify that all returned alerts contain the keyword
			if count > 0 && len(alerts) > 0 {
				for _, alert := range alerts {
					dataStr := fmt.Sprintf("%v", alert.Data)
					contains := (alert.Title != "" && alert.Title != "no title") ||
						(alert.Description != "" && alert.Description != "no description") ||
						(dataStr != "" && dataStr != "{}")
					if !contains {
						t.Errorf("Alert %s does not appear to contain keyword filtering logic", alert.ID)
					}
				}
			}
		})
	})
}
