package storage

import (
	"context"
	"io"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
)

type Mock struct {
	data map[string]string
}

var _ interfaces.StorageClient = &Mock{}

func NewMock() *Mock {
	return &Mock{
		data: make(map[string]string),
	}
}

type mockWriter struct {
	data   map[string]string
	object string
}

func (m *mockWriter) Write(p []byte) (n int, err error) {
	m.data[m.object] = string(p)
	return len(p), nil
}

func (m *mockWriter) Close() error {
	return nil
}

func (m *Mock) PutObject(ctx context.Context, object string) io.WriteCloser {
	return &mockWriter{
		data:   m.data,
		object: object,
	}
}

func (m *Mock) GetObject(ctx context.Context, object string) (io.ReadCloser, error) {
	v, ok := m.data[object]
	if !ok {
		return nil, goerr.New("object not found", goerr.V("object", object))
	}
	return io.NopCloser(strings.NewReader(v)), nil
}

func (m *Mock) Close(_ context.Context) {}
