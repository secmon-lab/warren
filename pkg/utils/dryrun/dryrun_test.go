package dryrun_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/utils/dryrun"
)

func TestDryRun(t *testing.T) {
	runTest := func(tc struct {
		name          string
		isDryRun      bool
		expectedValue bool
	}) func(t *testing.T) {
		return func(t *testing.T) {
			ctx := context.Background()

			// Test With function
			newCtx := dryrun.With(ctx, tc.isDryRun)

			// Test From function
			retrievedValue := dryrun.From(newCtx)
			gt.Equal(t, tc.expectedValue, retrievedValue)

			// Test IsDryRun function
			isDryRunResult := dryrun.IsDryRun(newCtx)
			gt.Equal(t, tc.expectedValue, isDryRunResult)
		}
	}

	t.Run("dry-run enabled", runTest(struct {
		name          string
		isDryRun      bool
		expectedValue bool
	}{
		name:          "dry-run enabled",
		isDryRun:      true,
		expectedValue: true,
	}))

	t.Run("dry-run disabled", runTest(struct {
		name          string
		isDryRun      bool
		expectedValue bool
	}{
		name:          "dry-run disabled",
		isDryRun:      false,
		expectedValue: false,
	}))
}

func TestDryRunDefault(t *testing.T) {
	ctx := context.Background()

	// Test default behavior when no dry-run context is set
	isDryRun := dryrun.IsDryRun(ctx)
	gt.Equal(t, false, isDryRun)

	value := dryrun.From(ctx)
	gt.Equal(t, false, value)
}

func TestDryRunContextPropagation(t *testing.T) {
	ctx := context.Background()

	// Set dry-run to true
	ctx1 := dryrun.With(ctx, true)
	gt.Equal(t, true, dryrun.IsDryRun(ctx1))

	// Create a child context and verify propagation
	ctx2, cancel := context.WithCancel(ctx1)
	defer cancel()
	gt.Equal(t, true, dryrun.IsDryRun(ctx2))

	// Override with false
	ctx3 := dryrun.With(ctx2, false)
	gt.Equal(t, false, dryrun.IsDryRun(ctx3))
}
