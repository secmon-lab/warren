package storage

import (
	"context"
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

func (x *Client) Close(ctx context.Context) {
	safe.Close(ctx, x.client)
}
