package trace

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"cloud.google.com/go/storage"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem/trace"
	"google.golang.org/api/option"
)

// Repository implements trace.Repository for GCS backend storage.
type Repository struct {
	client *storage.Client
	bucket string
	prefix string
}

var _ trace.Repository = &Repository{}

// New creates a new GCS Trace Repository.
func New(ctx context.Context, bucket string, opts ...option.ClientOption) (*Repository, error) {
	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create GCS client for trace repository")
	}

	return &Repository{
		client: client,
		bucket: bucket,
		prefix: "traces/",
	}, nil
}

// WithPrefix sets the object prefix for trace data.
func (r *Repository) WithPrefix(prefix string) *Repository {
	r.prefix = prefix
	return r
}

// Save persists trace data to GCS as JSON.
func (r *Repository) Save(ctx context.Context, t *trace.Trace) error {
	objectPath := fmt.Sprintf("%s%s.json", r.prefix, t.TraceID)

	w := r.client.Bucket(r.bucket).Object(objectPath).NewWriter(ctx)
	w.ContentType = "application/json"

	if err := json.NewEncoder(w).Encode(t); err != nil {
		_ = w.Close()
		return goerr.Wrap(err, "failed to encode trace data",
			goerr.V("bucket", r.bucket),
			goerr.V("object", objectPath),
		)
	}

	if err := w.Close(); err != nil {
		return goerr.Wrap(err, "failed to write trace data to GCS",
			goerr.V("bucket", r.bucket),
			goerr.V("object", objectPath),
		)
	}

	return nil
}

// Close closes the underlying GCS client.
func (r *Repository) Close() error {
	return r.client.Close()
}

// safeRepository wraps a trace.Repository and logs errors instead of propagating them.
// This ensures trace save failures never affect agent execution results.
type safeRepository struct {
	inner  trace.Repository
	logger *slog.Logger
}

var _ trace.Repository = &safeRepository{}

// NewSafe wraps a trace.Repository so that Save errors are logged as warnings
// instead of being returned. This prevents trace storage failures from affecting
// agent execution.
func NewSafe(repo trace.Repository, logger *slog.Logger) trace.Repository {
	return &safeRepository{
		inner:  repo,
		logger: logger,
	}
}

func (r *safeRepository) Save(ctx context.Context, t *trace.Trace) error {
	if err := r.inner.Save(ctx, t); err != nil {
		r.logger.WarnContext(ctx, "failed to save trace data", "error", err, "trace_id", t.TraceID)
	}
	return nil
}
