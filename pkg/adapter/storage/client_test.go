package storage

import (
	"bytes"
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
	prefix := os.Getenv("TEST_STORAGE_PREFIX") + "test-" + time.Now().Format("20060102150405")

	ctx := context.Background()
	client, err := New(ctx)
	gt.NoError(t, err)
	defer client.Close(ctx)

	objectName := prefix + "/test.txt"
	testData := []byte("test data")

	t.Run("PutObject", func(t *testing.T) {
		err := client.PutObject(ctx, bucket, objectName, bytes.NewReader(testData))
		gt.NoError(t, err)
	})

	t.Run("GetObject", func(t *testing.T) {
		rc, err := client.GetObject(ctx, bucket, objectName)
		gt.NoError(t, err)
		defer rc.Close()

		data, err := io.ReadAll(rc)
		gt.NoError(t, err)
		gt.Array(t, data).Equal(testData)
	})

	t.Run("GetObject not found", func(t *testing.T) {
		_, err := client.GetObject(ctx, bucket, "non-existent-object")
		gt.Error(t, err)
		gt.True(t, errors.Is(err, storage.ErrObjectNotExist))
	})
}
