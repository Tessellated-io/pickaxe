package log

import (
	"os"
	"strings"

	"github.com/dpotapov/slogpfx"
	"log/slog"
)

// Logger is a wrapper around a slog.logger, which keeps track of prefixes and raw keys,
// such that prefixese can be added in a hierarchical manner.
type Logger struct {
	*slog.Logger

	rawLogLevel string
	prefixes    []string
}

// Default logger is simply at INFO level.
func Default() *Logger {
	return NewLogger("info")
}

// Create a new logger without a prefix
func NewLogger(rawLogLevel string) *Logger {
	return NewLoggerWithPrefixes(rawLogLevel, []string{})
}

// Create a new logger with a set of prefixes.
func NewLoggerWithPrefixes(rawLogLevel string, prefixes []string) *Logger {
	slogger := newLoggerWithLogLevel(rawLogLevel)
	return newLoggerWithSlogger(slogger, rawLogLevel, prefixes)
}

func newLoggerWithSlogger(slogger *slog.Logger, rawLogLevel string, prefixes []string) *Logger {
	// Set the prefix key to always be the prefix
	prefix := strings.Join(prefixes, "")
	prefixedSlogger := slogger.With(prefixKey, prefix)

	return &Logger{
		Logger:      prefixedSlogger,
		rawLogLevel: rawLogLevel,
		prefixes:    prefixes,
	}
}

// Add an additional prefix to the logger
func (l *Logger) ApplyPrefix(prefix string) *Logger {
	return newLoggerWithSlogger(l.Logger, l.rawLogLevel, append(l.prefixes, prefix))
}

// Add a value to the logger
func (l *Logger) With(args ...any) *Logger {
	slogger := l.Logger.With(args...)
	return newLoggerWithSlogger(slogger, l.rawLogLevel, l.prefixes)
}

// Prefix key is the "magic" key that makes this all work. Any value sent to this key is a prefix,
// with the intermediate handlers.
const prefixKey = "_prefixKey"

func newLoggerWithLogLevel(rawLogLevel string) *slog.Logger {
	// Get the default logging level
	loggingLevel := ParseLogLevel(rawLogLevel)
	lvl := new(slog.LevelVar)
	lvl.Set(loggingLevel)

	// This is the standard way to get slog to log to stdout
	textHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvl,
	})

	// Custom prefix formatter. The default in slogpfx uses a '>' symbol.
	prefixFormatter := func(prefixes []slog.Value) string {
		p := make([]string, 0, len(prefixes))
		for _, prefix := range prefixes {
			if prefix.Any() == nil || prefix.String() == "" {
				continue // skip empty prefixes
			}
			p = append(p, prefix.String())
		}
		if len(p) == 0 {
			return ""
		}
		return strings.Join(p, "") + " "
	}

	// Handler that adds the prefix.
	prefixHandler := slogpfx.NewHandler(textHandler, &slogpfx.HandlerOptions{
		PrefixKeys:      []string{prefixKey},
		PrefixFormatter: prefixFormatter,
	})

	// Create a new slogger with the handler
	logger := slog.New(prefixHandler)

	return logger
}
