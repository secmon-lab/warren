package clock

import (
	"context"
	"time"
)

type ctxClockKey struct{}

type Clock func() time.Time

func Now(ctx context.Context) time.Time {
	clock, ok := ctx.Value(ctxClockKey{}).(Clock)
	if !ok {
		return time.Now()
	}
	return clock()
}

func Since(ctx context.Context, t time.Time) time.Duration {
	return Now(ctx).Sub(t)
}

func With(ctx context.Context, clock Clock) context.Context {
	return context.WithValue(ctx, ctxClockKey{}, clock)
}

type ctxTimezoneKey struct{}

func WithTimezone(ctx context.Context, location *time.Location) context.Context {
	return context.WithValue(ctx, ctxTimezoneKey{}, location)
}

func Timezone(ctx context.Context) *time.Location {
	location, ok := ctx.Value(ctxTimezoneKey{}).(*time.Location)
	if !ok {
		return time.Local
	}
	return location
}
