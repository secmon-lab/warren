package config

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/safe"
	"github.com/urfave/cli/v3"
)

type Logger struct {
	level      string
	format     string
	output     string
	quiet      bool
	stacktrace bool
}

func (x *Logger) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "log-level",
			Category:    "logging",
			Aliases:     []string{"l"},
			Sources:     cli.EnvVars("WARREN_LOG_LEVEL"),
			Usage:       "Set log level [debug|info|warn|error]",
			Value:       "info",
			Destination: &x.level,
		},
		&cli.StringFlag{
			Name:        "log-format",
			Category:    "logging",
			Aliases:     []string{"f"},
			Sources:     cli.EnvVars("WARREN_LOG_FORMAT"),
			Usage:       "Set log format [console|json]",
			Value:       "console",
			Destination: &x.format,
		},
		&cli.StringFlag{
			Name:        "log-output",
			Category:    "logging",
			Aliases:     []string{"o"},
			Sources:     cli.EnvVars("WARREN_LOG_OUTPUT"),
			Usage:       "Set log output (create file other than '-', 'stdout', 'stderr')",
			Value:       "stdout",
			Destination: &x.output,
		},
		&cli.BoolFlag{
			Name:        "log-quiet",
			Category:    "logging",
			Aliases:     []string{"q"},
			Usage:       "Quiet mode (no log output)",
			Sources:     cli.EnvVars("WARREN_LOG_QUIET"),
			Destination: &x.quiet,
		},
		&cli.BoolFlag{
			Name:        "log-stacktrace",
			Category:    "logging",
			Aliases:     []string{"s"},
			Usage:       "Show stacktrace (only for console format)",
			Sources:     cli.EnvVars("WARREN_LOG_STACKTRACE"),
			Destination: &x.stacktrace,
			Value:       true,
		},
	}
}

func (x Logger) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("level", x.level),
		slog.String("format", x.format),
		slog.String("output", x.output),
	)
}

// Configure sets up logger and returns closer function and error. You can call closer even if error is not nil.
func (x *Logger) Configure() (func(), error) {
	if x.quiet {
		logging.Quiet()
		return func() {}, nil
	}

	closer := func() {}
	formatMap := map[string]logging.Format{
		"console": logging.FormatConsole,
		"json":    logging.FormatJSON,
	}

	var format logging.Format
	if x.format == "" {
		term := os.Getenv("TERM")
		if strings.Contains(term, "color") || strings.Contains(term, "xterm") {
			format = logging.FormatConsole
		} else {
			format = logging.FormatJSON
		}
	} else {
		fmt, ok := formatMap[x.format]
		if !ok {
			return closer, goerr.New("Invalid log format", goerr.V("format", x.format))
		}
		format = fmt
	}

	levelMap := map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}
	level, ok := levelMap[x.level]
	if !ok {
		return closer, goerr.New("Invalid log level", goerr.V("level", x.level))
	}

	var output io.Writer
	switch x.output {
	case "stdout", "-":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	default:
		f, err := os.OpenFile(filepath.Clean(x.output), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			return closer, goerr.Wrap(err, "Failed to open log file", goerr.V("path", x.output))
		}
		output = f
		closer = func() {
			safe.Close(context.Background(), f)
		}
	}

	logger := logging.New(output, level, format, x.stacktrace)
	logging.SetDefault(logger)

	return closer, nil
}
