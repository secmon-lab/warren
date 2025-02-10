package clock

import (
	"context"
	"time"
)

var ctxClockKey = struct{}{}

type Clock func() time.Time

func Now(ctx context.Context) time.Time {
	clock, ok := ctx.Value(ctxClockKey).(Clock)
	if !ok {
		return time.Now()
	}
	return clock()
}

func With(ctx context.Context, clock Clock) context.Context {
	return context.WithValue(ctx, ctxClockKey, clock)
}
