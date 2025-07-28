package storage

import (
	"bytes"
	"context"
	"io"
	"sync"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
)

type MemoryClient struct {
	mu      sync.RWMutex
	objects map[string][]byte
}

var _ interfaces.StorageClient = &MemoryClient{}

func NewMemoryClient() *MemoryClient {
	return &MemoryClient{
		objects: make(map[string][]byte),
	}
}

func (m *MemoryClient) PutObject(ctx context.Context, object string) io.WriteCloser {
	return &memoryWriter{
		client: m,
		object: object,
		buffer: &bytes.Buffer{},
	}
}

func (m *MemoryClient) GetObject(ctx context.Context, object string) (io.ReadCloser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, exists := m.objects[object]
	if !exists {
		return nil, goerr.New("object not found", goerr.V("object", object))
	}

	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *MemoryClient) Close(ctx context.Context) {
	// Nothing to do for development purposes
}

type memoryWriter struct {
	client *MemoryClient
	object string
	buffer *bytes.Buffer
	closed bool
	mu     sync.Mutex
}

func (w *memoryWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return 0, goerr.New("writer is closed")
	}

	return w.buffer.Write(p)
}

func (w *memoryWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}

	data := w.buffer.Bytes()

	w.client.mu.Lock()
	defer w.client.mu.Unlock()

	// Store the new object
	w.client.objects[w.object] = data
	w.closed = true

	return nil
}
