package memory

import (
	"context"
	"sort"

	"github.com/secmon-lab/warren/pkg/domain/model/activity"
)

// Activity related methods
func (r *Memory) PutActivity(ctx context.Context, activity *activity.Activity) error {
	r.activityMu.Lock()
	defer r.activityMu.Unlock()

	r.activities[activity.ID] = activity
	return nil
}

func (r *Memory) GetActivities(ctx context.Context, offset, limit int) ([]*activity.Activity, error) {
	r.activityMu.RLock()
	defer r.activityMu.RUnlock()

	var activities []*activity.Activity
	for _, a := range r.activities {
		activities = append(activities, a)
	}

	// Sort by CreatedAt in descending order (newest first)
	sort.Slice(activities, func(i, j int) bool {
		return activities[i].CreatedAt.After(activities[j].CreatedAt)
	})

	// Apply offset and limit
	start := offset
	if start >= len(activities) {
		return []*activity.Activity{}, nil
	}

	end := start + limit
	if end > len(activities) {
		end = len(activities)
	}

	return activities[start:end], nil
}

func (r *Memory) CountActivities(ctx context.Context) (int, error) {
	r.activityMu.RLock()
	defer r.activityMu.RUnlock()

	return len(r.activities), nil
}
