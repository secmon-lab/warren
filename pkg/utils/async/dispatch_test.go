package async_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/lang"
	"github.com/secmon-lab/warren/pkg/utils/async"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/user"
)

func TestDispatch(t *testing.T) {
	t.Run("executes handler asynchronously", func(t *testing.T) {
		ctx := context.Background()
		var wg sync.WaitGroup
		executed := false

		wg.Add(1)
		async.Dispatch(ctx, func(ctx context.Context) error {
			defer wg.Done()
			executed = true
			return nil
		})

		wg.Wait()
		gt.True(t, executed)
	})

	t.Run("handles errors without crashing", func(t *testing.T) {
		ctx := context.Background()
		var wg sync.WaitGroup

		wg.Add(1)
		async.Dispatch(ctx, func(ctx context.Context) error {
			defer wg.Done()
			return errors.New("test error")
		})

		wg.Wait()
		// Test passes if no panic occurs
	})

	t.Run("recovers from panic", func(t *testing.T) {
		ctx := context.Background()
		done := make(chan bool, 1)

		async.Dispatch(ctx, func(ctx context.Context) error {
			defer func() {
				done <- true
			}()
			panic("test panic")
		})

		select {
		case <-done:
			// Test passes if panic was recovered
		case <-time.After(1 * time.Second):
			t.Fatal("handler did not complete within timeout")
		}
	})

	t.Run("preserves context values", func(t *testing.T) {
		ctx := context.Background()

		// Set context values
		ctx = user.WithUserID(ctx, "test-user")
		ctx = lang.With(ctx, lang.Japanese)
		ctx = logging.With(ctx, logging.Default())

		var wg sync.WaitGroup
		wg.Add(1)

		async.Dispatch(ctx, func(newCtx context.Context) error {
			defer wg.Done()

			// Check preserved values
			gt.Equal(t, user.FromContext(newCtx), "test-user")
			gt.Equal(t, lang.From(newCtx), lang.Japanese)
			gt.NotNil(t, logging.From(newCtx))

			return nil
		})

		wg.Wait()
	})

	t.Run("creates new background context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		var wg sync.WaitGroup
		wg.Add(1)

		async.Dispatch(ctx, func(newCtx context.Context) error {
			defer wg.Done()

			// Cancel original context
			cancel()

			// New context should not be affected
			select {
			case <-newCtx.Done():
				t.Error("new context was cancelled")
			default:
				// Expected: context is not cancelled
			}

			return nil
		})

		wg.Wait()
	})
}
