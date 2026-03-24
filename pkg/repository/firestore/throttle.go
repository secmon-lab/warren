package firestore

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	collectionQueuedAlerts  = "queued_alerts"
	collectionReprocessJobs = "reprocess_jobs"
	collectionThrottle      = "throttle"
	docAlertThrottle        = "alert"
)

// PutQueuedAlert saves a queued alert to Firestore
func (r *Firestore) PutQueuedAlert(ctx context.Context, qa *alert.QueuedAlert) error {
	_, err := r.db.Collection(collectionQueuedAlerts).Doc(string(qa.ID)).Set(ctx, qa)
	if err != nil {
		return r.eb.Wrap(err, "failed to put queued alert", goerr.V("id", qa.ID))
	}
	return nil
}

// GetQueuedAlert retrieves a queued alert by ID
func (r *Firestore) GetQueuedAlert(ctx context.Context, id types.QueuedAlertID) (*alert.QueuedAlert, error) {
	doc, err := r.db.Collection(collectionQueuedAlerts).Doc(string(id)).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, r.eb.Wrap(goerr.New("queued alert not found"),
				"not found",
				goerr.T(errutil.TagNotFound),
				goerr.V("id", id))
		}
		return nil, r.eb.Wrap(err, "failed to get queued alert", goerr.V("id", id))
	}

	var qa alert.QueuedAlert
	if err := doc.DataTo(&qa); err != nil {
		return nil, r.eb.Wrap(err, "failed to decode queued alert", goerr.V("id", id))
	}
	return &qa, nil
}

// ListQueuedAlerts returns queued alerts ordered by CreatedAt ASC with pagination
func (r *Firestore) ListQueuedAlerts(ctx context.Context, offset, limit int) ([]*alert.QueuedAlert, error) {
	query := r.db.Collection(collectionQueuedAlerts).OrderBy("CreatedAt", firestore.Asc)
	if offset > 0 {
		query = query.Offset(offset)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}

	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, r.eb.Wrap(err, "failed to list queued alerts")
	}

	results := make([]*alert.QueuedAlert, 0, len(docs))
	for _, doc := range docs {
		var qa alert.QueuedAlert
		if err := doc.DataTo(&qa); err != nil {
			return nil, r.eb.Wrap(err, "failed to decode queued alert")
		}
		results = append(results, &qa)
	}
	return results, nil
}

// DeleteQueuedAlerts deletes queued alerts by IDs
func (r *Firestore) DeleteQueuedAlerts(ctx context.Context, ids []types.QueuedAlertID) error {
	bw := r.db.BulkWriter(ctx)
	for _, id := range ids {
		ref := r.db.Collection(collectionQueuedAlerts).Doc(string(id))
		if _, err := bw.Delete(ref); err != nil {
			return r.eb.Wrap(err, "failed to schedule delete for queued alert", goerr.V("id", id))
		}
	}
	bw.End()
	return nil
}

// CountQueuedAlerts returns the total number of queued alerts
func (r *Firestore) CountQueuedAlerts(ctx context.Context) (int, error) {
	result, err := r.db.Collection(collectionQueuedAlerts).NewAggregationQuery().WithCount("count").Get(ctx)
	if err != nil {
		return 0, r.eb.Wrap(err, "failed to count queued alerts")
	}
	return extractCountFromAggregationResult(result, "count")
}

// SearchQueuedAlerts searches queued alerts by keyword in title and data.
// Firestore doesn't support full-text search, so we fetch all and filter in memory.
func (r *Firestore) SearchQueuedAlerts(ctx context.Context, keyword string, offset, limit int) ([]*alert.QueuedAlert, int, error) {
	docs, err := r.db.Collection(collectionQueuedAlerts).OrderBy("CreatedAt", firestore.Asc).Documents(ctx).GetAll()
	if err != nil {
		return nil, 0, r.eb.Wrap(err, "failed to search queued alerts")
	}

	lowerKeyword := strings.ToLower(keyword)
	var matched []*alert.QueuedAlert
	for _, doc := range docs {
		var qa alert.QueuedAlert
		if err := doc.DataTo(&qa); err != nil {
			continue
		}
		if matchesQueuedAlertKeyword(&qa, lowerKeyword) {
			matched = append(matched, &qa)
		}
	}

	totalCount := len(matched)

	start := offset
	if start > len(matched) {
		start = len(matched)
	}
	end := start + limit
	if limit <= 0 || end > len(matched) {
		end = len(matched)
	}

	return matched[start:end], totalCount, nil
}

func matchesQueuedAlertKeyword(qa *alert.QueuedAlert, lowerKeyword string) bool {
	if strings.Contains(strings.ToLower(qa.Title), lowerKeyword) {
		return true
	}
	dataBytes, err := json.Marshal(qa.Data)
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(dataBytes)), lowerKeyword)
}

// PutReprocessJob saves a reprocess job
func (r *Firestore) PutReprocessJob(ctx context.Context, job *alert.ReprocessJob) error {
	_, err := r.db.Collection(collectionReprocessJobs).Doc(string(job.ID)).Set(ctx, job)
	if err != nil {
		return r.eb.Wrap(err, "failed to put reprocess job", goerr.V("id", job.ID))
	}
	return nil
}

// GetReprocessJob retrieves a reprocess job by ID
func (r *Firestore) GetReprocessJob(ctx context.Context, id types.ReprocessJobID) (*alert.ReprocessJob, error) {
	doc, err := r.db.Collection(collectionReprocessJobs).Doc(string(id)).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, r.eb.Wrap(goerr.New("reprocess job not found"),
				"not found",
				goerr.T(errutil.TagNotFound),
				goerr.V("id", id))
		}
		return nil, r.eb.Wrap(err, "failed to get reprocess job", goerr.V("id", id))
	}

	var job alert.ReprocessJob
	if err := doc.DataTo(&job); err != nil {
		return nil, r.eb.Wrap(err, "failed to decode reprocess job", goerr.V("id", id))
	}
	return &job, nil
}

const bucketKeyFormat = "2006-01-02T15:04"

func toBucketKey(t time.Time) string {
	return t.UTC().Format(bucketKeyFormat)
}

func parseBucketKey(key string) (time.Time, error) {
	t, err := time.Parse(bucketKeyFormat, key)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid bucket key: %s", key)
	}
	return t, nil
}

// firestoreThrottle is the Firestore representation of AlertThrottle
type firestoreThrottle struct {
	Buckets    map[string]int `firestore:"buckets"`
	NotifiedAt time.Time      `firestore:"notified_at"`
}

// AcquireAlertThrottleSlot atomically checks and consumes a throttle slot using Firestore transaction.
func (r *Firestore) AcquireAlertThrottleSlot(ctx context.Context, window time.Duration, limit int) (*alert.ThrottleResult, error) {
	docRef := r.db.Collection(collectionThrottle).Doc(docAlertThrottle)

	var result alert.ThrottleResult
	err := r.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		doc, err := tx.Get(docRef)

		var throttle firestoreThrottle
		if err != nil {
			if status.Code(err) == codes.NotFound {
				throttle = firestoreThrottle{
					Buckets: make(map[string]int),
				}
			} else {
				return goerr.Wrap(err, "failed to get throttle document")
			}
		} else {
			if err := doc.DataTo(&throttle); err != nil {
				return goerr.Wrap(err, "failed to decode throttle document")
			}
			if throttle.Buckets == nil {
				throttle.Buckets = make(map[string]int)
			}
		}

		now := time.Now().UTC()
		cutoff := now.Add(-window)
		bucketKey := toBucketKey(now)

		// Remove expired buckets and count remaining
		total := 0
		keysToDelete := make([]string, 0)
		for k, v := range throttle.Buckets {
			t, parseErr := parseBucketKey(k)
			if parseErr != nil {
				keysToDelete = append(keysToDelete, k)
				continue
			}
			if t.Before(cutoff) {
				keysToDelete = append(keysToDelete, k)
			} else {
				total += v
			}
		}
		for _, k := range keysToDelete {
			delete(throttle.Buckets, k)
		}

		if total < limit {
			throttle.Buckets[bucketKey] += 1
			result = alert.ThrottleResult{Allowed: true, ShouldNotify: false}
		} else {
			shouldNotify := false
			if throttle.NotifiedAt.IsZero() || now.Sub(throttle.NotifiedAt) >= window {
				throttle.NotifiedAt = now
				shouldNotify = true
			}
			result = alert.ThrottleResult{Allowed: false, ShouldNotify: shouldNotify}
		}

		return tx.Set(docRef, throttle)
	})

	if err != nil {
		return nil, r.eb.Wrap(err, "failed to acquire alert throttle slot")
	}

	return &result, nil
}
