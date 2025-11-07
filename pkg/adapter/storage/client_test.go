package storage

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/m-mizutani/gt"
)

func TestClient(t *testing.T) {
	bucket := os.Getenv("TEST_STORAGE_BUCKET")
	if bucket == "" {
		t.Skip("TEST_STORAGE_BUCKET is not set")
	}
	prefix := os.Getenv("TEST_STORAGE_PREFIX") + "test-" + time.Now().Format("20060102150405") + "/"

	ctx := context.Background()
	client, err := New(ctx, bucket)
	gt.NoError(t, err)
	defer client.Close(ctx)

	objectName := prefix + "test.txt"
	testData := []byte("test data")

	t.Run("PutObject", func(t *testing.T) {
		w := client.PutObject(ctx, objectName)
		_, err := w.Write(testData)
		gt.NoError(t, err).Required()
		gt.NoError(t, w.Close())
	})

	t.Run("GetObject", func(t *testing.T) {
		rc, err := client.GetObject(ctx, objectName)
		gt.NoError(t, err)
		defer func() {
			_ = rc.Close()
		}()

		data, err := io.ReadAll(rc)
		gt.NoError(t, err)
		gt.Array(t, data).Equal(testData)
	})

	t.Run("GetObject not found", func(t *testing.T) {
		_, err := client.GetObject(ctx, "non-existent-object")
		gt.Error(t, err)
		gt.True(t, errors.Is(err, storage.ErrObjectNotExist))
	})
}
