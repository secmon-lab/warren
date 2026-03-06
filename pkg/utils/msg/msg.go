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

func With(ctx context.Context, notifyFunc NotifyFunc, traceFunc TraceFunc, warnFunc WarnFunc) context.Context {
	ctx = context.WithValue(ctx, ctxNotifyFuncKey{}, notifyFunc)
	ctx = context.WithValue(ctx, ctxTraceFuncKey{}, traceFunc)
	ctx = context.WithValue(ctx, ctxWarnFuncKey{}, warnFunc)
	return ctx
}

func Notify(ctx context.Context, format string, args ...any) {
	if v := ctx.Value(ctxNotifyFuncKey{}); v != nil {
		if notifyFunc, ok := v.(NotifyFunc); ok && notifyFunc != nil {
			notifyFunc(ctx, fmt.Sprintf(format, args...))
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

// CopyTo copies msg handler functions from src context to dst context.
func CopyTo(dst context.Context, src context.Context) context.Context {
	dst = context.WithValue(dst, ctxNotifyFuncKey{}, src.Value(ctxNotifyFuncKey{}))
	dst = context.WithValue(dst, ctxTraceFuncKey{}, src.Value(ctxTraceFuncKey{}))
	dst = context.WithValue(dst, ctxWarnFuncKey{}, src.Value(ctxWarnFuncKey{}))
	return dst
}
