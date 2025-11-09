package firestore

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/activity"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"google.golang.org/api/iterator"
)

// Activity related methods
func (r *Firestore) PutActivity(ctx context.Context, activity *activity.Activity) error {
	doc := r.db.Collection(collectionActivities).Doc(activity.ID.String())
	_, err := doc.Set(ctx, activity)
	if err != nil {
		return goerr.Wrap(err, "failed to put activity", goerr.V("activity_id", activity.ID))
	}

	return nil
}

func (r *Firestore) GetActivities(ctx context.Context, offset, limit int) ([]*activity.Activity, error) {
	iter := r.db.Collection(collectionActivities).
		OrderBy("CreatedAt", firestore.Desc).
		Offset(offset).
		Limit(limit).
		Documents(ctx)

	var activities []*activity.Activity
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get next activity")
		}

		var a activity.Activity
		if err := doc.DataTo(&a); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to activity")
		}

		activities = append(activities, &a)
	}

	return activities, nil
}

func (r *Firestore) CountActivities(ctx context.Context) (int, error) {
	countQuery := r.db.Collection(collectionActivities).NewAggregationQuery().WithCount("count")
	result, err := countQuery.Get(ctx)
	if err != nil {
		return 0, goerr.Wrap(err, "failed to get activity count")
	}

	return extractCountFromAggregationResult(result, "count")
}

func (r *Firestore) DeleteActivity(ctx context.Context, activityID types.ActivityID) error {
	_, err := r.db.Collection(collectionActivities).Doc(activityID.String()).Delete(ctx)
	if err != nil {
		return goerr.Wrap(err, "failed to delete activity", goerr.V("activity_id", activityID))
	}
	return nil
}
