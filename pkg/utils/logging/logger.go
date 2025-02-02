package logging

import (
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/m-mizutani/clog"
	"github.com/m-mizutani/masq"
)

type Format int

const (
	FormatConsole Format = iota + 1
	FormatJSON
)

var (
	logger      = slog.Default()
	loggerMutex sync.Mutex
)

func Default() *slog.Logger {
	return logger
}

func ReconfigureLogger(w io.Writer, level slog.Level, format Format) {
	filter := masq.New(
		masq.WithTag("secret"),
		masq.WithTag("quiet"),
		masq.WithFieldPrefix("secret_"),
		masq.WithAllowedType(reflect.TypeOf(time.Time{})),
	)

	var handler slog.Handler
	switch format {
	case FormatConsole:
		handler = clog.New(
			clog.WithWriter(w),
			clog.WithLevel(level),
			clog.WithReplaceAttr(filter),
			// clog.WithSource(true),
			// clog.WithTimeFmt("2006-01-02 15:04:05"),
			clog.WithColorMap(&clog.ColorMap{
				Level: map[slog.Level]*color.Color{
					slog.LevelDebug: color.New(color.FgGreen, color.Bold),
					slog.LevelInfo:  color.New(color.FgCyan, color.Bold),
					slog.LevelWarn:  color.New(color.FgYellow, color.Bold),
					slog.LevelError: color.New(color.FgRed, color.Bold),
				},
				LevelDefault: color.New(color.FgBlue, color.Bold),
				Time:         color.New(color.FgWhite),
				Message:      color.New(color.FgHiWhite),
				AttrKey:      color.New(color.FgHiCyan),
				AttrValue:    color.New(color.FgHiWhite),
			}),
		)

	case FormatJSON:
		handler = slog.NewJSONHandler(w, &slog.HandlerOptions{
			AddSource:   true,
			Level:       level,
			ReplaceAttr: filter,
		})

	default:
		panic("Unsupported log format: " + fmt.Sprintf("%d", format))
	}

	loggerMutex.Lock()
	logger = slog.New(handler)
	loggerMutex.Unlock()
}

func ErrAttr(err error) slog.Attr { return slog.Any("error", err) }
