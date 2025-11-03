package msg

import (
	"context"
	"fmt"

	"github.com/secmon-lab/warren/pkg/utils/logging"
)

type NotifyFunc func(ctx context.Context, msg string)
type TraceFunc func(ctx context.Context, msg string) func(ctx context.Context, msg string)
type NewUpdatableFunc func(ctx context.Context, msg string) func(ctx context.Context, msg string)

type ctxNotifyFuncKey struct{}
type ctxTraceFuncKey struct{}
type ctxNewUpdatableFuncKey struct{}

func With(ctx context.Context, NotifyFunc NotifyFunc, TraceFunc TraceFunc) context.Context {
	ctx = context.WithValue(ctx, ctxNotifyFuncKey{}, NotifyFunc)
	ctx = context.WithValue(ctx, ctxTraceFuncKey{}, TraceFunc)
	return ctx
}

func WithUpdatable(ctx context.Context, NotifyFunc NotifyFunc, TraceFunc TraceFunc, NewUpdatableFunc NewUpdatableFunc) context.Context {
	ctx = context.WithValue(ctx, ctxNotifyFuncKey{}, NotifyFunc)
	ctx = context.WithValue(ctx, ctxTraceFuncKey{}, TraceFunc)
	ctx = context.WithValue(ctx, ctxNewUpdatableFuncKey{}, NewUpdatableFunc)
	return ctx
}

func Notify(ctx context.Context, format string, args ...any) {
	if v := ctx.Value(ctxNotifyFuncKey{}); v != nil {
		if NotifyFunc, ok := v.(NotifyFunc); ok && NotifyFunc != nil {
			NotifyFunc(ctx, fmt.Sprintf(format, args...))
			return
		}
	}
}

func Trace(ctx context.Context, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if v := ctx.Value(ctxTraceFuncKey{}); v != nil {
		if fn, ok := v.(TraceFunc); ok && fn != nil {
			fn(ctx, msg)
		}
	} else {
		logging.From(ctx).Debug("failed to propagate trace func", "message", msg)
	}
}

func NewUpdatable(ctx context.Context, format string, args ...any) func(ctx context.Context, msg string) {
	if v := ctx.Value(ctxNewUpdatableFuncKey{}); v != nil {
		if fn, ok := v.(NewUpdatableFunc); ok && fn != nil {
			return fn(ctx, fmt.Sprintf(format, args...))
		}
	}
	return func(ctx context.Context, msg string) {}
}

func WithContext(original context.Context) context.Context {
	ctx := original
	ctx = context.WithValue(ctx, ctxNotifyFuncKey{}, original.Value(ctxNotifyFuncKey{}))
	ctx = context.WithValue(ctx, ctxTraceFuncKey{}, original.Value(ctxTraceFuncKey{}))
	ctx = context.WithValue(ctx, ctxTraceFuncKey{}, original.Value(ctxTraceFuncKey{}))
	ctx = context.WithValue(ctx, ctxNewUpdatableFuncKey{}, original.Value(ctxNewUpdatableFuncKey{}))
	return ctx
}
