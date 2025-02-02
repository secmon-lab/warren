package safe

import (
	"context"
	"io"
	"log/slog"

	"github.com/secmon-lab/warren/pkg/utils/logging"
)

func Close(ctx context.Context, closer io.Closer) {
	if closer == nil {
		return
	}
	if err := closer.Close(); err != nil {
		logging.From(ctx).Error("Failed to close", slog.Any("error", err))
	}
}
