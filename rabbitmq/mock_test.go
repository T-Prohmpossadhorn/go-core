package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/T-Prohmpossadhorn/go-core/config"
	"github.com/T-Prohmpossadhorn/go-core/otel"
	"github.com/stretchr/testify/require"
)

type mockChannel struct {
	published  []amqp.Publishing
	consumeCh  chan amqp.Delivery
	closed     bool
	declareErr error
	consumeErr error
	publishErr error
}

func (m *mockChannel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	return amqp.Queue{Name: name}, m.declareErr
}

func (m *mockChannel) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	if m.publishErr != nil {
		return m.publishErr
	}
	m.published = append(m.published, msg)
	return nil
}

func (m *mockChannel) ConsumeWithContext(ctx context.Context, queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	if m.consumeErr != nil {
		return nil, m.consumeErr
	}
	return m.consumeCh, nil
}

func (m *mockChannel) Close() error { m.closed = true; return nil }

type mockConn struct {
	ch     *mockChannel
	closed bool
}

type errConn struct{}

func (e *errConn) Channel() (amqpChannel, error) { return nil, fmt.Errorf("chan") }
func (e *errConn) Close() error                  { return nil }

func (c *mockConn) Channel() (amqpChannel, error) { return c.ch, nil }
func (c *mockConn) Close() error                  { c.closed = true; return nil }

func TestRabbitMQPublishConsumeMock(t *testing.T) {
	ch := &mockChannel{consumeCh: make(chan amqp.Delivery, 1)}
	ch.consumeCh <- amqp.Delivery{Body: []byte("consumed")}
	close(ch.consumeCh)

	origDial := dialFunc
	dialFunc = func(string) (amqpConn, error) { return &mockConn{ch: ch}, nil }
	defer func() { dialFunc = origDial }()

	cfg, _ := config.New(config.WithDefault(map[string]interface{}{}))
	rmq, err := New(cfg)
	require.NoError(t, err)

	out, err := rmq.Consume(context.Background(), "q1")
	require.NoError(t, err)

	require.NoError(t, rmq.Publish(context.Background(), "q1", []byte("hello")))
	require.Equal(t, []byte("hello"), ch.published[0].Body)

	msg := <-out
	require.Equal(t, []byte("consumed"), msg)
}

func TestRabbitMQPublishConsumeJSONMock(t *testing.T) {
	type msg struct {
		Name string `json:"name"`
	}

	ch := &mockChannel{consumeCh: make(chan amqp.Delivery, 1)}
	b, _ := json.Marshal(msg{Name: "consumed"})
	ch.consumeCh <- amqp.Delivery{Body: b}
	close(ch.consumeCh)

	origDial := dialFunc
	dialFunc = func(string) (amqpConn, error) { return &mockConn{ch: ch}, nil }
	defer func() { dialFunc = origDial }()

	cfg, _ := config.New(config.WithDefault(map[string]interface{}{}))
	rmq, err := New(cfg)
	require.NoError(t, err)

	out, err := ConsumeJSON[msg](context.Background(), rmq, "q1")
	require.NoError(t, err)

	require.NoError(t, PublishJSON(context.Background(), rmq, "q1", msg{Name: "hello"}))
	require.Len(t, ch.published, 1)
	var sent msg
	_ = json.Unmarshal(ch.published[0].Body, &sent)
	require.Equal(t, "hello", sent.Name)

	m := <-out
	require.Equal(t, "consumed", m.Name)
}

func TestRabbitMQCloseMock(t *testing.T) {
	ch := &mockChannel{consumeCh: make(chan amqp.Delivery)}
	conn := &mockConn{ch: ch}
	origDial := dialFunc
	dialFunc = func(string) (amqpConn, error) { return conn, nil }
	defer func() { dialFunc = origDial }()

	cfg, _ := config.New(config.WithDefault(map[string]interface{}{}))
	rmq, err := New(cfg)
	require.NoError(t, err)

	_, err = rmq.Consume(context.Background(), "q")
	require.NoError(t, err)
	require.NoError(t, rmq.Publish(context.Background(), "q", []byte("x")))

	require.NoError(t, rmq.Close())
	require.True(t, ch.closed)
	require.True(t, conn.closed)
}

func TestRabbitMQPublishCanceledMock(t *testing.T) {
	ch := &mockChannel{consumeCh: make(chan amqp.Delivery)}
	origDial := dialFunc
	dialFunc = func(string) (amqpConn, error) { return &mockConn{ch: ch}, nil }
	defer func() { dialFunc = origDial }()

	cfg, _ := config.New(config.WithDefault(map[string]interface{}{}))
	rmq, err := New(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = rmq.Publish(ctx, "q", []byte("x"))
	require.Error(t, err)
}

func TestRabbitMQPublishTracingMock(t *testing.T) {
	ch := &mockChannel{consumeCh: make(chan amqp.Delivery)}
	origDial := dialFunc
	dialFunc = func(string) (amqpConn, error) { return &mockConn{ch: ch}, nil }
	defer func() { dialFunc = origDial }()

	logWriter, _, cleanup := setupLogger(t)
	defer cleanup()
	resetLogs(logWriter)

	cfg, _ := config.New(config.WithDefault(map[string]interface{}{
		"otel_enabled": true,
	}))

	os.Setenv("OTEL_TEST_MOCK_EXPORTER", "true")
	defer os.Unsetenv("OTEL_TEST_MOCK_EXPORTER")
	require.NoError(t, otel.Init(cfg))
	defer otel.Shutdown(context.Background())

	rmq, err := New(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, rmq.Publish(ctx, "q", []byte("hi")))

	logs := getLogs(logWriter)
	require.Contains(t, logs, "\"trace_id\"")
	require.Contains(t, logs, "\"span_id\"")
}

func TestRabbitMQConsumeErrorMock(t *testing.T) {
	ch := &mockChannel{declareErr: fmt.Errorf("boom")}
	origDial := dialFunc
	dialFunc = func(string) (amqpConn, error) { return &mockConn{ch: ch}, nil }
	defer func() { dialFunc = origDial }()

	cfg, _ := config.New(config.WithDefault(map[string]interface{}{}))
	rmq, err := New(cfg)
	require.NoError(t, err)

	_, err = rmq.Consume(context.Background(), "q")
	require.Error(t, err)
}

func TestRabbitMQPublishErrorMock(t *testing.T) {
	ch := &mockChannel{publishErr: fmt.Errorf("boom")}
	origDial := dialFunc
	dialFunc = func(string) (amqpConn, error) { return &mockConn{ch: ch}, nil }
	defer func() { dialFunc = origDial }()

	cfg, _ := config.New(config.WithDefault(map[string]interface{}{}))
	rmq, err := New(cfg)
	require.NoError(t, err)

	err = rmq.Publish(context.Background(), "q", []byte("x"))
	require.Error(t, err)
}

func TestRabbitMQNewDialError(t *testing.T) {
	origDial := dialFunc
	dialFunc = func(string) (amqpConn, error) { return nil, fmt.Errorf("dial") }
	defer func() { dialFunc = origDial }()

	cfg, _ := config.New(config.WithDefault(map[string]interface{}{}))
	_, err := New(cfg)
	require.Error(t, err)
}

func TestRabbitMQNewChannelError(t *testing.T) {
	origDial := dialFunc
	dialFunc = func(string) (amqpConn, error) { return &errConn{}, nil }
	defer func() { dialFunc = origDial }()

	cfg, _ := config.New(config.WithDefault(map[string]interface{}{}))
	_, err := New(cfg)
	require.Error(t, err)
}

func TestRabbitMQPublishDeclareErrorMock(t *testing.T) {
	ch := &mockChannel{declareErr: fmt.Errorf("decl")}
	origDial := dialFunc
	dialFunc = func(string) (amqpConn, error) { return &mockConn{ch: ch}, nil }
	defer func() { dialFunc = origDial }()

	cfg, _ := config.New(config.WithDefault(map[string]interface{}{}))
	rmq, err := New(cfg)
	require.NoError(t, err)

	err = rmq.Publish(context.Background(), "q", []byte("x"))
	require.Error(t, err)
}

func TestRabbitMQPublishInjectsTraceContext(t *testing.T) {
	ch := &mockChannel{consumeCh: make(chan amqp.Delivery)}
	origDial := dialFunc
	dialFunc = func(string) (amqpConn, error) { return &mockConn{ch: ch}, nil }
	defer func() { dialFunc = origDial }()

	cfg, _ := config.New(config.WithDefault(map[string]interface{}{
		"otel_enabled": true,
	}))

	os.Setenv("OTEL_TEST_MOCK_EXPORTER", "true")
	defer os.Unsetenv("OTEL_TEST_MOCK_EXPORTER")
	require.NoError(t, otel.Init(cfg))
	defer otel.Shutdown(context.Background())

	rmq, err := New(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, rmq.Publish(ctx, "q1", []byte("msg")))
	require.Len(t, ch.published, 1)

	_, found := ch.published[0].Headers["traceparent"]
	require.True(t, found, "traceparent header not found")
}

func TestRabbitMQConsumeJSONInvalidDataMock(t *testing.T) {
	ch := &mockChannel{consumeCh: make(chan amqp.Delivery, 1)}
	ch.consumeCh <- amqp.Delivery{Body: []byte("notjson")}
	close(ch.consumeCh)

	origDial := dialFunc
	dialFunc = func(string) (amqpConn, error) { return &mockConn{ch: ch}, nil }
	defer func() { dialFunc = origDial }()

	cfg, _ := config.New(config.WithDefault(map[string]interface{}{}))
	rmq, err := New(cfg)
	require.NoError(t, err)

	out, err := ConsumeJSON[map[string]string](context.Background(), rmq, "q1")
	require.NoError(t, err)

	// No panic or send due to invalid JSON
	_, ok := <-out
	require.False(t, ok)
}
