package graphql

import (
	"context"
	"time"

	goerr "github.com/m-mizutani/goerr/v2"
	graphql1 "github.com/secmon-lab/warren/pkg/domain/model/graphql"
	"github.com/secmon-lab/warren/pkg/service/clustering"
	"github.com/secmon-lab/warren/pkg/usecase"
)

// convertToGraphQLClusteringSummary converts usecase clustering summary to GraphQL model
func (r *queryResolver) convertToGraphQLClusteringSummary(ctx context.Context, summary *usecase.ClusteringSummary) (*graphql1.ClusteringSummary, error) {
	clusters := make([]*graphql1.AlertCluster, len(summary.Clusters))
	for i, cluster := range summary.Clusters {
		graphqlCluster, err := r.convertToGraphQLAlertCluster(ctx, cluster)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to convert alert cluster")
		}
		clusters[i] = graphqlCluster
	}

	// Fetch noise alerts in batch to avoid N+1 query problem
	noiseAlerts, err := r.repo.BatchGetAlerts(ctx, summary.NoiseAlertIDs)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get noise alerts")
	}

	return &graphql1.ClusteringSummary{
		Clusters:    clusters,
		NoiseAlerts: noiseAlerts,
		Parameters: &graphql1.DBSCANParameters{
			Eps:        summary.Parameters.Eps,
			MinSamples: summary.Parameters.MinSamples,
		},
		ComputedAt: summary.ComputedAt.Format("2006-01-02T15:04:05Z07:00"),
		TotalCount: summary.TotalCount,
	}, nil
}

// convertToGraphQLAlertCluster converts clustering service alert cluster to GraphQL model
func (r *queryResolver) convertToGraphQLAlertCluster(ctx context.Context, cluster *clustering.AlertCluster) (*graphql1.AlertCluster, error) {
	// Fetch center alert
	centerAlert, err := r.repo.GetAlert(ctx, cluster.CenterAlertID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get center alert", goerr.V("centerAlertID", cluster.CenterAlertID))
	}

	// Fetch all alerts in the cluster in batch to avoid N+1 query problem
	alerts, err := r.repo.BatchGetAlerts(ctx, cluster.AlertIDs)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get cluster alerts")
	}

	return &graphql1.AlertCluster{
		ID:          cluster.ID,
		CenterAlert: centerAlert,
		Alerts:      alerts,
		Size:        cluster.Size,
		Keywords:    cluster.Keywords,
		CreatedAt:   time.Now().Format("2006-01-02T15:04:05Z07:00"), // Use current time as creation time
	}, nil
}
