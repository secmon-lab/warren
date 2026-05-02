package storage

import (
	"context"
	"errors"
	"io"

	"cloud.google.com/go/storage"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/utils/safe"
	"google.golang.org/api/option"
)

type Client struct {
	client *storage.Client
	bucket string
}

var _ interfaces.StorageClient = &Client{}

func New(ctx context.Context, bucket string, opts ...option.ClientOption) (*Client, error) {
	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create storage client")
	}

	return &Client{
		client: client,
		bucket: bucket,
	}, nil
}

func (x *Client) PutObject(ctx context.Context, object string) io.WriteCloser {
	return x.client.Bucket(x.bucket).Object(object).NewWriter(ctx)
}

func (x *Client) GetObject(ctx context.Context, object string) (io.ReadCloser, error) {
	rc, err := x.client.Bucket(x.bucket).Object(object).NewReader(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create reader",
			goerr.V("bucket", x.bucket),
			goerr.V("object", object),
		)
	}

	return rc, nil
}

// CopyObject copies src to dst inside the same bucket using GCS's
// server-side copy API. No bytes traverse the caller's process; the
// rewrite runs entirely on Google's side. A missing source surfaces
// as storage.ErrObjectNotExist, wrapped so callers can discriminate.
func (x *Client) CopyObject(ctx context.Context, src, dst string) error {
	bucket := x.client.Bucket(x.bucket)
	_, err := bucket.Object(dst).CopierFrom(bucket.Object(src)).Run(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return goerr.Wrap(err, "source object not found",
				goerr.V("bucket", x.bucket),
				goerr.V("src", src),
				goerr.V("dst", dst),
			)
		}
		return goerr.Wrap(err, "failed to copy object",
			goerr.V("bucket", x.bucket),
			goerr.V("src", src),
			goerr.V("dst", dst),
		)
	}
	return nil
}

func (x *Client) DeleteObject(ctx context.Context, object string) error {
	err := x.client.Bucket(x.bucket).Object(object).Delete(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil
		}
		return goerr.Wrap(err, "failed to delete object",
			goerr.V("bucket", x.bucket),
			goerr.V("object", object),
		)
	}
	return nil
}

func (x *Client) Close(ctx context.Context) {
	safe.Close(ctx, x.client)
}
