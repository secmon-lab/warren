package usecase

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/m-mizutani/goerr"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/clustering"
)

// ClusteringUseCase provides clustering-related use cases
type ClusteringUseCase struct {
	repo              interfaces.Repository
	clusteringService clustering.Service
	cache             *clusteringCache
}

// NewClusteringUseCase creates a new clustering use case instance
func NewClusteringUseCase(repo interfaces.Repository) *ClusteringUseCase {
	return &ClusteringUseCase{
		repo:              repo,
		clusteringService: clustering.NewService(),
		cache:             newClusteringCache(),
	}
}

// GetClustersParams represents parameters for getting clusters
type GetClustersParams struct {
	MinClusterSize int
	Limit          int
	Offset         int
	DBSCANParams   clustering.DBSCANParams
}

// ClusteringSummary represents the summary of clustering results
type ClusteringSummary struct {
	Clusters      []*clustering.AlertCluster
	NoiseAlertIDs []types.AlertID
	Parameters    clustering.DBSCANParams
	ComputedAt    time.Time
}

// GetAlertClusters retrieves alert clusters with caching
func (uc *ClusteringUseCase) GetAlertClusters(ctx context.Context, params GetClustersParams) (*ClusteringSummary, error) {
	// Generate cache key
	cacheKey := uc.generateCacheKey(params.DBSCANParams)

	// Try to get from cache
	if cached := uc.cache.get(cacheKey); cached != nil {
		return uc.filterAndPaginateClusters(cached, params), nil
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

	// Cache the result
	uc.cache.set(cacheKey, summary)

	// Filter and paginate
	return uc.filterAndPaginateClusters(summary, params), nil
}

// GetClusterAlerts retrieves alerts in a specific cluster with filtering
func (uc *ClusteringUseCase) GetClusterAlerts(ctx context.Context, clusterID string, keyword string, limit, offset int) ([]*alert.Alert, int, error) {
	// Get cached clustering result
	// Note: In a real implementation, we might want to store cluster-alert mappings separately
	// For now, we'll iterate through cached results to find the cluster

	var targetCluster *clustering.AlertCluster
	uc.cache.mu.RLock()
	for _, item := range uc.cache.items {
		for _, cluster := range item.summary.Clusters {
			if cluster.ID == clusterID {
				targetCluster = cluster
				break
			}
		}
		if targetCluster != nil {
			break
		}
	}
	uc.cache.mu.RUnlock()

	if targetCluster == nil {
		return nil, 0, goerr.New("cluster not found").With("clusterID", clusterID)
	}

	// Batch get alerts
	alerts, err := uc.repo.BatchGetAlerts(ctx, targetCluster.AlertIDs)
	if err != nil {
		return nil, 0, goerr.Wrap(err, "failed to get cluster alerts")
	}

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

// CreateTicketFromCluster creates a ticket from selected alerts in a cluster
func (uc *ClusteringUseCase) CreateTicketFromCluster(ctx context.Context, clusterID string, alertIDs []string) error {
	// This will be implemented in the GraphQL resolver by calling existing CreateTicketFromAlerts
	return goerr.New("not implemented")
}

// BindClusterToTicket binds selected alerts from a cluster to an existing ticket
func (uc *ClusteringUseCase) BindClusterToTicket(ctx context.Context, clusterID string, ticketID string, alertIDs []string) error {
	// This will be implemented in the GraphQL resolver by calling existing BindAlertsToTicket
	return goerr.New("not implemented")
}

// Helper methods

func (uc *ClusteringUseCase) generateCacheKey(params clustering.DBSCANParams) string {
	data, _ := json.Marshal(params)
	return "clustering:" + string(data)
}

func (uc *ClusteringUseCase) filterAndPaginateClusters(summary *ClusteringSummary, params GetClustersParams) *ClusteringSummary {
	// Filter by minimum cluster size
	filteredClusters := make([]*clustering.AlertCluster, 0, len(summary.Clusters))
	for _, cluster := range summary.Clusters {
		if cluster.Size >= params.MinClusterSize {
			filteredClusters = append(filteredClusters, cluster)
		}
	}

	// Apply pagination
	start := params.Offset
	if start >= len(filteredClusters) {
		return &ClusteringSummary{
			Clusters:      []*clustering.AlertCluster{},
			NoiseAlertIDs: summary.NoiseAlertIDs,
			Parameters:    summary.Parameters,
			ComputedAt:    summary.ComputedAt,
		}
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
	}
}

// Simple in-memory cache for clustering results
type clusteringCache struct {
	mu    sync.RWMutex
	items map[string]*cacheItem
}

type cacheItem struct {
	summary   *ClusteringSummary
	expiresAt time.Time
}

func newClusteringCache() *clusteringCache {
	cache := &clusteringCache{
		items: make(map[string]*cacheItem),
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

func (c *clusteringCache) get(key string) *ClusteringSummary {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists || time.Now().After(item.expiresAt) {
		return nil
	}

	return item.summary
}

func (c *clusteringCache) set(key string, summary *ClusteringSummary) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &cacheItem{
		summary:   summary,
		expiresAt: time.Now().Add(1 * time.Hour), // 1 hour TTL
	}
}

func (c *clusteringCache) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.expiresAt) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}
