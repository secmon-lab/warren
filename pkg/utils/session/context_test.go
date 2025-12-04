package session_test

import (
	"context"
	"errors"
	"testing"

	"github.com/secmon-lab/warren/pkg/utils/session"
)

func TestWithStatusCheck_and_CheckStatus(t *testing.T) {
	ctx := context.Background()

	t.Run("no check function set returns nil", func(t *testing.T) {
		err := session.CheckStatus(ctx)
		if err != nil {
			t.Errorf("CheckStatus() should return nil when no check function is set, got %v", err)
		}
	})

	t.Run("check function returns nil", func(t *testing.T) {
		checkFunc := func(ctx context.Context) error {
			return nil
		}

		ctx = session.WithStatusCheck(ctx, checkFunc)
		err := session.CheckStatus(ctx)
		if err != nil {
			t.Errorf("CheckStatus() should return nil when check function returns nil, got %v", err)
		}
	})

	t.Run("check function returns error", func(t *testing.T) {
		expectedErr := errors.New("session aborted")
		checkFunc := func(ctx context.Context) error {
			return expectedErr
		}

		ctx = session.WithStatusCheck(ctx, checkFunc)
		err := session.CheckStatus(ctx)
		if err != expectedErr {
			t.Errorf("CheckStatus() = %v, want %v", err, expectedErr)
		}
	})

	t.Run("check function is called with context", func(t *testing.T) {
		type testKey string
		const key testKey = "test"
		expectedValue := "test-value"

		ctx = context.WithValue(ctx, key, expectedValue)

		var capturedValue string
		checkFunc := func(ctx context.Context) error {
			if v, ok := ctx.Value(key).(string); ok {
				capturedValue = v
			}
			return nil
		}

		ctx = session.WithStatusCheck(ctx, checkFunc)
		_ = session.CheckStatus(ctx)

		if capturedValue != expectedValue {
			t.Errorf("check function received wrong context value: %v, want %v", capturedValue, expectedValue)
		}
	})

	t.Run("nil check function is gracefully handled", func(t *testing.T) {
		ctx = session.WithStatusCheck(ctx, nil)
		err := session.CheckStatus(ctx)
		if err != nil {
			t.Errorf("CheckStatus() should return nil for nil check function, got %v", err)
		}
	})
}

func TestCheckStatus_MultipleChecks(t *testing.T) {
	ctx := context.Background()

	callCount := 0
	checkFunc := func(ctx context.Context) error {
		callCount++
		return nil
	}

	ctx = session.WithStatusCheck(ctx, checkFunc)

	// Call multiple times
	for i := 0; i < 5; i++ {
		_ = session.CheckStatus(ctx)
	}

	if callCount != 5 {
		t.Errorf("check function was called %d times, want 5", callCount)
	}
}

func TestCheckStatus_DifferentErrors(t *testing.T) {
	tests := []struct {
		name          string
		checkFunc     session.StatusCheckFunc
		expectedError error
	}{
		{
			name: "custom error 1",
			checkFunc: func(ctx context.Context) error {
				return errors.New("error 1")
			},
			expectedError: errors.New("error 1"),
		},
		{
			name: "custom error 2",
			checkFunc: func(ctx context.Context) error {
				return errors.New("error 2")
			},
			expectedError: errors.New("error 2"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = session.WithStatusCheck(ctx, tt.checkFunc)

			err := session.CheckStatus(ctx)
			if err == nil {
				t.Error("CheckStatus() should return error")
				return
			}

			if err.Error() != tt.expectedError.Error() {
				t.Errorf("CheckStatus() = %v, want %v", err, tt.expectedError)
			}
		})
	}
}
