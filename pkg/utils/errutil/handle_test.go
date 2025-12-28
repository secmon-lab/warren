package errutil_test

import (
	"context"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/request_id"
)

func TestHandle(t *testing.T) {
	// Initialize Sentry with a test transport to capture events
	transport := &testTransport{
		events: make([]*sentry.Event, 0),
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:       "https://test@test.ingest.sentry.io/test",
		Transport: transport,
	})
	gt.NoError(t, err)
	defer sentry.Flush(0)

	t.Run("with request ID", func(t *testing.T) {
		transport.events = nil // Reset events

		ctx := context.Background()
		ctx = request_id.With(ctx, "test-request-id-123")

		testErr := goerr.New("test error with request ID")
		errutil.Handle(ctx, testErr)

		sentry.Flush(0)

		gt.A(t, transport.events).Length(1)
		event := transport.events[0]
		gt.V(t, event.Tags["request_id"]).Equal("test-request-id-123")
	})

	t.Run("without request ID", func(t *testing.T) {
		transport.events = nil // Reset events

		ctx := context.Background()

		testErr := goerr.New("test error without request ID")
		errutil.Handle(ctx, testErr)

		sentry.Flush(0)

		gt.A(t, transport.events).Length(1)
		event := transport.events[0]
		_, exists := event.Tags["request_id"]
		gt.False(t, exists)
	})

	t.Run("with empty request ID", func(t *testing.T) {
		transport.events = nil // Reset events

		ctx := context.Background()
		ctx = request_id.With(ctx, "")

		testErr := goerr.New("test error with empty request ID")
		errutil.Handle(ctx, testErr)

		sentry.Flush(0)

		gt.A(t, transport.events).Length(1)
		event := transport.events[0]
		_, exists := event.Tags["request_id"]
		gt.False(t, exists)
	})
}

// testTransport is a custom Sentry transport for testing
type testTransport struct {
	events []*sentry.Event
}

func (t *testTransport) Configure(options sentry.ClientOptions) {}

func (t *testTransport) SendEvent(event *sentry.Event) {
	t.events = append(t.events, event)
}

func (t *testTransport) Flush(timeout time.Duration) bool {
	return true
}

func (t *testTransport) FlushWithContext(ctx context.Context) bool {
	return true
}

func (t *testTransport) Close() {}
