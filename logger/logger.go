package logger

import (
	"context"
	"fmt"
	"os"
	"sync"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Field represents a key-value pair for logging.
type Field struct {
	Key   string
	Value interface{}
	Type  string
}

// String creates a string field.
func String(key, value string) interface{} {
	return Field{Key: key, Value: value, Type: "string"}
}

// Int creates an integer field.
func Int(key string, value int) interface{} {
	return Field{Key: key, Value: value, Type: "int"}
}

// Float creates a float64 field.
func Float(key string, value float64) interface{} {
	return Field{Key: key, Value: value, Type: "float"}
}

// Bool creates a boolean field.
func Bool(key string, value bool) interface{} {
	return Field{Key: key, Value: value, Type: "bool"}
}

// ErrField creates an error field.
func ErrField(err error) interface{} {
	return Field{Key: "error", Value: err, Type: "error"}
}

// Any creates a field for any data type.
func Any(key string, value interface{}) interface{} {
	return Field{Key: key, Value: value, Type: "any"}
}

// LoggerConfig defines the configuration for the logger.
type LoggerConfig struct {
	Level      string `mapstructure:"level" default:"info"`
	Output     string `mapstructure:"output" default:"console"`
	FilePath   string `mapstructure:"file_path" default:""`
	JSONFormat bool   `mapstructure:"json_format" default:"true"`
}

var (
	globalLogger *zap.Logger
	loggerMu     sync.RWMutex
)

// Init initializes the global logger with default settings (info level, console output, JSON format).
func Init() error {
	return InitWithConfig(LoggerConfig{
		Level:      "info",
		Output:     "console",
		JSONFormat: true,
	})
}

// InitWithConfig initializes the global logger with the provided configuration.
func InitWithConfig(cfg LoggerConfig) error {
	loggerMu.Lock()
	defer loggerMu.Unlock()

	var level zapcore.Level
	switch cfg.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	case "fatal":
		level = zapcore.FatalLevel
	default:
		return fmt.Errorf("invalid log level: %s", cfg.Level)
	}

	var core zapcore.Core
	var syncer zapcore.WriteSyncer

	if cfg.Output == "file" && cfg.FilePath != "" {
		file, err := os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("failed to open log file %s: %w", cfg.FilePath, err)
		}
		syncer = zapcore.AddSync(file)
	} else {
		syncer = zapcore.AddSync(os.Stdout)
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		MessageKey:     "msg",
		CallerKey:      "caller",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	if cfg.JSONFormat {
		encoder := zapcore.NewJSONEncoder(encoderConfig)
		core = zapcore.NewCore(encoder, syncer, level)
	} else {
		encoder := zapcore.NewConsoleEncoder(encoderConfig)
		core = zapcore.NewCore(encoder, syncer, level)
	}

	globalLogger = zap.New(core, zap.AddCaller())
	return nil
}

// Sync flushes any buffered log entries.
func Sync() error {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	if globalLogger == nil {
		return fmt.Errorf("logger not initialized")
	}
	return globalLogger.Sync()
}

// Debug logs a debug-level message with default context.
func Debug(msg string, fields ...interface{}) error {
	return DebugContext(context.Background(), msg, fields...)
}

// Info logs an info-level message with default context.
func Info(msg string, fields ...interface{}) error {
	return InfoContext(context.Background(), msg, fields...)
}

// Warn logs a warn-level message with default context.
func Warn(msg string, fields ...interface{}) error {
	return WarnContext(context.Background(), msg, fields...)
}

// Error logs an error-level message with default context.
func Error(msg string, fields ...interface{}) error {
	return ErrorContext(context.Background(), msg, fields...)
}

// Fatal logs a fatal-level message with default context and exits.
func Fatal(msg string, fields ...interface{}) error {
	return FatalContext(context.Background(), msg, fields...)
}

// DebugContext logs a debug-level message with context and fields.
func DebugContext(ctx context.Context, msg string, fields ...interface{}) error {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	if globalLogger == nil {
		return fmt.Errorf("logger not initialized")
	}
	zapFields := extractTraceFields(ctx)
	for _, f := range fields {
		if field, ok := f.(Field); ok {
			zapFields = append(zapFields, fieldToZap(field))
		}
	}
	globalLogger.Debug(msg, zapFields...)
	return nil
}

// InfoContext logs an info-level message with context and fields.
func InfoContext(ctx context.Context, msg string, fields ...interface{}) error {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	if globalLogger == nil {
		return fmt.Errorf("logger not initialized")
	}
	zapFields := extractTraceFields(ctx)
	for _, f := range fields {
		if field, ok := f.(Field); ok {
			zapFields = append(zapFields, fieldToZap(field))
		}
	}
	globalLogger.Info(msg, zapFields...)
	return nil
}

// WarnContext logs a warn-level message with context and fields.
func WarnContext(ctx context.Context, msg string, fields ...interface{}) error {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	if globalLogger == nil {
		return fmt.Errorf("logger not initialized")
	}
	zapFields := extractTraceFields(ctx)
	for _, f := range fields {
		if field, ok := f.(Field); ok {
			zapFields = append(zapFields, fieldToZap(field))
		}
	}
	globalLogger.Warn(msg, zapFields...)
	return nil
}

// ErrorContext logs an error-level message with context and fields.
func ErrorContext(ctx context.Context, msg string, fields ...interface{}) error {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	if globalLogger == nil {
		return fmt.Errorf("logger not initialized")
	}
	zapFields := extractTraceFields(ctx)
	for _, f := range fields {
		if field, ok := f.(Field); ok {
			zapFields = append(zapFields, fieldToZap(field))
		}
	}
	globalLogger.Error(msg, zapFields...)
	return nil
}

// FatalContext logs a fatal-level message with context and fields, then exits.
func FatalContext(ctx context.Context, msg string, fields ...interface{}) error {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	if globalLogger == nil {
		return fmt.Errorf("logger not initialized")
	}
	zapFields := extractTraceFields(ctx)
	for _, f := range fields {
		if field, ok := f.(Field); ok {
			zapFields = append(zapFields, fieldToZap(field))
		}
	}
	globalLogger.Fatal(msg, zapFields...)
	return nil
}

// fieldToZap converts a Field to a zap.Field.
func fieldToZap(field Field) zap.Field {
	switch field.Type {
	case "string":
		return zap.String(field.Key, fmt.Sprint(field.Value))
	case "int":
		if v, ok := field.Value.(int); ok {
			return zap.Int(field.Key, v)
		}
	case "float":
		if v, ok := field.Value.(float64); ok {
			return zap.Float64(field.Key, v)
		}
	case "bool":
		if v, ok := field.Value.(bool); ok {
			return zap.Bool(field.Key, v)
		}
	case "error":
		if err, ok := field.Value.(error); ok && err != nil {
			return zap.Error(err)
		}
	case "any":
		return zap.Any(field.Key, field.Value)
	}
	return zap.Any(field.Key, field.Value)
}

// extractTraceFields extracts OpenTelemetry trace fields from the context.
func extractTraceFields(ctx context.Context) []zap.Field {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return nil
	}
	return []zap.Field{
		zap.String("trace_id", span.SpanContext().TraceID().String()),
		zap.String("span_id", span.SpanContext().SpanID().String()),
	}
}
