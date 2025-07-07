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

func Write(ctx context.Context, w io.Writer, data []byte) {
	if w == nil {
		return
	}
	if _, err := w.Write(data); err != nil {
		logging.From(ctx).Error("Failed to write", slog.Any("error", err))
	}
}

func Copy(ctx context.Context, dst io.Writer, src io.Reader) {
	if _, err := io.Copy(dst, src); err != nil {
		logging.From(ctx).Error("Failed to copy", slog.Any("error", err))
	}
}
