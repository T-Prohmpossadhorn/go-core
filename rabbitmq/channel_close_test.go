package rabbitmq

import (
	"context"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/T-Prohmpossadhorn/go-core/config"
)

// mockChan to test Channel method
type mockChan struct{}

func (m *mockChan) QueueDeclare(string, bool, bool, bool, bool, amqp.Table) (amqp.Queue, error) {
	return amqp.Queue{}, nil
}
func (m *mockChan) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	return nil
}
func (m *mockChan) ConsumeWithContext(ctx context.Context, queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	return nil, nil
}
func (m *mockChan) Close() error { return nil }

type mockConnForChannel struct{}

func (m *mockConnForChannel) Channel() (amqpChannel, error) { return &mockChan{}, nil }
func (m *mockConnForChannel) Close() error                  { return nil }

func TestRabbitMQChannelClose(t *testing.T) {
	origDial := dialFunc
	dialFunc = func(string) (amqpConn, error) { return &mockConnForChannel{}, nil }
	defer func() { dialFunc = origDial }()

	cfg, _ := config.New()
	r, err := New(cfg)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if r.channel == nil {
		t.Fatal("expected channel initialized")
	}

	if err := r.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}
