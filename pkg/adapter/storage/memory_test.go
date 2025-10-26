package storage_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/adapter/storage"
)

func TestMemoryClient_BasicOperations(t *testing.T) {
	client := storage.NewMemoryClient()
	ctx := context.Background()
	defer client.Close(ctx)

	t.Run("put and get object", func(t *testing.T) {
		// Test data
		objectName := "test-object"
		testData := []byte("hello world")

		// Put object
		writer := client.PutObject(ctx, objectName)
		n, err := writer.Write(testData)
		gt.NoError(t, err)
		gt.Equal(t, len(testData), n)
		gt.NoError(t, writer.Close())

		// Get object
		reader, err := client.GetObject(ctx, objectName)
		gt.NoError(t, err)
		defer func() {
			_ = reader.Close()
		}()

		retrievedData, err := io.ReadAll(reader)
		gt.NoError(t, err)
		gt.Equal(t, testData, retrievedData)
	})

	t.Run("get nonexistent object", func(t *testing.T) {
		_, err := client.GetObject(ctx, "nonexistent")
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("object not found")
	})

	t.Run("overwrite existing object", func(t *testing.T) {
		objectName := "overwrite-test"
		firstData := []byte("first data")
		secondData := []byte("second data")

		// Write first data
		writer := client.PutObject(ctx, objectName)
		_, err := writer.Write(firstData)
		gt.NoError(t, err)
		gt.NoError(t, writer.Close())

		// Overwrite with second data
		writer = client.PutObject(ctx, objectName)
		_, err = writer.Write(secondData)
		gt.NoError(t, err)
		gt.NoError(t, writer.Close())

		// Verify second data is stored
		reader, err := client.GetObject(ctx, objectName)
		gt.NoError(t, err)
		defer func() {
			_ = reader.Close()
		}()

		retrievedData, err := io.ReadAll(reader)
		gt.NoError(t, err)
		gt.Equal(t, secondData, retrievedData)
	})

	t.Run("write to closed writer", func(t *testing.T) {
		writer := client.PutObject(ctx, "closed-writer-test")
		gt.NoError(t, writer.Close())

		// Try to write after close
		_, err := writer.Write([]byte("should fail"))
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("writer is closed")
	})

	t.Run("close writer multiple times", func(t *testing.T) {
		writer := client.PutObject(ctx, "multiple-close-test")
		_, err := writer.Write([]byte("test data"))
		gt.NoError(t, err)

		// Close multiple times should not error
		gt.NoError(t, writer.Close())
		gt.NoError(t, writer.Close())
		gt.NoError(t, writer.Close())
	})
}

func TestMemoryClient_ConcurrentAccess(t *testing.T) {
	client := storage.NewMemoryClient()
	ctx := context.Background()
	defer client.Close(ctx)

	// Test concurrent writes to different objects
	const numWorkers = 10
	errCh := make(chan error, numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func(id int) {
			objectName := fmt.Sprintf("object-%d", id)
			testData := []byte(fmt.Sprintf("data-%d", id))

			writer := client.PutObject(ctx, objectName)
			if _, err := writer.Write(testData); err != nil {
				errCh <- err
				return
			}
			if err := writer.Close(); err != nil {
				errCh <- err
				return
			}

			errCh <- nil
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numWorkers; i++ {
		err := <-errCh
		gt.NoError(t, err)
	}

	// Verify all objects were stored correctly
	for i := 0; i < numWorkers; i++ {
		objectName := fmt.Sprintf("object-%d", i)
		expectedData := []byte(fmt.Sprintf("data-%d", i))

		reader, err := client.GetObject(ctx, objectName)
		gt.NoError(t, err)

		retrievedData, err := io.ReadAll(reader)
		_ = reader.Close()
		gt.NoError(t, err)
		gt.Equal(t, expectedData, retrievedData)
	}
}

func TestMemoryClient_EmptyData(t *testing.T) {
	client := storage.NewMemoryClient()
	ctx := context.Background()
	defer client.Close(ctx)

	t.Run("store empty data", func(t *testing.T) {
		objectName := "empty-object"

		// Write empty data
		writer := client.PutObject(ctx, objectName)
		n, err := writer.Write([]byte{})
		gt.NoError(t, err)
		gt.Equal(t, 0, n)
		gt.NoError(t, writer.Close())

		// Retrieve empty data
		reader, err := client.GetObject(ctx, objectName)
		gt.NoError(t, err)
		defer func() { _ = reader.Close() }()

		retrievedData, err := io.ReadAll(reader)
		gt.NoError(t, err)
		gt.Equal(t, []byte{}, retrievedData)
	})
}

func TestMemoryClient_LargeData(t *testing.T) {
	client := storage.NewMemoryClient()
	ctx := context.Background()
	defer client.Close(ctx)

	t.Run("store large data", func(t *testing.T) {
		objectName := "large-object"
		// Create 1MB of data
		largeData := bytes.Repeat([]byte("x"), 1024*1024)

		// Write large data
		writer := client.PutObject(ctx, objectName)
		n, err := writer.Write(largeData)
		gt.NoError(t, err)
		gt.Equal(t, len(largeData), n)
		gt.NoError(t, writer.Close())

		// Retrieve large data
		reader, err := client.GetObject(ctx, objectName)
		gt.NoError(t, err)
		defer func() { _ = reader.Close() }()

		retrievedData, err := io.ReadAll(reader)
		gt.NoError(t, err)
		gt.Equal(t, largeData, retrievedData)
	})
}
