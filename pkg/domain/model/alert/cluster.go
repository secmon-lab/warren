package alert

import (
	"context"
	"math"
	"sort"
)

type cluster struct {
	Alerts []*Alert
}

type clusters []*cluster

func ClusterAlerts(ctx context.Context, alerts []*Alert, similarityThreshold float64, topN int) [][]*Alert {
	clusters := newAlertCluster(alerts, similarityThreshold)

	sort.Slice(clusters, func(i, j int) bool {
		return len(clusters[i].Alerts) > len(clusters[j].Alerts)
	})

	if topN > 0 && topN < len(clusters) {
		clusters = clusters[:topN]
	}

	alertSets := make([][]*Alert, len(clusters))
	for i, cluster := range clusters {
		alertSets[i] = cluster.Alerts
	}
	return alertSets
}

func newAlertCluster(alerts []*Alert, similarityThreshold float64) clusters {
	// Initialize clusters
	clusters := make(clusters, 0)

	// Process each alert
	for _, alert := range alerts {
		if len(alert.Embedding) == 0 {
			continue // Skip alerts without embeddings
		}

		// Try to find matching cluster
		matched := false
		for j := range clusters {
			// Compare with first alert in cluster as representative
			if CosineSimilarity(alert.Embedding, clusters[j].Alerts[0].Embedding) >= similarityThreshold {
				clusters[j].Alerts = append(clusters[j].Alerts, alert)
				matched = true
				break
			}
		}

		// Create new cluster if no match found
		if !matched {
			clusters = append(clusters, &cluster{Alerts: []*Alert{alert}})
		}
	}

	return clusters
}

func (x Alerts) MaxSimilarity() float64 {
	max := 0.0

	for i, a := range x {
		for j := i + 1; j < len(x); j++ {
			if CosineSimilarity(a.Embedding, x[j].Embedding) > max {
				max = CosineSimilarity(a.Embedding, x[j].Embedding)
			}
		}
	}

	return max
}

func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float64
	var magnitudeA, magnitudeB float64

	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		magnitudeA += float64(a[i]) * float64(a[i])
		magnitudeB += float64(b[i]) * float64(b[i])
	}

	return dotProduct / (math.Sqrt(magnitudeA) * math.Sqrt(magnitudeB))
}
