package config

import (
	"context"
	"log/slog"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/adapter/storage"
	"google.golang.org/api/option"

	"github.com/urfave/cli/v3"
)

type Storage struct {
	bucket    string
	prefix    string
	projectID string
}

func (x *Storage) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "storage-bucket",
			Usage:       "Storage bucket",
			Category:    "Storage",
			Destination: &x.bucket,
			Sources:     cli.EnvVars("WARREN_STORAGE_BUCKET"),
		},
		&cli.StringFlag{
			Name:        "storage-prefix",
			Usage:       "Storage prefix",
			Category:    "Storage",
			Destination: &x.prefix,
			Sources:     cli.EnvVars("WARREN_STORAGE_PREFIX"),
		},
		&cli.StringFlag{
			Name:        "storage-project-id",
			Usage:       "Storage project ID",
			Category:    "Storage",
			Destination: &x.projectID,
			Sources:     cli.EnvVars("WARREN_STORAGE_PROJECT_ID"),
		},
	}
}

func (x *Storage) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("bucket", x.bucket),
		slog.String("prefix", x.prefix),
		slog.String("project_id", x.projectID),
	)
}

func (x *Storage) Configure(ctx context.Context) (*storage.Client, error) {
	if x.bucket == "" {
		return nil, goerr.New("storage bucket is not set")
	}

	var opts []option.ClientOption
	if x.projectID != "" {
		opts = append(opts, option.WithQuotaProject(x.projectID))
	}

	client, err := storage.New(ctx, x.bucket, x.prefix, opts...)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// Bucket returns the bucket name (exported for serve command)
func (x *Storage) Bucket() string {
	return x.bucket
}

// IsConfigured returns true if Storage is configured
func (x *Storage) IsConfigured() bool {
	return x.bucket != ""
}
