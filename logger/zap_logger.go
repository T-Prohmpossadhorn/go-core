package logger

import (
	"context"
	"io"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type zapLogger struct {
	logger    *zap.Logger
	config    Config
	baseField []Field
}

func newZapLogger(cfg Config) Logger {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	// Set the log level
	var level zapcore.Level
	switch cfg.Level {
	case DebugLevel:
		level = zapcore.DebugLevel
	case InfoLevel:
		level = zapcore.InfoLevel
	case WarnLevel:
		level = zapcore.WarnLevel
	case ErrorLevel:
		level = zapcore.ErrorLevel
	case FatalLevel:
		level = zapcore.FatalLevel
	default:
		level = zapcore.InfoLevel
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(cfg.Output),
		level,
	)

	logger := zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	)

	// Add the service name
	logger = logger.With(zap.String("service", cfg.ServiceName))

	return &zapLogger{
		logger: logger,
		config: cfg,
	}
}

func (l *zapLogger) Debug(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, DebugLevel, msg, fields...)
}

func (l *zapLogger) Info(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, InfoLevel, msg, fields...)
}

func (l *zapLogger) Warn(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, WarnLevel, msg, fields...)
}

func (l *zapLogger) Error(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, ErrorLevel, msg, fields...)
}

func (l *zapLogger) Fatal(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, FatalLevel, msg, fields...)
}

func (l *zapLogger) WithFields(fields ...Field) Logger {
	newLogger := &zapLogger{
		logger:    l.logger,
		config:    l.config,
		baseField: make([]Field, 0, len(l.baseField)+len(fields)),
	}
	newLogger.baseField = append(newLogger.baseField, l.baseField...)
	newLogger.baseField = append(newLogger.baseField, fields...)
	return newLogger
}

func (l *zapLogger) SetOutput(w io.Writer) {
	// Create a new zap.Logger with the new output
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	var level zapcore.Level
	switch l.config.Level {
	case DebugLevel:
		level = zapcore.DebugLevel
	case InfoLevel:
		level = zapcore.InfoLevel
	case WarnLevel:
		level = zapcore.WarnLevel
	case ErrorLevel:
		level = zapcore.ErrorLevel
	case FatalLevel:
		level = zapcore.FatalLevel
	default:
		level = zapcore.InfoLevel
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(w),
		level,
	)

	l.logger = zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	)
	l.config.Output = w
}

func (l *zapLogger) SetLevel(level LogLevel) {
	// Create a new zap.Logger with the new level
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	var zapLevel zapcore.Level
	switch level {
	case DebugLevel:
		zapLevel = zapcore.DebugLevel
	case InfoLevel:
		zapLevel = zapcore.InfoLevel
	case WarnLevel:
		zapLevel = zapcore.WarnLevel
	case ErrorLevel:
		zapLevel = zapcore.ErrorLevel
	case FatalLevel:
		zapLevel = zapcore.FatalLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(l.config.Output),
		zapLevel,
	)

	l.logger = zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	)
	l.config.Level = level
}

func (l *zapLogger) log(ctx context.Context, level LogLevel, msg string, fields ...Field) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Add trace information if available
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		traceID := span.SpanContext().TraceID().String()
		spanID := span.SpanContext().SpanID().String()
		fields = append(fields, String("trace_id", traceID), String("span_id", spanID))
	}

	// Add base fields
	allFields := make([]Field, 0, len(l.baseField)+len(fields))
	allFields = append(allFields, l.baseField...)
	allFields = append(allFields, fields...)

	// Convert fields to zap fields
	zapFields := make([]zap.Field, 0, len(allFields))
	for _, field := range allFields {
		zapFields = append(zapFields, fieldToZapField(field))
	}

	switch level {
	case DebugLevel:
		l.logger.Debug(msg, zapFields...)
	case InfoLevel:
		l.logger.Info(msg, zapFields...)
	case WarnLevel:
		l.logger.Warn(msg, zapFields...)
	case ErrorLevel:
		l.logger.Error(msg, zapFields...)
	case FatalLevel:
		l.logger.Fatal(msg, zapFields...)
	}
}

func fieldToZapField(field Field) zap.Field {
	switch field.Type {
	case StringType:
		return zap.String(field.Key, field.String)
	case IntType:
		return zap.Int64(field.Key, field.Int)
	case BoolType:
		return zap.Bool(field.Key, field.Bool)
	case FloatType:
		return zap.Float64(field.Key, field.Float)
	case ErrorType:
		return zap.Error(field.Error)
	case TimeType:
		return zap.Time(field.Key, field.Time)
	case DurationType:
		return zap.Duration(field.Key, field.Duration)
	case ObjectType:
		return zap.Any(field.Key, field.Interface)
	default:
		return zap.String(field.Key, "unknown field type")
	}
}
