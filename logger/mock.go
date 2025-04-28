package logger

import (
	"context"
	"io"
	"sync"
)

// MockLogger is a mock implementation of the Logger interface for testing
type MockLogger struct {
	mu         sync.Mutex
	debugLogs  []LogEntry
	infoLogs   []LogEntry
	warnLogs   []LogEntry
	errorLogs  []LogEntry
	fatalLogs  []LogEntry
	baseFields []Field
	level      LogLevel
	output     io.Writer
}

// LogEntry represents a log entry for testing
type LogEntry struct {
	Msg    string
	Fields []Field
}

// NewMockLogger creates a new mock logger
func NewMockLogger() *MockLogger {
	return &MockLogger{
		level: InfoLevel,
	}
}

// Debug logs a debug message
func (m *MockLogger) Debug(ctx context.Context, msg string, fields ...Field) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.level <= DebugLevel {
		allFields := make([]Field, 0, len(m.baseFields)+len(fields))
		allFields = append(allFields, m.baseFields...)
		allFields = append(allFields, fields...)

		m.debugLogs = append(m.debugLogs, LogEntry{
			Msg:    msg,
			Fields: allFields,
		})
	}
}

// Info logs an info message
func (m *MockLogger) Info(ctx context.Context, msg string, fields ...Field) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.level <= InfoLevel {
		allFields := make([]Field, 0, len(m.baseFields)+len(fields))
		allFields = append(allFields, m.baseFields...)
		allFields = append(allFields, fields...)

		m.infoLogs = append(m.infoLogs, LogEntry{
			Msg:    msg,
			Fields: allFields,
		})
	}
}

// Warn logs a warning message
func (m *MockLogger) Warn(ctx context.Context, msg string, fields ...Field) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.level <= WarnLevel {
		allFields := make([]Field, 0, len(m.baseFields)+len(fields))
		allFields = append(allFields, m.baseFields...)
		allFields = append(allFields, fields...)

		m.warnLogs = append(m.warnLogs, LogEntry{
			Msg:    msg,
			Fields: allFields,
		})
	}
}

// Error logs an error message
func (m *MockLogger) Error(ctx context.Context, msg string, fields ...Field) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.level <= ErrorLevel {
		allFields := make([]Field, 0, len(m.baseFields)+len(fields))
		allFields = append(allFields, m.baseFields...)
		allFields = append(allFields, fields...)

		m.errorLogs = append(m.errorLogs, LogEntry{
			Msg:    msg,
			Fields: allFields,
		})
	}
}

// Fatal logs a fatal message
func (m *MockLogger) Fatal(ctx context.Context, msg string, fields ...Field) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.level <= FatalLevel {
		allFields := make([]Field, 0, len(m.baseFields)+len(fields))
		allFields = append(allFields, m.baseFields...)
		allFields = append(allFields, fields...)

		m.fatalLogs = append(m.fatalLogs, LogEntry{
			Msg:    msg,
			Fields: allFields,
		})
	}
}

// WithFields returns a logger with the given fields
func (m *MockLogger) WithFields(fields ...Field) Logger {
	m.mu.Lock()
	defer m.mu.Unlock()

	newLogger := &MockLogger{
		level:      m.level,
		output:     m.output,
		baseFields: make([]Field, 0, len(m.baseFields)+len(fields)),
	}
	newLogger.baseFields = append(newLogger.baseFields, m.baseFields...)
	newLogger.baseFields = append(newLogger.baseFields, fields...)
	return newLogger
}

// SetOutput sets the output writer
func (m *MockLogger) SetOutput(w io.Writer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.output = w
}

// SetLevel sets the minimum log level
func (m *MockLogger) SetLevel(level LogLevel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.level = level
}

// GetDebugLogs returns all debug logs
func (m *MockLogger) GetDebugLogs() []LogEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.debugLogs
}

// GetInfoLogs returns all info logs
func (m *MockLogger) GetInfoLogs() []LogEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.infoLogs
}

// GetWarnLogs returns all warning logs
func (m *MockLogger) GetWarnLogs() []LogEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.warnLogs
}

// GetErrorLogs returns all error logs
func (m *MockLogger) GetErrorLogs() []LogEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.errorLogs
}

// GetFatalLogs returns all fatal logs
func (m *MockLogger) GetFatalLogs() []LogEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.fatalLogs
}

// Reset clears all logs
func (m *MockLogger) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.debugLogs = nil
	m.infoLogs = nil
	m.warnLogs = nil
	m.errorLogs = nil
	m.fatalLogs = nil
}
