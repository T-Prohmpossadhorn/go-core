package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
)

// LogLevel represents the severity level of a log message
type LogLevel int

const (
	// Debug level for detailed information
	DebugLevel LogLevel = iota
	// Info level for general information
	InfoLevel
	// Warn level for warnings
	WarnLevel
	// Error level for errors
	ErrorLevel
	// Fatal level for fatal errors
	FatalLevel
)

// LoggerType represents the underlying logging library
type LoggerType int

const (
	// ZapLogger uses the zap logging library
	ZapLogger LoggerType = iota
	// LogrusLogger uses the logrus logging library
	LogrusLogger
)

// Logger interface defines the logging methods
type Logger interface {
	Debug(ctx context.Context, msg string, fields ...Field)
	Info(ctx context.Context, msg string, fields ...Field)
	Warn(ctx context.Context, msg string, fields ...Field)
	Error(ctx context.Context, msg string, fields ...Field)
	Fatal(ctx context.Context, msg string, fields ...Field)
	WithFields(fields ...Field) Logger
	SetOutput(w io.Writer)
	SetLevel(level LogLevel)
}

// Config represents the logger configuration
type Config struct {
	// Type of logger to use
	Type LoggerType
	// Level is the minimum log level to output
	Level LogLevel
	// Output is where logs will be written
	Output io.Writer
	// ServiceName is the name of the service
	ServiceName string
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	return Config{
		Type:        ZapLogger,
		Level:       InfoLevel,
		Output:      os.Stdout,
		ServiceName: "service",
	}
}

var (
	defaultLogger Logger
	once          sync.Once
)

// New creates a new logger with the given configuration
func New(cfg Config) Logger {
	switch cfg.Type {
	case ZapLogger:
		return newZapLogger(cfg)
	case LogrusLogger:
		return newLogrusLogger(cfg)
	default:
		return newZapLogger(cfg)
	}
}

// GetLogger returns the default logger instance
func GetLogger() Logger {
	once.Do(func() {
		defaultLogger = New(DefaultConfig())
	})
	return defaultLogger
}

// SetLogger sets the default logger instance
func SetLogger(logger Logger) {
	defaultLogger = logger
}

// Debug logs a debug message
func Debug(ctx context.Context, msg string, fields ...Field) {
	GetLogger().Debug(ctx, msg, fields...)
}

// Info logs an info message
func Info(ctx context.Context, msg string, fields ...Field) {
	GetLogger().Info(ctx, msg, fields...)
}

// Warn logs a warning message
func Warn(ctx context.Context, msg string, fields ...Field) {
	GetLogger().Warn(ctx, msg, fields...)
}

// Error logs an error message
func Error(ctx context.Context, msg string, fields ...Field) {
	GetLogger().Error(ctx, msg, fields...)
}

// Fatal logs a fatal message and exits
func Fatal(ctx context.Context, msg string, fields ...Field) {
	GetLogger().Fatal(ctx, msg, fields...)
}

// WithFields returns a logger with the given fields
func WithFields(fields ...Field) Logger {
	return GetLogger().WithFields(fields...)
}

// SetOutput sets the output writer for the logger
func SetOutput(w io.Writer) {
	GetLogger().SetOutput(w)
}

// SetLevel sets the minimum log level
func SetLevel(level LogLevel) {
	GetLogger().SetLevel(level)
}

// Convert a LogLevel to a string
func (l LogLevel) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	case FatalLevel:
		return "fatal"
	default:
		return fmt.Sprintf("LogLevel(%d)", l)
	}
}

// ParseLevel parses a string into a LogLevel
func ParseLevel(levelStr string) (LogLevel, error) {
	switch levelStr {
	case "debug":
		return DebugLevel, nil
	case "info":
		return InfoLevel, nil
	case "warn", "warning":
		return WarnLevel, nil
	case "error":
		return ErrorLevel, nil
	case "fatal":
		return FatalLevel, nil
	default:
		return InfoLevel, fmt.Errorf("unknown log level: %s", levelStr)
	}
}
