package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggerConfiguration(t *testing.T) {
	tests := []struct {
		name       string
		loggerType LoggerType
		level      LogLevel
		message    string
		logLevel   LogLevel
		shouldLog  bool
	}{
		{
			name:       "ZapLogger debug message at debug level",
			loggerType: ZapLogger,
			level:      DebugLevel,
			message:    "debug message",
			logLevel:   DebugLevel,
			shouldLog:  true,
		},
		{
			name:       "ZapLogger info message at debug level",
			loggerType: ZapLogger,
			level:      DebugLevel,
			message:    "info message",
			logLevel:   InfoLevel,
			shouldLog:  true,
		},
		{
			name:       "ZapLogger debug message at info level",
			loggerType: ZapLogger,
			level:      InfoLevel,
			message:    "debug message",
			logLevel:   DebugLevel,
			shouldLog:  false,
		},
		{
			name:       "LogrusLogger debug message at debug level",
			loggerType: LogrusLogger,
			level:      DebugLevel,
			message:    "debug message",
			logLevel:   DebugLevel,
			shouldLog:  true,
		},
		{
			name:       "LogrusLogger info message at debug level",
			loggerType: LogrusLogger,
			level:      DebugLevel,
			message:    "info message",
			logLevel:   InfoLevel,
			shouldLog:  true,
		},
		{
			name:       "LogrusLogger debug message at info level",
			loggerType: LogrusLogger,
			level:      InfoLevel,
			message:    "debug message",
			logLevel:   DebugLevel,
			shouldLog:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			cfg := Config{
				Type:        tt.loggerType,
				Level:       tt.level,
				Output:      &buf,
				ServiceName: "test-service",
			}

			logger := New(cfg)

			ctx := context.Background()
			switch tt.logLevel {
			case DebugLevel:
				logger.Debug(ctx, tt.message)
			case InfoLevel:
				logger.Info(ctx, tt.message)
			case WarnLevel:
				logger.Warn(ctx, tt.message)
			case ErrorLevel:
				logger.Error(ctx, tt.message)
			}

			if tt.shouldLog {
				assert.Contains(t, buf.String(), tt.message)
			} else {
				assert.Empty(t, buf.String())
			}
		})
	}
}

func TestLoggerFields(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Type:        ZapLogger,
		Level:       DebugLevel,
		Output:      &buf,
		ServiceName: "test-service",
	}

	logger := New(cfg)
	ctx := context.Background()

	now := time.Now()
	logger.Info(ctx, "test message with fields",
		String("string", "value"),
		Int("int", 123),
		Bool("bool", true),
		Float("float", 123.456),
		Errors(errors.New("test error")),
		Time("time", now),
		Duration("duration", time.Second),
		Any("object", map[string]string{"key": "value"}),
	)

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "test message with fields", logEntry["msg"])
	assert.Equal(t, "value", logEntry["string"])
	assert.Equal(t, float64(123), logEntry["int"])
	assert.Equal(t, true, logEntry["bool"])
	assert.Equal(t, 123.456, logEntry["float"])
	assert.Contains(t, logEntry["error"], "test error")
	assert.NotEmpty(t, logEntry["time"])
	assert.Equal(t, float64(1), logEntry["duration"])
	assert.Equal(t, "value", logEntry["object"].(map[string]interface{})["key"])
	assert.Equal(t, "test-service", logEntry["service"])
}

func TestWithFields(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Type:        ZapLogger,
		Level:       DebugLevel,
		Output:      &buf,
		ServiceName: "test-service",
	}

	logger := New(cfg)
	ctx := context.Background()

	// Create a logger with fields
	loggerWithFields := logger.WithFields(
		String("permanent", "field"),
		Int("count", 42),
	)

	// Log with the logger that has fields
	loggerWithFields.Info(ctx, "test message")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "test message", logEntry["msg"])
	assert.Equal(t, "field", logEntry["permanent"])
	assert.Equal(t, float64(42), logEntry["count"])

	// Reset buffer and log again with additional fields
	buf.Reset()
	loggerWithFields.Info(ctx, "another message", String("additional", "value"))

	err = json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "another message", logEntry["msg"])
	assert.Equal(t, "field", logEntry["permanent"])
	assert.Equal(t, float64(42), logEntry["count"])
	assert.Equal(t, "value", logEntry["additional"])
}

func TestSetLevel(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Type:        ZapLogger,
		Level:       InfoLevel,
		Output:      &buf,
		ServiceName: "test-service",
	}

	logger := New(cfg)
	ctx := context.Background()

	// Debug should not log at Info level
	logger.Debug(ctx, "debug message")
	assert.Empty(t, buf.String())

	// Change level to Debug
	logger.SetLevel(DebugLevel)

	// Now Debug should log
	logger.Debug(ctx, "debug message after level change")
	assert.Contains(t, buf.String(), "debug message after level change")
}

func TestSetOutput(t *testing.T) {
	var buf1 bytes.Buffer
	cfg := Config{
		Type:        ZapLogger,
		Level:       InfoLevel,
		Output:      &buf1,
		ServiceName: "test-service",
	}

	logger := New(cfg)
	ctx := context.Background()

	// Log to first buffer
	logger.Info(ctx, "message to first buffer")
	assert.Contains(t, buf1.String(), "message to first buffer")

	// Change output to second buffer
	var buf2 bytes.Buffer
	logger.SetOutput(&buf2)

	// Log to second buffer
	logger.Info(ctx, "message to second buffer")
	assert.Contains(t, buf2.String(), "message to second buffer")
	assert.NotContains(t, buf1.String(), "message to second buffer")
}

func TestLogrusLogger(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Type:        LogrusLogger,
		Level:       InfoLevel,
		Output:      &buf,
		ServiceName: "test-service",
	}

	logger := New(cfg)
	ctx := context.Background()

	logger.Info(ctx, "test logrus message", String("key", "value"))

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "test logrus message", logEntry["msg"])
	assert.Equal(t, "value", logEntry["key"])
	assert.Equal(t, "test-service", logEntry["service"])
}

func TestDefaultLogger(t *testing.T) {
	// Get the default logger
	logger := GetLogger()
	assert.NotNil(t, logger)

	// Set a custom logger
	var buf bytes.Buffer
	cfg := Config{
		Type:        ZapLogger,
		Level:       InfoLevel,
		Output:      &buf,
		ServiceName: "test-service",
	}
	customLogger := New(cfg)
	SetLogger(customLogger)

	// Use the global functions
	ctx := context.Background()
	Info(ctx, "test global function")
	assert.Contains(t, buf.String(), "test global function")
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		levelStr string
		want     LogLevel
		wantErr  bool
	}{
		{"debug", DebugLevel, false},
		{"info", InfoLevel, false},
		{"warn", WarnLevel, false},
		{"warning", WarnLevel, false},
		{"error", ErrorLevel, false},
		{"fatal", FatalLevel, false},
		{"unknown", InfoLevel, true},
	}

	for _, tt := range tests {
		t.Run(tt.levelStr, func(t *testing.T) {
			got, err := ParseLevel(tt.levelStr)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
