package clock_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

func TestClock(t *testing.T) {
	now := time.Now()
	c := func() time.Time {
		return now
	}
	ctx := context.Background()
	ctx = clock.With(ctx, c)
	gt.Equal(t, clock.Now(ctx), now)
}
