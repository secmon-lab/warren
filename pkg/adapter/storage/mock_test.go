package storage_test

import (
	"context"
	"io"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/adapter/storage"
)

func TestMock(t *testing.T) {
	type testCase struct {
		object string
		data   string
	}

	runTest := func(tc testCase) func(t *testing.T) {
		return func(t *testing.T) {
			ctx := context.Background()
			mock := storage.NewMock()

			writer := mock.PutObject(ctx, tc.object)
			_, err := writer.Write([]byte(tc.data))
			gt.NoError(t, err)
			err = writer.Close()
			gt.NoError(t, err)

			reader, err := mock.GetObject(ctx, tc.object)
			gt.NoError(t, err)
			defer func() { _ = reader.Close() }()

			readData, err := io.ReadAll(reader)
			gt.NoError(t, err)
			gt.Value(t, string(readData)).Equal(tc.data)

			mock.Close(ctx)
		}
	}

	t.Run("success case", runTest(testCase{
		object: "test.txt",
		data:   "Hello, World!",
	}))
}
