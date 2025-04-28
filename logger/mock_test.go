package logger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockLogger(t *testing.T) {
	mockLogger := NewMockLogger()
	ctx := context.Background()

	// Test logging at different levels
	mockLogger.Debug(ctx, "debug message", String("key", "value"))
	mockLogger.Info(ctx, "info message", Int("count", 42))
	mockLogger.Warn(ctx, "warn message", Bool("flag", true))
	mockLogger.Error(ctx, "error message", Float("value", 3.14))
	mockLogger.Fatal(ctx, "fatal message", Errors(assert.AnError))

	// Check logs
	assert.Len(t, mockLogger.GetDebugLogs(), 0) // Default level is Info, so Debug shouldn't be logged
	assert.Len(t, mockLogger.GetInfoLogs(), 1)
	assert.Len(t, mockLogger.GetWarnLogs(), 1)
	assert.Len(t, mockLogger.GetErrorLogs(), 1)
	assert.Len(t, mockLogger.GetFatalLogs(), 1)

	// Check log content
	infoLog := mockLogger.GetInfoLogs()[0]
	assert.Equal(t, "info message", infoLog.Msg)
	assert.Len(t, infoLog.Fields, 1)
	assert.Equal(t, "count", infoLog.Fields[0].Key)

	// Change log level
	mockLogger.SetLevel(DebugLevel)
	mockLogger.Debug(ctx, "debug after level change")
	assert.Len(t, mockLogger.GetDebugLogs(), 1)

	// Test with fields
	loggerWithFields := mockLogger.WithFields(String("common", "field"))
	loggerWithFields.Info(ctx, "message with fields")

	// Reset logs
	mockLogger.Reset()
	assert.Len(t, mockLogger.GetDebugLogs(), 0)
	assert.Len(t, mockLogger.GetInfoLogs(), 0)
	assert.Len(t, mockLogger.GetWarnLogs(), 0)
	assert.Len(t, mockLogger.GetErrorLogs(), 0)
	assert.Len(t, mockLogger.GetFatalLogs(), 0)
}
