package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestLogger(t *testing.T) {
	// Create a real OpenTelemetry TracerProvider with a stdout exporter
	ctx := context.Background()
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter)),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			t.Logf("Failed to shutdown TracerProvider: %v", err)
		}
	}()

	tracer := tp.Tracer("test-logger")
	_, span := tracer.Start(ctx, "test-span")
	ctx = oteltrace.ContextWithSpan(ctx, span)
	defer span.End()

	for _, jsonFormat := range []bool{true, false} {
		for _, output := range []string{"console", "file"} {
			t.Run("JSONFormat="+strings.ToLower(fmt.Sprint(jsonFormat))+"_Output="+output, func(t *testing.T) {
				var logString string
				var cleanup func()

				if output == "console" {
					// Redirect stdout to capture logs
					originalStdout := os.Stdout
					r, w, _ := os.Pipe()
					os.Stdout = w

					// Initialize logger with locked syncer
					cfg := LoggerConfig{
						Level:      "debug",
						Output:     "console",
						JSONFormat: jsonFormat,
					}
					err := InitWithConfig(cfg)
					assert.NoError(t, err)

					// Test logging
					performTestLogging(t, ctx)

					// Sync logger before closing pipe
					err = Sync()
					if err != nil {
						t.Logf("Sync error (non-fatal): %v", err)
					}

					// Brief delay to ensure flush
					time.Sleep(10 * time.Millisecond)

					// Capture log output
					w.Close()
					var logBuf bytes.Buffer
					_, err = logBuf.ReadFrom(r)
					assert.NoError(t, err)
					logString = logBuf.String()

					// Restore stdout
					os.Stdout = originalStdout
					cleanup = func() {}
				} else {
					// Create temporary log file
					tmpfile, err := os.CreateTemp("", "test-log*.log")
					assert.NoError(t, err)

					// Initialize logger
					cfg := LoggerConfig{
						Level:      "debug",
						Output:     "file",
						FilePath:   tmpfile.Name(),
						JSONFormat: jsonFormat,
					}
					err = InitWithConfig(cfg)
					assert.NoError(t, err)

					// Test logging
					performTestLogging(t, ctx)

					// Sync logger
					err = Sync()
					assert.NoError(t, err)

					// Read log content
					logContent, err := os.ReadFile(tmpfile.Name())
					assert.NoError(t, err)
					logString = string(logContent)

					// Cleanup
					cleanup = func() { os.Remove(tmpfile.Name()) }
				}
				defer cleanup()

				t.Logf("Log output: %s", logString)

				if jsonFormat {
					// Parse JSON logs
					lines := strings.Split(strings.TrimSpace(logString), "\n")
					var logEntry map[string]interface{}
					for _, line := range lines {
						if strings.Contains(line, "Test message") {
							err := json.Unmarshal([]byte(line), &logEntry)
							assert.NoError(t, err)
							assert.Equal(t, "value", logEntry["string_field"])
							assert.Equal(t, float64(42), logEntry["int_field"])
							assert.Equal(t, 3.14, logEntry["float_field"])
							assert.Equal(t, true, logEntry["bool_field"])
							assert.Equal(t, "test error", logEntry["error"])
							anyField, ok := logEntry["any_field"].([]interface{})
							assert.True(t, ok, "any_field should be a []interface{}")
							assert.Equal(t, []interface{}{"x", "y"}, anyField)
							assert.NotEmpty(t, logEntry["trace_id"], "trace_id should be present")
							assert.NotEmpty(t, logEntry["span_id"], "span_id should be present")
							break
						}
					}
				} else {
					// Check console format logs
					assert.Contains(t, logString, "Test message")
					assert.Contains(t, logString, `"string_field": "value"`)
					assert.Contains(t, logString, `"int_field": 42`)
					assert.Contains(t, logString, `"float_field": 3.14`)
					assert.Contains(t, logString, `"bool_field": true`)
					assert.Contains(t, logString, `"error": "test error"`)
					assert.Contains(t, logString, `"any_field": ["x", "y"]`)
					assert.Contains(t, logString, `"trace_id":`)
					assert.Contains(t, logString, `"span_id":`)
					assert.Contains(t, logString, "debug")
					assert.Contains(t, logString, "Debug message")
					assert.Contains(t, logString, "warn")
					assert.Contains(t, logString, "Warn message")
					assert.Contains(t, logString, "error")
					assert.Contains(t, logString, "Error message")
				}
			})
		}
	}
}

// TestInvalidLogLevel tests initialization with an invalid log level.
func TestInvalidLogLevel(t *testing.T) {
	cfg := LoggerConfig{
		Level:      "invalid",
		Output:     "console",
		JSONFormat: true,
	}
	err := InitWithConfig(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid log level: invalid")
}

// TestUninitializedLogger tests logging without initialization.
func TestUninitializedLogger(t *testing.T) {
	// Ensure globalLogger is nil
	loggerMu.Lock()
	globalLogger = nil
	loggerMu.Unlock()

	ctx := context.Background()
	tests := []struct {
		name string
		log  func(ctx context.Context, msg string, fields ...interface{}) error
	}{
		{"DebugContext", DebugContext},
		{"InfoContext", InfoContext},
		{"WarnContext", WarnContext},
		{"ErrorContext", ErrorContext},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.log(ctx, "Test message", String("key", "value"))
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "logger not initialized")
		})
	}

	// Test Sync
	err := Sync()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "logger not initialized")
}

// TestFileOpenError tests initialization with an invalid file path.
func TestFileOpenError(t *testing.T) {
	cfg := LoggerConfig{
		Level:      "info",
		Output:     "file",
		FilePath:   "/invalid/path/log.log",
		JSONFormat: true,
	}
	err := InitWithConfig(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open log file")
}

// TestEmptyFilePath tests initialization with an empty file path for file output.
func TestEmptyFilePath(t *testing.T) {
	cfg := LoggerConfig{
		Level:      "info",
		Output:     "file",
		FilePath:   "",
		JSONFormat: true,
	}
	err := InitWithConfig(cfg)
	assert.NoError(t, err) // Should fall back to stdout
}

// TestDefaultInit tests the default Init function.
func TestDefaultInit(t *testing.T) {
	err := Init()
	assert.NoError(t, err)

	// Redirect stdout to capture logs
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Log a message
	err = Info("Default init test", String("key", "value"))
	assert.NoError(t, err)

	// Sync logger
	err = Sync()
	if err != nil {
		t.Logf("Sync error (non-fatal): %v", err)
	}

	// Brief delay to ensure flush
	time.Sleep(10 * time.Millisecond)

	// Capture log output
	w.Close()
	var logBuf bytes.Buffer
	_, err = logBuf.ReadFrom(r)
	assert.NoError(t, err)
	logString := logBuf.String()

	// Restore stdout
	os.Stdout = originalStdout

	// Verify JSON format (default)
	var logEntry map[string]interface{}
	lines := strings.Split(strings.TrimSpace(logString), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Default init test") {
			err := json.Unmarshal([]byte(line), &logEntry)
			assert.NoError(t, err)
			assert.Equal(t, "value", logEntry["key"])
			break
		}
	}
}

// TestConcurrentLogging tests logging from multiple goroutines.
func TestConcurrentLogging(t *testing.T) {
	// Create a pipe to capture logs
	r, w, err := os.Pipe()
	assert.NoError(t, err)

	// Configure logger to write directly to the pipe
	var level zapcore.Level = zapcore.DebugLevel
	syncer := zapcore.Lock(zapcore.AddSync(w)) // Thread-safe syncer
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
	encoder := zapcore.NewJSONEncoder(encoderConfig)
	core := zapcore.NewCore(encoder, syncer, level)
	logger := zap.New(core, zap.AddCaller())

	// Set global logger
	loggerMu.Lock()
	globalLogger = logger
	loggerMu.Unlock()

	// Read from pipe concurrently to avoid blocking
	var logBuf bytes.Buffer
	var wgRead sync.WaitGroup
	wgRead.Add(1)
	go func() {
		defer wgRead.Done()
		_, err := logBuf.ReadFrom(r)
		assert.NoError(t, err)
		r.Close()
	}()

	var wg sync.WaitGroup
	numGoroutines := 10
	logMessages := make([]string, numGoroutines)

	// Start goroutines to log concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		msg := fmt.Sprintf("Concurrent message %d", i)
		logMessages[i] = msg
		go func(id int) {
			defer wg.Done()
			err := Info(fmt.Sprintf("Concurrent message %d", id), Int("goroutine", id))
			assert.NoError(t, err)
		}(i)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Sync logger
	err = Sync()
	if err != nil {
		t.Logf("Sync error (non-fatal): %v", err)
	}

	// Close write end of pipe after a brief delay
	time.Sleep(50 * time.Millisecond) // Increased delay to ensure flush
	w.Close()

	// Wait for reading to complete
	wgRead.Wait()

	// Get log output
	logString := logBuf.String()
	t.Logf("Captured log output: %s", logString)

	// Verify all messages are present
	for _, msg := range logMessages {
		assert.Contains(t, logString, msg, "Log output should contain message: %s", msg)
	}
}

// TestFatalLogging tests Fatal and FatalContext in a subprocess.
func TestFatalLogging(t *testing.T) {
	if os.Getenv("TEST_FATAL_SUBPROCESS") == "1" {
		// Subprocess logic
		err := Init()
		assert.NoError(t, err)

		ctx := context.Background()
		tracer := sdktrace.NewTracerProvider().Tracer("test-logger")
		_, span := tracer.Start(ctx, "fatal-span")
		ctx = oteltrace.ContextWithSpan(ctx, span)

		// Test Fatal
		_ = Info("Before fatal", String("key", "value"))
		_ = Fatal("Fatal message", String("fatal_key", "fatal_value"))

		// Unreachable
		t.Fatal("Should have exited")
		return
	}

	if os.Getenv("TEST_FATAL_CONTEXT_SUBPROCESS") == "1" {
		// Subprocess logic for FatalContext
		err := Init()
		assert.NoError(t, err)

		ctx := context.Background()
		tracer := sdktrace.NewTracerProvider().Tracer("test-logger")
		_, span := tracer.Start(ctx, "fatal-span")
		ctx = oteltrace.ContextWithSpan(ctx, span)

		// Test FatalContext
		_ = Info("Before fatal context", String("key", "value"))
		_ = FatalContext(ctx, "Fatal context message", String("fatal_key", "fatal_context_value"))

		// Unreachable
		t.Fatal("Should have exited")
		return
	}

	// Main test logic
	t.Run("Fatal", func(t *testing.T) {
		// Create a temporary file to capture output
		tmpfile, err := os.CreateTemp("", "fatal-log*.log")
		assert.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		// Run subprocess
		cmd := exec.Command(os.Args[0], "-test.run=TestFatalLogging")
		cmd.Env = append(os.Environ(), "TEST_FATAL_SUBPROCESS=1")
		cmd.Stdout = tmpfile
		cmd.Stderr = tmpfile
		err = cmd.Run()

		// Expect non-zero exit code due to os.Exit
		if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() == 0 {
			t.Fatalf("Expected non-zero exit code, got: %v", err)
		}

		// Read log output
		logContent, err := os.ReadFile(tmpfile.Name())
		assert.NoError(t, err)
		logString := string(logContent)

		// Verify logs
		var foundFatal bool
		lines := strings.Split(strings.TrimSpace(logString), "\n")
		for _, line := range lines {
			var logEntry map[string]interface{}
			if err := json.Unmarshal([]byte(line), &logEntry); err == nil {
				if logEntry["msg"] == "Fatal message" {
					assert.Equal(t, "fatal_value", logEntry["fatal_key"])
					foundFatal = true
				}
			}
		}
		assert.True(t, foundFatal, "Fatal message not found in logs")
	})

	t.Run("FatalContext", func(t *testing.T) {
		// Create a temporary file to capture output
		tmpfile, err := os.CreateTemp("", "fatal-context-log*.log")
		assert.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		// Run subprocess
		cmd := exec.Command(os.Args[0], "-test.run=TestFatalLogging")
		cmd.Env = append(os.Environ(), "TEST_FATAL_CONTEXT_SUBPROCESS=1")
		cmd.Stdout = tmpfile
		cmd.Stderr = tmpfile
		err = cmd.Run()

		// Expect non-zero exit code due to os.Exit
		if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() == 0 {
			t.Fatalf("Expected non-zero exit code, got: %v", err)
		}

		// Read log output
		logContent, err := os.ReadFile(tmpfile.Name())
		assert.NoError(t, err)
		logString := string(logContent)

		// Verify logs
		var foundFatalContext bool
		lines := strings.Split(strings.TrimSpace(logString), "\n")
		for _, line := range lines {
			var logEntry map[string]interface{}
			if err := json.Unmarshal([]byte(line), &logEntry); err == nil {
				if logEntry["msg"] == "Fatal context message" {
					assert.Equal(t, "fatal_context_value", logEntry["fatal_key"])
					assert.NotEmpty(t, logEntry["trace_id"], "trace_id should be present")
					assert.NotEmpty(t, logEntry["span_id"], "span_id should be present")
					foundFatalContext = true
				}
			}
		}
		assert.True(t, foundFatalContext, "Fatal context message not found in logs")
	})
}

// TestEmptyFields tests logging with no fields.
func TestEmptyFields(t *testing.T) {
	cfg := LoggerConfig{
		Level:      "info",
		Output:     "console",
		JSONFormat: true,
	}
	err := InitWithConfig(cfg)
	assert.NoError(t, err)

	// Redirect stdout to capture logs
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Log without fields
	err = Info("No fields test")
	assert.NoError(t, err)

	// Sync logger
	err = Sync()
	if err != nil {
		t.Logf("Sync error (non-fatal): %v", err)
	}

	// Brief delay to ensure flush
	time.Sleep(10 * time.Millisecond)

	// Capture log output
	w.Close()
	var logBuf bytes.Buffer
	_, err = logBuf.ReadFrom(r)
	assert.NoError(t, err)
	logString := logBuf.String()

	// Restore stdout
	os.Stdout = originalStdout

	// Verify JSON format
	var logEntry map[string]interface{}
	lines := strings.Split(strings.TrimSpace(logString), "\n")
	for _, line := range lines {
		if strings.Contains(line, "No fields test") {
			err := json.Unmarshal([]byte(line), &logEntry)
			assert.NoError(t, err)
			assert.Equal(t, "No fields test", logEntry["msg"])
			break
		}
	}
}

// TestInvalidFieldType tests logging with an invalid field type.
func TestInvalidFieldType(t *testing.T) {
	cfg := LoggerConfig{
		Level:      "info",
		Output:     "console",
		JSONFormat: true,
	}
	err := InitWithConfig(cfg)
	assert.NoError(t, err)

	// Redirect stdout to capture logs
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Log with invalid field type
	err = Info("Invalid field test", Field{Key: "invalid", Value: "value", Type: "unknown"})
	assert.NoError(t, err)

	// Sync logger
	err = Sync()
	if err != nil {
		t.Logf("Sync error (non-fatal): %v", err)
	}

	// Brief delay to ensure flush
	time.Sleep(10 * time.Millisecond)

	// Capture log output
	w.Close()
	var logBuf bytes.Buffer
	_, err = logBuf.ReadFrom(r)
	assert.NoError(t, err)
	logString := logBuf.String()

	// Restore stdout
	os.Stdout = originalStdout

	// Verify the field is logged as Any
	var logEntry map[string]interface{}
	lines := strings.Split(strings.TrimSpace(logString), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Invalid field test") {
			err := json.Unmarshal([]byte(line), &logEntry)
			assert.NoError(t, err)
			assert.Equal(t, "value", logEntry["invalid"])
			break
		}
	}
}

// TestNonFieldInput tests logging with a non-Field type.
func TestNonFieldInput(t *testing.T) {
	cfg := LoggerConfig{
		Level:      "info",
		Output:     "console",
		JSONFormat: true,
	}
	err := InitWithConfig(cfg)
	assert.NoError(t, err)

	// Redirect stdout to capture logs
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Log with non-Field type
	err = Info("Non-field test", "not_a_field")
	assert.NoError(t, err)

	// Sync logger
	err = Sync()
	if err != nil {
		t.Logf("Sync error (non-fatal): %v", err)
	}

	// Brief delay to ensure flush
	time.Sleep(10 * time.Millisecond)

	// Capture log output
	w.Close()
	var logBuf bytes.Buffer
	_, err = logBuf.ReadFrom(r)
	assert.NoError(t, err)
	logString := logBuf.String()

	// Restore stdout
	os.Stdout = originalStdout

	// Verify the message is logged without additional fields
	var logEntry map[string]interface{}
	lines := strings.Split(strings.TrimSpace(logString), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Non-field test") {
			err := json.Unmarshal([]byte(line), &logEntry)
			assert.NoError(t, err)
			assert.Equal(t, "Non-field test", logEntry["msg"])
			// No additional fields should be present
			assert.NotContains(t, logEntry, "not_a_field")
			break
		}
	}
}

// TestTypeAssertionFailure tests logging with mismatched field type and value.
func TestTypeAssertionFailure(t *testing.T) {
	cfg := LoggerConfig{
		Level:      "info",
		Output:     "console",
		JSONFormat: true,
	}
	err := InitWithConfig(cfg)
	assert.NoError(t, err)

	// Redirect stdout to capture logs
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Log with mismatched field types
	err = Info("Type assertion test",
		Field{Key: "int_field", Value: "not_an_int", Type: "int"},
		Field{Key: "float_field", Value: "not_a_float", Type: "float"},
		Field{Key: "bool_field", Value: "not_a_bool", Type: "bool"},
		Field{Key: "error_field", Value: "not_an_error", Type: "error"},
	)
	assert.NoError(t, err)

	// Sync logger
	err = Sync()
	if err != nil {
		t.Logf("Sync error (non-fatal): %v", err)
	}

	// Brief delay to ensure flush
	time.Sleep(10 * time.Millisecond)

	// Capture log output
	w.Close()
	var logBuf bytes.Buffer
	_, err = logBuf.ReadFrom(r)
	assert.NoError(t, err)
	logString := logBuf.String()

	// Restore stdout
	os.Stdout = originalStdout

	// Verify fields are logged as Any due to type assertion failures
	var logEntry map[string]interface{}
	lines := strings.Split(strings.TrimSpace(logString), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Type assertion test") {
			err := json.Unmarshal([]byte(line), &logEntry)
			assert.NoError(t, err)
			assert.Equal(t, "not_an_int", logEntry["int_field"])
			assert.Equal(t, "not_a_float", logEntry["float_field"])
			assert.Equal(t, "not_a_bool", logEntry["bool_field"])
			assert.Equal(t, "not_an_error", logEntry["error_field"])
			break
		}
	}
}

// TestNilErrorField tests logging with a nil error field.
func TestNilErrorField(t *testing.T) {
	cfg := LoggerConfig{
		Level:      "info",
		Output:     "console",
		JSONFormat: true,
	}
	err := InitWithConfig(cfg)
	assert.NoError(t, err)

	// Redirect stdout to capture logs
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Log with nil error
	err = Info("Nil error test", ErrField(nil))
	assert.NoError(t, err)

	// Sync logger
	err = Sync()
	if err != nil {
		t.Logf("Sync error (non-fatal): %v", err)
	}

	// Brief delay to ensure flush
	time.Sleep(10 * time.Millisecond)

	// Capture log output
	w.Close()
	var logBuf bytes.Buffer
	_, err = logBuf.ReadFrom(r)
	assert.NoError(t, err)
	logString := logBuf.String()

	// Restore stdout
	os.Stdout = originalStdout

	// Verify no error field in output
	var logEntry map[string]interface{}
	lines := strings.Split(strings.TrimSpace(logString), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Nil error test") {
			err := json.Unmarshal([]byte(line), &logEntry)
			assert.NoError(t, err)
			_, hasError := logEntry["error"]
			assert.False(t, hasError, "Error field should not be present")
			break
		}
	}
}

// TestInvalidSpanContext tests logging with an invalid span context.
func TestInvalidSpanContext(t *testing.T) {
	cfg := LoggerConfig{
		Level:      "info",
		Output:     "console",
		JSONFormat: true,
	}
	err := InitWithConfig(cfg)
	assert.NoError(t, err)

	// Redirect stdout to capture logs
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Log with empty context (no span)
	ctx := context.Background()
	err = InfoContext(ctx, "Invalid span test", String("key", "value"))
	assert.NoError(t, err)

	// Sync logger
	err = Sync()
	if err != nil {
		t.Logf("Sync error (non-fatal): %v", err)
	}

	// Brief delay to ensure flush
	time.Sleep(10 * time.Millisecond)

	// Capture log output
	w.Close()
	var logBuf bytes.Buffer
	_, err = logBuf.ReadFrom(r)
	assert.NoError(t, err)
	logString := logBuf.String()

	// Restore stdout
	os.Stdout = originalStdout

	// Verify no trace_id or span_id in output
	var logEntry map[string]interface{}
	lines := strings.Split(strings.TrimSpace(logString), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Invalid span test") {
			err := json.Unmarshal([]byte(line), &logEntry)
			assert.NoError(t, err)
			_, hasTraceID := logEntry["trace_id"]
			_, hasSpanID := logEntry["span_id"]
			assert.False(t, hasTraceID, "trace_id should not be present")
			assert.False(t, hasSpanID, "span_id should not be present")
			assert.Equal(t, "value", logEntry["key"])
			break
		}
	}
}

// performTestLogging executes a set of logging operations for testing.
func performTestLogging(t *testing.T, ctx context.Context) {
	err := InfoContext(ctx, "Test message",
		String("string_field", "value"),
		Int("int_field", 42),
		Float("float_field", 3.14),
		Bool("bool_field", true),
		ErrField(errors.New("test error")),
		Any("any_field", []string{"x", "y"}),
	)
	assert.NoError(t, err)

	err = DebugContext(ctx, "Debug message", String("debug_field", "debug"))
	assert.NoError(t, err)
	err = InfoContext(ctx, "Info message", String("info_field", "info"))
	assert.NoError(t, err)
	err = WarnContext(ctx, "Warn message", String("warn_field", "warn"))
	assert.NoError(t, err)
	err = ErrorContext(ctx, "Error message", String("error_field", "error"))
	assert.NoError(t, err)

	err = Debug("Debug message", String("debug_field", "debug"))
	assert.NoError(t, err)
	err = Info("Info message", String("info_field", "info"))
	assert.NoError(t, err)
	err = Warn("Warn message", String("warn_field", "warn"))
	assert.NoError(t, err)
	err = Error("Error message", String("error_field", "error"))
	assert.NoError(t, err)
}
