package msg

import (
	"context"
	"fmt"

	"github.com/secmon-lab/warren/pkg/utils/logging"
)

type NotifyFunc func(ctx context.Context, msg string)
type TraceFunc func(ctx context.Context, msg string)
type WarnFunc func(ctx context.Context, msg string)

type ctxNotifyFuncKey struct{}
type ctxTraceFuncKey struct{}
type ctxWarnFuncKey struct{}

func With(ctx context.Context, NotifyFunc NotifyFunc, TraceFunc TraceFunc, WarnFunc WarnFunc) context.Context {
	ctx = context.WithValue(ctx, ctxNotifyFuncKey{}, NotifyFunc)
	ctx = context.WithValue(ctx, ctxTraceFuncKey{}, TraceFunc)
	ctx = context.WithValue(ctx, ctxWarnFuncKey{}, WarnFunc)
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

func Warn(ctx context.Context, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if v := ctx.Value(ctxWarnFuncKey{}); v != nil {
		if fn, ok := v.(WarnFunc); ok && fn != nil {
			fn(ctx, msg)
		}
	} else {
		logging.From(ctx).Debug("failed to propagate warn func", "message", msg)
	}
}

func WithContext(original context.Context) context.Context {
	ctx := original
	ctx = context.WithValue(ctx, ctxNotifyFuncKey{}, original.Value(ctxNotifyFuncKey{}))
	ctx = context.WithValue(ctx, ctxTraceFuncKey{}, original.Value(ctxTraceFuncKey{}))
	ctx = context.WithValue(ctx, ctxWarnFuncKey{}, original.Value(ctxWarnFuncKey{}))
	return ctx
}
