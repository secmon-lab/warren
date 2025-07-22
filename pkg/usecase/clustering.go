package usecase

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	goerr "github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/clustering"
)

// ClusteringUseCase provides clustering-related use cases
type ClusteringUseCase struct {
	repo              interfaces.Repository
	clusteringService clustering.Service
}

// NewClusteringUseCase creates a new clustering use case instance
func NewClusteringUseCase(repo interfaces.Repository) *ClusteringUseCase {
	return &ClusteringUseCase{
		repo:              repo,
		clusteringService: clustering.NewService(),
	}
}

// GetClustersParams represents parameters for getting clusters
type GetClustersParams struct {
	MinClusterSize int
	Limit          int
	Offset         int
	Keyword        string
	DBSCANParams   clustering.DBSCANParams
}

// ClusteringSummary represents the summary of clustering results
type ClusteringSummary struct {
	Clusters      []*clustering.AlertCluster
	NoiseAlertIDs []types.AlertID
	Parameters    clustering.DBSCANParams
	ComputedAt    time.Time
	TotalCount    int
}

// GetAlertClusters retrieves alert clusters
func (uc *ClusteringUseCase) GetAlertClusters(ctx context.Context, params GetClustersParams) (*ClusteringSummary, error) {
	// Set default DBSCAN parameters if not provided
	if params.DBSCANParams.Eps == 0 && params.DBSCANParams.MinSamples == 0 {
		params.DBSCANParams = clustering.DBSCANParams{
			Eps:        0.3, // Default epsilon
			MinSamples: 2,   // Default minimum samples
		}
	}

	// Get unbound alerts with embeddings
	unboundAlerts, err := uc.repo.GetAlertWithoutTicket(ctx, 0, 0)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get unbound alerts")
	}

	// Filter alerts with embeddings
	alertsWithEmbedding := make([]*alert.Alert, 0, len(unboundAlerts))
	for _, a := range unboundAlerts {
		if len(a.Embedding) > 0 {
			alertsWithEmbedding = append(alertsWithEmbedding, a)
		}
	}

	// Perform clustering
	result, err := uc.clusteringService.ClusterAlerts(ctx, alertsWithEmbedding, params.DBSCANParams)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to cluster alerts")
	}

	// Create summary
	summary := &ClusteringSummary{
		Clusters:      result.Clusters,
		NoiseAlertIDs: result.NoiseAlertIDs,
		Parameters:    result.Parameters,
		ComputedAt:    time.Now(),
	}

	// Filter and paginate
	return uc.filterAndPaginateClusters(ctx, summary, params)
}

// GetClusterAlerts retrieves alerts in a specific cluster with filtering
// Since we no longer cache clusters, this method now requires re-clustering to find the specific cluster
func (uc *ClusteringUseCase) GetClusterAlerts(ctx context.Context, clusterID string, keyword string, limit, offset int) ([]*alert.Alert, int, error) {
	return uc.GetClusterAlertsWithParams(ctx, clusterID, keyword, limit, offset, clustering.DBSCANParams{})
}

// GetClusterAlertsWithParams retrieves alerts in a specific cluster with filtering using specific DBSCAN parameters
func (uc *ClusteringUseCase) GetClusterAlertsWithParams(ctx context.Context, clusterID string, keyword string, limit, offset int, params clustering.DBSCANParams) ([]*alert.Alert, int, error) {
	// Since we don't cache clusters anymore, we need to re-cluster to find the target cluster
	// This is less efficient but ensures we get current data

	// Get all unbound alerts with embeddings
	unboundAlerts, err := uc.repo.GetAlertWithoutTicket(ctx, 0, 0)
	if err != nil {
		return nil, 0, goerr.Wrap(err, "failed to get unbound alerts")
	}

	// Filter alerts with embeddings
	alertsWithEmbedding := make([]*alert.Alert, 0, len(unboundAlerts))
	for _, a := range unboundAlerts {
		if len(a.Embedding) > 0 {
			alertsWithEmbedding = append(alertsWithEmbedding, a)
		}
	}

	// Use provided DBSCAN parameters, or defaults if not specified
	if params.Eps == 0 && params.MinSamples == 0 {
		// Use same default parameters as GetAlertClusters
		params = clustering.DBSCANParams{
			Eps:        0.3, // Default epsilon
			MinSamples: 2,   // Default minimum samples
		}
	}

	// Perform clustering
	result, err := uc.clusteringService.ClusterAlerts(ctx, alertsWithEmbedding, params)
	if err != nil {
		return nil, 0, goerr.Wrap(err, "failed to cluster alerts")
	}

	// Find the target cluster
	var targetCluster *clustering.AlertCluster
	for _, cluster := range result.Clusters {
		if cluster.ID == clusterID {
			targetCluster = cluster
			break
		}
	}

	if targetCluster == nil {
		return nil, 0, goerr.New("cluster not found", goerr.V("clusterID", clusterID))
	}

	// Batch get alerts
	alerts, err := uc.repo.BatchGetAlerts(ctx, targetCluster.AlertIDs)
	if err != nil {
		return nil, 0, goerr.Wrap(err, "failed to get cluster alerts")
	}

	// Sort alerts by ID to ensure consistent pagination
	sort.Slice(alerts, func(i, j int) bool {
		return string(alerts[i].ID) < string(alerts[j].ID)
	})

	// Filter by keyword if provided
	filteredAlerts := alerts
	if keyword != "" {
		filtered := make([]*alert.Alert, 0, len(alerts))
		for _, a := range alerts {
			// Search in Alert.Data (JSON)
			dataBytes, _ := json.Marshal(a.Data)
			dataStr := string(dataBytes)
			if strings.Contains(strings.ToLower(dataStr), strings.ToLower(keyword)) {
				filtered = append(filtered, a)
			}
		}
		filteredAlerts = filtered
	}

	totalCount := len(filteredAlerts)

	// Apply pagination
	start := offset
	if start >= len(filteredAlerts) {
		return []*alert.Alert{}, totalCount, nil
	}

	end := start + limit
	if end > len(filteredAlerts) {
		end = len(filteredAlerts)
	}

	return filteredAlerts[start:end], totalCount, nil
}

// Helper methods

func (uc *ClusteringUseCase) filterAndPaginateClusters(ctx context.Context, summary *ClusteringSummary, params GetClustersParams) (*ClusteringSummary, error) {
	// Filter by minimum cluster size and keyword
	filteredClusters := make([]*clustering.AlertCluster, 0, len(summary.Clusters))
	for _, cluster := range summary.Clusters {
		if cluster.Size < params.MinClusterSize {
			continue
		}

		// If keyword is provided, check if cluster matches
		if params.Keyword != "" {
			// Get center alert to check its data
			centerAlert, err := uc.repo.GetAlert(ctx, cluster.CenterAlertID)
			if err != nil {
				return nil, goerr.Wrap(err, "failed to get center alert for keyword search")
			}

			// Check if keyword exists in center alert's data
			dataBytes, _ := json.Marshal(centerAlert.Data)
			dataStr := string(dataBytes)
			if !strings.Contains(strings.ToLower(dataStr), strings.ToLower(params.Keyword)) {
				// Also check keywords if they exist
				keywordFound := false
				for _, kw := range cluster.Keywords {
					if strings.Contains(strings.ToLower(kw), strings.ToLower(params.Keyword)) {
						keywordFound = true
						break
					}
				}
				if !keywordFound {
					continue
				}
			}
		}

		filteredClusters = append(filteredClusters, cluster)
	}

	// Calculate total count after filtering
	totalCount := len(filteredClusters)

	// Apply pagination
	start := params.Offset
	if start >= len(filteredClusters) {
		return &ClusteringSummary{
			Clusters:      []*clustering.AlertCluster{},
			NoiseAlertIDs: summary.NoiseAlertIDs,
			Parameters:    summary.Parameters,
			ComputedAt:    summary.ComputedAt,
			TotalCount:    totalCount,
		}, nil
	}

	end := len(filteredClusters)
	if params.Limit > 0 {
		end = start + params.Limit
		if end > len(filteredClusters) {
			end = len(filteredClusters)
		}
	}

	return &ClusteringSummary{
		Clusters:      filteredClusters[start:end],
		NoiseAlertIDs: summary.NoiseAlertIDs,
		Parameters:    summary.Parameters,
		ComputedAt:    summary.ComputedAt,
		TotalCount:    totalCount,
	}, nil
}
