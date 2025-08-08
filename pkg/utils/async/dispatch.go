package async

import (
	"context"
	"runtime/debug"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/model/lang"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"github.com/secmon-lab/warren/pkg/utils/user"
)

// Dispatch executes a handler function asynchronously with proper context and panic recovery
func Dispatch(ctx context.Context, handler func(ctx context.Context) error) {
	newCtx := newBackgroundContext(ctx)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				errs.Handle(newCtx, goerr.New("panic in async handler",
					goerr.V("recover", r),
					goerr.V("stack", string(stack))))
			}
		}()

		if err := handler(newCtx); err != nil {
			errs.Handle(newCtx, err)
		}
	}()
}

// newBackgroundContext creates a new background context preserving important values
func newBackgroundContext(ctx context.Context) context.Context {
	newCtx := context.Background()
	newCtx = logging.With(newCtx, logging.From(ctx))
	newCtx = msg.WithContext(newCtx)
	newCtx = lang.With(newCtx, lang.From(ctx))
	if userID := user.FromContext(ctx); userID != "" {
		newCtx = user.WithUserID(newCtx, userID)
	}
	return newCtx
}
