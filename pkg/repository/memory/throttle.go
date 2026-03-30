package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
)

// PutQueuedAlert saves a queued alert
func (r *Memory) PutQueuedAlert(ctx context.Context, qa *alert.QueuedAlert) error {
	r.throttleMu.Lock()
	defer r.throttleMu.Unlock()

	qaCopy := *qa
	r.queuedAlerts[qa.ID] = &qaCopy
	return nil
}

// GetQueuedAlert retrieves a queued alert by ID
func (r *Memory) GetQueuedAlert(ctx context.Context, id types.QueuedAlertID) (*alert.QueuedAlert, error) {
	r.throttleMu.RLock()
	defer r.throttleMu.RUnlock()

	qa, ok := r.queuedAlerts[id]
	if !ok {
		return nil, r.eb.Wrap(goerr.New("queued alert not found"),
			"not found",
			goerr.T(errutil.TagNotFound),
			goerr.V("id", id))
	}

	qaCopy := *qa
	return &qaCopy, nil
}

// ListQueuedAlerts returns queued alerts ordered by CreatedAt ASC (FIFO) with pagination
func (r *Memory) ListQueuedAlerts(ctx context.Context, offset, limit int) ([]*alert.QueuedAlert, error) {
	r.throttleMu.RLock()
	defer r.throttleMu.RUnlock()

	all := make([]*alert.QueuedAlert, 0, len(r.queuedAlerts))
	for _, qa := range r.queuedAlerts {
		qaCopy := *qa
		all = append(all, &qaCopy)
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt.Before(all[j].CreatedAt)
	})

	start := offset
	if start > len(all) {
		start = len(all)
	}
	end := start + limit
	if limit <= 0 || end > len(all) {
		end = len(all)
	}

	return all[start:end], nil
}

// DeleteQueuedAlerts deletes queued alerts by IDs
func (r *Memory) DeleteQueuedAlerts(ctx context.Context, ids []types.QueuedAlertID) error {
	r.throttleMu.Lock()
	defer r.throttleMu.Unlock()

	for _, id := range ids {
		delete(r.queuedAlerts, id)
	}
	return nil
}

// CountQueuedAlerts returns the total number of queued alerts
func (r *Memory) CountQueuedAlerts(ctx context.Context) (int, error) {
	r.throttleMu.RLock()
	defer r.throttleMu.RUnlock()

	return len(r.queuedAlerts), nil
}

// SearchQueuedAlerts searches queued alerts by keyword in title and data
func (r *Memory) SearchQueuedAlerts(ctx context.Context, keyword string, offset, limit int) ([]*alert.QueuedAlert, int, error) {
	r.throttleMu.RLock()
	defer r.throttleMu.RUnlock()

	lowerKeyword := strings.ToLower(keyword)

	var matched []*alert.QueuedAlert
	for _, qa := range r.queuedAlerts {
		if matchesKeyword(qa, lowerKeyword) {
			qaCopy := *qa
			matched = append(matched, &qaCopy)
		}
	}

	sort.Slice(matched, func(i, j int) bool {
		return matched[i].CreatedAt.Before(matched[j].CreatedAt)
	})

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

func matchesKeyword(qa *alert.QueuedAlert, lowerKeyword string) bool {
	if strings.Contains(strings.ToLower(qa.Title), lowerKeyword) {
		return true
	}
	if strings.Contains(strings.ToLower(string(qa.Schema)), lowerKeyword) {
		return true
	}
	dataBytes, err := json.Marshal(qa.Data)
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(dataBytes)), lowerKeyword)
}

// PutReprocessJob saves a reprocess job
func (r *Memory) PutReprocessJob(ctx context.Context, job *alert.ReprocessJob) error {
	r.throttleMu.Lock()
	defer r.throttleMu.Unlock()

	jobCopy := *job
	r.reprocessJobs[job.ID] = &jobCopy
	return nil
}

// GetReprocessJob retrieves a reprocess job by ID
func (r *Memory) GetReprocessJob(ctx context.Context, id types.ReprocessJobID) (*alert.ReprocessJob, error) {
	r.throttleMu.RLock()
	defer r.throttleMu.RUnlock()

	job, ok := r.reprocessJobs[id]
	if !ok {
		return nil, r.eb.Wrap(goerr.New("reprocess job not found"),
			"not found",
			goerr.T(errutil.TagNotFound),
			goerr.V("id", id))
	}

	jobCopy := *job
	return &jobCopy, nil
}

// PutReprocessBatchJob saves a reprocess batch job
func (r *Memory) PutReprocessBatchJob(ctx context.Context, job *alert.ReprocessBatchJob) error {
	r.throttleMu.Lock()
	defer r.throttleMu.Unlock()

	jobCopy := *job
	r.reprocessBatchJobs[job.ID] = &jobCopy
	return nil
}

// GetReprocessBatchJob retrieves a reprocess batch job by ID
func (r *Memory) GetReprocessBatchJob(ctx context.Context, id types.ReprocessBatchJobID) (*alert.ReprocessBatchJob, error) {
	r.throttleMu.RLock()
	defer r.throttleMu.RUnlock()

	job, ok := r.reprocessBatchJobs[id]
	if !ok {
		return nil, r.eb.Wrap(goerr.New("reprocess batch job not found"),
			"not found",
			goerr.T(errutil.TagNotFound),
			goerr.V("id", id))
	}

	jobCopy := *job
	return &jobCopy, nil
}

// countActiveSlots removes expired buckets and returns the total count of active slots.
// Must be called with throttleMu held.
func (r *Memory) countActiveSlots(now time.Time, window time.Duration) int {
	if r.alertThrottle == nil {
		r.alertThrottle = &alert.AlertThrottle{
			Buckets: make(map[string]int),
		}
	}

	cutoff := now.Add(-window)
	total := 0
	for k, v := range r.alertThrottle.Buckets {
		t, err := parseBucketKey(k)
		if err != nil {
			delete(r.alertThrottle.Buckets, k)
			continue
		}
		if t.Before(cutoff) {
			delete(r.alertThrottle.Buckets, k)
		} else {
			total += v
		}
	}
	return total
}

// CheckAlertThrottle checks whether throttle slots are available (read-only).
// Does NOT consume a slot.
func (r *Memory) CheckAlertThrottle(ctx context.Context, window time.Duration, limit int) (*alert.ThrottleResult, error) {
	r.throttleMu.Lock()
	defer r.throttleMu.Unlock()

	now := clock.Now(ctx)
	total := r.countActiveSlots(now, window)

	if total < limit {
		return &alert.ThrottleResult{Allowed: true, ShouldNotify: false}, nil
	}

	// Slot exhausted: check if notification is needed
	shouldNotify := false
	if r.alertThrottle.NotifiedAt.IsZero() || now.Sub(r.alertThrottle.NotifiedAt) >= window {
		r.alertThrottle.NotifiedAt = now
		shouldNotify = true
	}

	return &alert.ThrottleResult{Allowed: false, ShouldNotify: shouldNotify}, nil
}

// AcquireAlertThrottleSlot atomically checks and consumes a throttle slot.
func (r *Memory) AcquireAlertThrottleSlot(ctx context.Context, window time.Duration, limit int) (*alert.ThrottleResult, error) {
	r.throttleMu.Lock()
	defer r.throttleMu.Unlock()

	now := clock.Now(ctx)
	total := r.countActiveSlots(now, window)

	if total < limit {
		bucketKey := toBucketKey(now)
		r.alertThrottle.Buckets[bucketKey] += 1
		return &alert.ThrottleResult{Allowed: true, ShouldNotify: false}, nil
	}

	shouldNotify := false
	if r.alertThrottle.NotifiedAt.IsZero() || now.Sub(r.alertThrottle.NotifiedAt) >= window {
		r.alertThrottle.NotifiedAt = now
		shouldNotify = true
	}

	return &alert.ThrottleResult{Allowed: false, ShouldNotify: shouldNotify}, nil
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
