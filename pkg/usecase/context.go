package usecase

import (
	"context"

	"github.com/secmon-lab/warren/pkg/utils/lang"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/thread"
)

func newBackgroundContext(ctx context.Context) context.Context {
	newCtx := context.Background()

	newCtx = logging.With(newCtx, logging.From(ctx))
	newCtx = thread.WithReplyFunc(newCtx, thread.ReplyFuncFrom(ctx))
	newCtx = lang.With(newCtx, lang.From(ctx))

	return newCtx
}
