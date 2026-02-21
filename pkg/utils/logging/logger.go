package logging

import (
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/fatih/color"
	"github.com/m-mizutani/clog"
	"github.com/m-mizutani/clog/hooks"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/masq"
)

type Format int

const (
	FormatConsole Format = iota + 1
	FormatJSON
)

var (
	defaultLogger = slog.Default()
	loggerMutex   sync.Mutex
)

func Default() *slog.Logger {
	return defaultLogger
}

func Quiet() {
	loggerMutex.Lock()
	defaultLogger = slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	loggerMutex.Unlock()
}

func goerrNoStacktrace(_ []string, attr slog.Attr) *clog.HandleAttr {
	if goErr, ok := attr.Value.Any().(*goerr.Error); ok {
		var attrs []any
		for k, v := range goErr.Values() {
			attrs = append(attrs, slog.Any(k, v))
		}
		attrs = append(attrs, slog.Any("cause", goErr.Error()))
		newAttr := slog.Group(attr.Key, attrs...)

		return &clog.HandleAttr{
			NewAttr: &newAttr,
		}
	}

	return nil
}

func New(w io.Writer, level slog.Level, format Format, stacktrace bool) *slog.Logger {
	filter := masq.New(
		masq.WithTag("secret"),
		masq.WithFieldPrefix("secret_"),
		masq.WithFieldName("Authorization"),
	)

	attrHook := hooks.GoErr()
	if !stacktrace {
		attrHook = goerrNoStacktrace
	}

	var handler slog.Handler
	switch format {
	case FormatConsole:
		handler = clog.New(
			clog.WithWriter(w),
			clog.WithLevel(level),
			clog.WithReplaceAttr(filter),
			// clog.WithSource(true),
			// clog.WithTimeFmt("2006-01-02 15:04:05"),
			clog.WithAttrHook(attrHook),
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

	return slog.New(handler)
}

func SetDefault(logger *slog.Logger) {
	loggerMutex.Lock()
	defaultLogger = logger
	loggerMutex.Unlock()
}

func ErrAttr(err error) slog.Attr { return slog.Any("error", err) }
