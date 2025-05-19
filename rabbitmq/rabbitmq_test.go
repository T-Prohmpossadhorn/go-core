package rabbitmq

import (
	"bytes"
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/T-Prohmpossadhorn/go-core/config"
	"github.com/T-Prohmpossadhorn/go-core/logger"
	"github.com/T-Prohmpossadhorn/go-core/otel"
	"github.com/stretchr/testify/require"
)

// syncWriter is a thread-safe writer for capturing logs
type syncWriter struct {
	buf *bytes.Buffer
	mu  sync.Mutex
}

func (sw *syncWriter) Write(p []byte) (n int, err error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.buf.Write(p)
}

func setupLogger(t *testing.T) (*syncWriter, *os.File, func()) {
	var logBuf bytes.Buffer
	logWriter := &syncWriter{buf: &logBuf}
	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	os.Stdout = w
	go func() {
		_, _ = logBuf.ReadFrom(r)
		time.Sleep(200 * time.Millisecond)
		r.Close()
	}()
	err = logger.InitWithConfig(logger.LoggerConfig{
		Level:      "info",
		Output:     "console",
		JSONFormat: true,
	})
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	return logWriter, w, func() {
		logger.Sync()
		time.Sleep(200 * time.Millisecond)
		w.Close()
		os.Stdout = originalStdout
	}
}

func getLogs(writer *syncWriter) string {
	logger.Sync()
	time.Sleep(200 * time.Millisecond)
	writer.mu.Lock()
	defer writer.mu.Unlock()
	return writer.buf.String()
}

func resetLogs(writer *syncWriter) {
	logger.Sync()
	time.Sleep(200 * time.Millisecond)
	writer.mu.Lock()
	defer writer.mu.Unlock()
	writer.buf.Reset()
}

func newRabbitMQForTest(t *testing.T) *RabbitMQ {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = "amqp://guest:guest@localhost:5672/"
	}
	cfg, err := config.New(config.WithDefault(map[string]interface{}{
		"rabbitmq_url": url,
	}))
	require.NoError(t, err)
	rmq, err := New(cfg)
	if err != nil {
		t.Skipf("RabbitMQ not available: %v", err)
	}
	return rmq
}

func TestPublishConsume(t *testing.T) {
	rmq := newRabbitMQForTest(t)
	defer rmq.Close()

	ctx := context.Background()
	msgs, err := rmq.Consume(ctx, "q")
	require.NoError(t, err)
	require.NoError(t, rmq.Publish(ctx, "q", []byte("hi")))
	select {
	case msg := <-msgs:
		require.Equal(t, []byte("hi"), msg)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestPublishCanceled(t *testing.T) {
	rmq := newRabbitMQForTest(t)
	defer rmq.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := rmq.Publish(ctx, "q", []byte("hi"))
	require.Error(t, err)
}

func TestPublishTracing(t *testing.T) {
	logWriter, _, cleanup := setupLogger(t)
	defer cleanup()
	resetLogs(logWriter)

	cfg, err := config.New(config.WithDefault(map[string]interface{}{
		"otel_enabled": true,
	}))
	require.NoError(t, err)

	os.Setenv("OTEL_TEST_MOCK_EXPORTER", "true")
	defer os.Unsetenv("OTEL_TEST_MOCK_EXPORTER")
	require.NoError(t, otel.Init(cfg))
	defer otel.Shutdown(context.Background())

	rmq := newRabbitMQForTest(t)
	defer rmq.Close()

	ctx := context.Background()
	require.NoError(t, rmq.Publish(ctx, "q", []byte("hi")))

	logs := getLogs(logWriter)
	require.Contains(t, logs, "\"trace_id\"")
	require.Contains(t, logs, "\"span_id\"")
}
