// logrus.go
package logger

import (
	"context"
	"io"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

type logrusLogger struct {
	logger    *logrus.Logger
	entry     *logrus.Entry
	config    Config
	baseField []Field
}

func newLogrusLogger(cfg Config) Logger {
	logger := logrus.New()

	// Set the output
	logger.SetOutput(cfg.Output)

	// Set the formatter to JSON
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime: "timestamp",
		},
	})

	// Set the log level
	var level logrus.Level
	switch cfg.Level {
	case DebugLevel:
		level = logrus.DebugLevel
	case InfoLevel:
		level = logrus.InfoLevel
	case WarnLevel:
		level = logrus.WarnLevel
	case ErrorLevel:
		level = logrus.ErrorLevel
	case FatalLevel:
		level = logrus.FatalLevel
	default:
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Create a base entry with the service name
	entry := logger.WithField("service", cfg.ServiceName)

	return &logrusLogger{
		logger: logger,
		entry:  entry,
		config: cfg,
	}
}

func (l *logrusLogger) Debug(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, DebugLevel, msg, fields...)
}

func (l *logrusLogger) Info(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, InfoLevel, msg, fields...)
}

func (l *logrusLogger) Warn(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, WarnLevel, msg, fields...)
}

func (l *logrusLogger) Error(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, ErrorLevel, msg, fields...)
}

func (l *logrusLogger) Fatal(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, FatalLevel, msg, fields...)
}

func (l *logrusLogger) WithFields(fields ...Field) Logger {
	newLogger := &logrusLogger{
		logger:    l.logger,
		entry:     l.entry,
		config:    l.config,
		baseField: make([]Field, 0, len(l.baseField)+len(fields)),
	}
	newLogger.baseField = append(newLogger.baseField, l.baseField...)
	newLogger.baseField = append(newLogger.baseField, fields...)
	return newLogger
}

func (l *logrusLogger) SetOutput(w io.Writer) {
	l.logger.SetOutput(w)
	l.config.Output = w
}

func (l *logrusLogger) SetLevel(level LogLevel) {
	var logrusLevel logrus.Level
	switch level {
	case DebugLevel:
		logrusLevel = logrus.DebugLevel
	case InfoLevel:
		logrusLevel = logrus.InfoLevel
	case WarnLevel:
		logrusLevel = logrus.WarnLevel
	case ErrorLevel:
		logrusLevel = logrus.ErrorLevel
	case FatalLevel:
		logrusLevel = logrus.FatalLevel
	default:
		logrusLevel = logrus.InfoLevel
	}
	l.logger.SetLevel(logrusLevel)
	l.config.Level = level
}

func (l *logrusLogger) log(ctx context.Context, level LogLevel, msg string, fields ...Field) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Create a new entry for this log
	entry := l.entry

	// Add trace information if available
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		traceID := span.SpanContext().TraceID().String()
		spanID := span.SpanContext().SpanID().String()
		entry = entry.WithField("trace_id", traceID).WithField("span_id", spanID)
	}

	// Add base fields
	allFields := make([]Field, 0, len(l.baseField)+len(fields))
	allFields = append(allFields, l.baseField...)
	allFields = append(allFields, fields...)

	// Convert fields to logrus fields
	logrusFields := make(logrus.Fields)
	for _, field := range allFields {
		logrusFields[field.Key] = fieldToLogrusValue(field)
	}

	// Create entry with fields
	entry = entry.WithFields(logrusFields)

	switch level {
	case DebugLevel:
		entry.Debug(msg)
	case InfoLevel:
		entry.Info(msg)
	case WarnLevel:
		entry.Warn(msg)
	case ErrorLevel:
		entry.Error(msg)
	case FatalLevel:
		entry.Fatal(msg)
	}
}

func fieldToLogrusValue(field Field) interface{} {
	switch field.Type {
	case StringType:
		return field.String
	case IntType:
		return field.Int
	case BoolType:
		return field.Bool
	case FloatType:
		return field.Float
	case ErrorType:
		return field.Error
	case TimeType:
		return field.Time
	case DurationType:
		return field.Duration
	case ObjectType:
		return field.Interface
	default:
		return "unknown field type"
	}
}
