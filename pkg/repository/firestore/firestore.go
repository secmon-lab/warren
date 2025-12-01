package firestore

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/firestore/apiv1/firestorepb"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
)

type Firestore struct {
	db *firestore.Client
	eb *goerr.Builder
}

var _ interfaces.Repository = &Firestore{}

func New(ctx context.Context, projectID, databaseID string) (*Firestore, error) {
	db, err := firestore.NewClientWithDatabase(ctx, projectID, databaseID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create firestore client")
	}

	return &Firestore{
		db: db,
		eb: goerr.NewBuilder(
			goerr.TV(errs.RepositoryKey, "firestore"),
			goerr.V("project_id", projectID),
			goerr.V("database_id", databaseID),
		),
	}, nil
}

func (r *Firestore) Close() error {
	return r.db.Close()
}

const (
	collectionAlerts            = "alerts"
	collectionAlertLists        = "lists"
	collectionHistories         = "histories"
	collectionTickets           = "tickets"
	collectionComments          = "comments"
	collectionTokens            = "tokens"
	collectionActivities        = "activities"
	collectionTags              = "tags"
	collectionNotices           = "notices"
	collectionExecutionMemories = "execution_memories"
	collectionTicketMemories    = "ticket_memories"
	collectionAgents            = "agents"
	collectionSessions          = "sessions"
	subcollectionMemories       = "memories"
)

// extractCountFromAggregationResult extracts an integer count from a Firestore aggregation result.
// It handles both int64 and *firestorepb.Value types that can be returned by the Firestore client.
func extractCountFromAggregationResult(result firestore.AggregationResult, alias string) (int, error) {
	countVal, ok := result[alias]
	if !ok {
		return 0, goerr.New("count alias not found in aggregation result",
			goerr.V("alias", alias),
			goerr.T(errs.TagInternal))
	}

	switch v := countVal.(type) {
	case int64:
		return int(v), nil
	case *firestorepb.Value:
		if v != nil && v.ValueType != nil {
			if _, okType := v.ValueType.(*firestorepb.Value_IntegerValue); okType {
				return int(v.GetIntegerValue()), nil
			}
			return 0, goerr.New("firestorepb.Value from count is not an integer type",
				goerr.V("value_type", fmt.Sprintf("%T", v.ValueType)),
				goerr.V("alias", alias),
				goerr.T(errs.TagInternal))
		}
		return 0, goerr.New("count value is a nil or invalid *firestorepb.Value",
			goerr.V("alias", alias),
			goerr.T(errs.TagInternal))
	default:
		return 0, goerr.New("unexpected count value type from Firestore aggregation",
			goerr.V("type", fmt.Sprintf("%T", v)),
			goerr.V("value", v),
			goerr.V("alias", alias),
			goerr.T(errs.TagInternal))
	}
}

// Helper function to check if embedding is invalid
func isInvalidEmbedding(embedding []float32) bool {
	if len(embedding) == 0 {
		return true
	}

	// Check if all values are zero
	for _, v := range embedding {
		if v != 0 {
			return false
		}
	}
	return true
}
