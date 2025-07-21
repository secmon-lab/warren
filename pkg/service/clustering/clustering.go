package clustering

import (
	"context"
	"math"
	"sort"

	"cloud.google.com/go/firestore"
	goerr "github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// DBSCANParams represents parameters for DBSCAN clustering algorithm
type DBSCANParams struct {
	Eps        float64 // Maximum distance between two samples for one to be considered as in the neighborhood
	MinSamples int     // Minimum number of samples in a neighborhood for a point to be considered as a core point
}

// AlertCluster represents a cluster of alerts
type AlertCluster struct {
	ID            string
	CenterAlertID types.AlertID   // ID of the alert closest to the cluster center
	AlertIDs      []types.AlertID // IDs of all alerts in the cluster
	Size          int
	Keywords      []string // Common keywords (optional)
}

// ClusteringResult represents the result of clustering
type ClusteringResult struct {
	Clusters      []*AlertCluster
	NoiseAlertIDs []types.AlertID // IDs of alerts that don't belong to any cluster
	Parameters    DBSCANParams
}

// Service defines the interface for clustering service
type Service interface {
	// ClusterAlerts performs DBSCAN clustering on alerts using their embedding vectors
	// Note: Only unbound alerts (TicketID == nil) should be included
	ClusterAlerts(ctx context.Context, alerts []*alert.Alert, params DBSCANParams) (*ClusteringResult, error)

	// FindCenterAlert finds the alert closest to the cluster center
	FindCenterAlert(ctx context.Context, alerts []*alert.Alert) (*alert.Alert, error)

	// ExtractKeywords extracts common keywords from alerts (optional)
	ExtractKeywords(ctx context.Context, alerts []*alert.Alert, limit int) ([]string, error)
}

// NewService creates a new clustering service instance
func NewService() Service {
	return &service{}
}

type service struct{}

// ClusterAlerts implements DBSCAN clustering algorithm
func (s *service) ClusterAlerts(ctx context.Context, alerts []*alert.Alert, params DBSCANParams) (*ClusteringResult, error) {
	if len(alerts) == 0 {
		return &ClusteringResult{
			Clusters:      []*AlertCluster{},
			NoiseAlertIDs: []types.AlertID{},
			Parameters:    params,
		}, nil
	}

	// Filter alerts with embeddings
	alertsWithEmbedding := make([]*alert.Alert, 0, len(alerts))
	for _, a := range alerts {
		if len(a.Embedding) > 0 {
			alertsWithEmbedding = append(alertsWithEmbedding, a)
		}
	}

	if len(alertsWithEmbedding) == 0 {
		return &ClusteringResult{
			Clusters:      []*AlertCluster{},
			NoiseAlertIDs: []types.AlertID{},
			Parameters:    params,
		}, nil
	}

	// Initialize DBSCAN
	n := len(alertsWithEmbedding)
	labels := make([]int, n) // -1: noise, 0+: cluster ID
	for i := range labels {
		labels[i] = -1
	}

	clusterID := 0

	// DBSCAN algorithm
	for i := 0; i < n; i++ {
		if labels[i] != -1 {
			continue // Already processed
		}

		neighbors := s.getNeighbors(alertsWithEmbedding, i, params.Eps)
		// In DBSCAN, we need to count the point itself
		if len(neighbors)+1 < params.MinSamples {
			continue // Mark as noise (already -1)
		}

		// Start a new cluster
		labels[i] = clusterID
		seeds := make([]int, len(neighbors))
		copy(seeds, neighbors)

		// Process seeds
		for len(seeds) > 0 {
			currentPoint := seeds[0]
			seeds = seeds[1:]

			if labels[currentPoint] == -1 {
				labels[currentPoint] = clusterID
			}

			if labels[currentPoint] != -1 && labels[currentPoint] != clusterID {
				continue // Already in another cluster
			}

			currentNeighbors := s.getNeighbors(alertsWithEmbedding, currentPoint, params.Eps)
			// In DBSCAN, we need to count the point itself
			if len(currentNeighbors)+1 >= params.MinSamples {
				for _, neighbor := range currentNeighbors {
					if labels[neighbor] == -1 {
						seeds = append(seeds, neighbor)
					}
				}
			}
		}

		clusterID++
	}

	// Build result
	clusterMap := make(map[int][]int)
	noiseIndices := []int{}

	for i, label := range labels {
		if label == -1 {
			noiseIndices = append(noiseIndices, i)
		} else {
			clusterMap[label] = append(clusterMap[label], i)
		}
	}

	// Create clusters
	clusters := make([]*AlertCluster, 0, len(clusterMap))
	for cid, indices := range clusterMap {
		clusterAlerts := make([]*alert.Alert, len(indices))
		alertIDs := make([]types.AlertID, len(indices))
		for i, idx := range indices {
			clusterAlerts[i] = alertsWithEmbedding[idx]
			alertIDs[i] = alertsWithEmbedding[idx].ID
		}

		centerAlert, err := s.FindCenterAlert(ctx, clusterAlerts)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to find center alert")
		}

		cluster := &AlertCluster{
			ID:            generateClusterID(cid),
			CenterAlertID: centerAlert.ID,
			AlertIDs:      alertIDs,
			Size:          len(alertIDs),
		}

		clusters = append(clusters, cluster)
	}

	// Sort clusters by size (descending)
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].Size > clusters[j].Size
	})

	// Build noise alert IDs
	noiseAlertIDs := make([]types.AlertID, len(noiseIndices))
	for i, idx := range noiseIndices {
		noiseAlertIDs[i] = alertsWithEmbedding[idx].ID
	}

	return &ClusteringResult{
		Clusters:      clusters,
		NoiseAlertIDs: noiseAlertIDs,
		Parameters:    params,
	}, nil
}

// FindCenterAlert finds the alert closest to the geometric center of the cluster
func (s *service) FindCenterAlert(ctx context.Context, alerts []*alert.Alert) (*alert.Alert, error) {
	if len(alerts) == 0 {
		return nil, goerr.New("no alerts provided")
	}

	if len(alerts) == 1 {
		return alerts[0], nil
	}

	// Calculate centroid
	dim := len(alerts[0].Embedding)
	centroid := make([]float32, dim)

	for _, a := range alerts {
		for i, val := range a.Embedding {
			centroid[i] += val
		}
	}

	for i := range centroid {
		centroid[i] /= float32(len(alerts))
	}

	// Find alert closest to centroid
	var closestAlert *alert.Alert
	minDistance := math.MaxFloat64

	for _, a := range alerts {
		distance := s.cosineDistance(centroid, a.Embedding)
		if distance < minDistance {
			minDistance = distance
			closestAlert = a
		}
	}

	return closestAlert, nil
}

// ExtractKeywords extracts common keywords from alerts
func (s *service) ExtractKeywords(ctx context.Context, alerts []*alert.Alert, limit int) ([]string, error) {
	// This is a placeholder implementation
	// Actual implementation would depend on the structure of Alert.Data
	// and might involve:
	// - JSON parsing and field extraction
	// - Text analysis from metadata
	// - LLM-based summarization
	return []string{}, nil
}

// getNeighbors finds all points within eps distance from the given point
func (s *service) getNeighbors(alerts []*alert.Alert, pointIdx int, eps float64) []int {
	neighbors := []int{}
	point := alerts[pointIdx]

	for i, other := range alerts {
		if i == pointIdx {
			continue
		}

		distance := s.cosineDistance(point.Embedding, other.Embedding)
		if distance <= eps {
			neighbors = append(neighbors, i)
		}
	}

	return neighbors
}

// cosineDistance calculates cosine distance between two vectors
// Cosine distance = 1 - cosine similarity
func (s *service) cosineDistance(v1, v2 firestore.Vector32) float64 {
	if len(v1) != len(v2) {
		return 1.0 // Maximum distance for mismatched dimensions
	}

	var dotProduct, norm1, norm2 float64
	for i := range v1 {
		dotProduct += float64(v1[i]) * float64(v2[i])
		norm1 += float64(v1[i]) * float64(v1[i])
		norm2 += float64(v2[i]) * float64(v2[i])
	}

	if norm1 == 0 || norm2 == 0 {
		return 1.0 // Maximum distance if one vector is zero
	}

	cosineSimilarity := dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
	return 1.0 - cosineSimilarity
}

// generateClusterID generates a human-readable cluster ID
func generateClusterID(clusterNum int) string {
	adjectives := []string{
		"swift", "brave", "bright", "clever", "gentle", "fierce", "quiet", "bold",
		"calm", "quick", "strong", "wise", "alert", "sharp", "agile", "keen",
		"vital", "smart", "fast", "noble", "proud", "steady", "fresh", "clear",
		"warm", "cool", "light", "deep", "soft", "hard", "wide", "tall",
	}

	nouns := []string{
		"eagle", "tiger", "wolf", "bear", "fox", "hawk", "lion", "deer",
		"whale", "shark", "horse", "bird", "fish", "cat", "dog", "owl",
		"ram", "elk", "bee", "ant", "frog", "duck", "goat", "pig",
		"cow", "hen", "rat", "bat", "fly", "bug", "oak", "pine",
	}

	adjIndex := clusterNum % len(adjectives)
	nounIndex := (clusterNum / len(adjectives)) % len(nouns)

	return adjectives[adjIndex] + "-" + nouns[nounIndex]
}
