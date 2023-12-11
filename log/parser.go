package log

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog"
)

func ParseLogLevel(input string) zerolog.Level {
	sanitized := strings.ToLower(strings.TrimSpace(input))

	switch sanitized {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		fmt.Printf("😬 Unable to parse a log level from input: \"%s\". Defaulting to log at INFO level.\n", input)
		return zerolog.InfoLevel
	}
}
