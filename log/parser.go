package log

import (
	"fmt"
	"log/slog"
	"strings"
)

func ParseLogLevel(input string) slog.Level {
	sanitized := strings.ToLower(strings.TrimSpace(input))

	switch sanitized {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "silent":
		return slog.LevelError + 1
	default:
		fmt.Printf("😬 Unable to parse a log level from input: \"%s\". Defaulting to log at INFO level.\n", input)
		return slog.LevelInfo
	}
}
