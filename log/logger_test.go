package log_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/tessellated-io/pickaxe/log"
)
func TestLogging_NoPrefix(t *testing.T) {
	logger := log.NewLogger("info")

	assert.Equal(t, "level=INFO msg=test\n", getLogValueWithoutTimestamp(logger, "test"))
	assert.Equal(t, "level=INFO msg=test key=value foo=bar\n", getLogValueWithoutTimestamp(logger, "test", "key", "value", "foo", "bar"))

	logger = logger.With("key", "value", "foo", "bar")
	assert.Equal(t, "level=INFO msg=test key=value foo=bar\n", getLogValueWithoutTimestamp(logger, "test"))
}

func TestLogging_ApplyPrefix(t *testing.T) {
	logger := log.NewLogger("info")

	logger = logger.ApplyPrefix("[PREFIX1]")
	assert.Equal(t, "level=INFO msg=\"[PREFIX1] test\"\n", getLogValueWithoutTimestamp(logger, "test"))
	assert.Equal(t, "level=INFO msg=\"[PREFIX1] test\" key=value\n", getLogValueWithoutTimestamp(logger, "test", "key", "value"))

	logger = logger.With("key", "value")
	assert.Equal(t, "level=INFO msg=\"[PREFIX1] test\" key=value\n", getLogValueWithoutTimestamp(logger, "test"))

	logger = logger.ApplyPrefix("[SECOND]")
	assert.Equal(t, "level=INFO msg=\"[PREFIX1][SECOND] test\" key=value\n", getLogValueWithoutTimestamp(logger, "test"))
	assert.Equal(t, "level=INFO msg=\"[PREFIX1][SECOND] test\" key=value foo=bar\n", getLogValueWithoutTimestamp(logger, "test", "foo", "bar"))

	logger = logger.With("foo", "bar")
	assert.Equal(t, "level=INFO msg=\"[PREFIX1][SECOND] test\" key=value foo=bar\n", getLogValueWithoutTimestamp(logger, "test"))
}

func TestLogging_DefaultPrefix(t *testing.T) {
	logger := log.NewLoggerWithPrefixes("info", []string{"[PREFIX1]"})

	assert.Equal(t, "level=INFO msg=\"[PREFIX1] test\"\n", getLogValueWithoutTimestamp(logger, "test"))
	assert.Equal(t, "level=INFO msg=\"[PREFIX1] test\" key=value\n", getLogValueWithoutTimestamp(logger, "test", "key", "value"))

	logger = logger.With("key", "value")
	assert.Equal(t, "level=INFO msg=\"[PREFIX1] test\" key=value\n", getLogValueWithoutTimestamp(logger, "test"))

	logger = logger.ApplyPrefix("[SECOND]")
	assert.Equal(t, "level=INFO msg=\"[PREFIX1][SECOND] test\" key=value\n", getLogValueWithoutTimestamp(logger, "test"))
	assert.Equal(t, "level=INFO msg=\"[PREFIX1][SECOND] test\" key=value foo=bar\n", getLogValueWithoutTimestamp(logger, "test", "foo", "bar"))

	logger = logger.With("foo", "bar")
	assert.Equal(t, "level=INFO msg=\"[PREFIX1][SECOND] test\" key=value foo=bar\n", getLogValueWithoutTimestamp(logger, "test"))
}

func TestLogging_WithPrefix(t *testing.T) {
	logger := log.NewLogger("info")

	assert.Equal(t, "level=INFO msg=test\n", getLogValueWithoutTimestamp(logger, "test"))
	assert.Equal(t, "level=INFO msg=test key=value foo=bar\n", getLogValueWithoutTimestamp(logger, "test", "key", "value", "foo", "bar"))
}

func getLogValueWithoutTimestamp(logger *log.Logger, msg string, vals ...any) string {
	// io.Writer that writes to a buffer
	buffer := bytes.NewBuffer([]byte{})

	// swizzle the logging handler to write to the buffer
	handler := logger.Logger.Handler()
	loggerValue := reflect.ValueOf(handler).Elem()

	next := loggerValue.FieldByName("Next")
	next = next.Elem()
	next = next.Elem()

	next = next.FieldByName("commonHandler")
	next = next.Elem()

	writerField := next.FieldByName("w")
	writerFieldPtr := unsafe.Pointer(writerField.UnsafeAddr())
	reflect.NewAt(writerField.Type(), writerFieldPtr).Elem().Set(reflect.ValueOf(buffer))

	// Log, and get the output
	logger.Info(msg, vals...)
	output := buffer.String()

	// Slice timestamp to make this deterministic for tests
	firstSpace := strings.Index(output, " ")
	return output[firstSpace+1:]
}
