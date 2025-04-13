package storage

import (
	"context"
	"io"

	"cloud.google.com/go/storage"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/utils/safe"
	"google.golang.org/api/option"
)

type Client struct {
	client *storage.Client
}

func New(ctx context.Context, opts ...option.ClientOption) (*Client, error) {
	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create storage client")
	}

	return &Client{
		client: client,
	}, nil
}

func (x *Client) PutObject(ctx context.Context, bucket, object string, r io.Reader) error {
	wc := x.client.Bucket(bucket).Object(object).NewWriter(ctx)
	if _, err := io.Copy(wc, r); err != nil {
		return goerr.Wrap(err, "failed to copy data to storage",
			goerr.V("bucket", bucket),
			goerr.V("object", object),
		)
	}

	if err := wc.Close(); err != nil {
		return goerr.Wrap(err, "failed to close writer",
			goerr.V("bucket", bucket),
			goerr.V("object", object),
		)
	}

	return nil
}

func (x *Client) GetObject(ctx context.Context, bucket, object string) (io.ReadCloser, error) {
	rc, err := x.client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create reader",
			goerr.V("bucket", bucket),
			goerr.V("object", object),
		)
	}

	return rc, nil
}

func (x *Client) Close(ctx context.Context) {
	safe.Close(ctx, x.client)
}
