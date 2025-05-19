package rabbitmq

import (
	"context"
	"fmt"
	"sync"

	"github.com/T-Prohmpossadhorn/go-core/config"
	"github.com/T-Prohmpossadhorn/go-core/logger"
)

// Config defines RabbitMQ settings.
type Config struct {
	OtelEnabled bool   `mapstructure:"otel_enabled" default:"false"`
	URL         string `mapstructure:"rabbitmq_url" default:"amqp://guest:guest@localhost:5672/"`
}

// RabbitMQ is an in-memory message queue used for demonstration.
type RabbitMQ struct {
	mu          sync.RWMutex
	queues      map[string]chan []byte
	otelEnabled bool
	url         string
}

// New creates a new RabbitMQ instance with the provided config.
func New(c *config.Config) (*RabbitMQ, error) {
	cfg := Config{
		OtelEnabled: c.GetBool("otel_enabled"),
		URL:         c.GetStringWithDefault("rabbitmq_url", "amqp://guest:guest@localhost:5672/"),
	}

	rmq := &RabbitMQ{
		queues:      make(map[string]chan []byte),
		otelEnabled: cfg.OtelEnabled,
		url:         cfg.URL,
	}
	logger.Info("RabbitMQ initialized", logger.String("url", cfg.URL))
	return rmq, nil
}

// Publish sends a message to the specified queue.
func (r *RabbitMQ) Publish(ctx context.Context, queue string, body []byte) error {
	if ctx.Err() != nil {
		return fmt.Errorf("publish canceled: %w", ctx.Err())
	}

	r.mu.Lock()
	q, ok := r.queues[queue]
	if !ok {
		q = make(chan []byte, 100)
		r.queues[queue] = q
	}
	r.mu.Unlock()

	select {
	case <-ctx.Done():
		return fmt.Errorf("publish canceled: %w", ctx.Err())
	case q <- body:
		logger.InfoContext(ctx, "Message published", logger.String("queue", queue))
		return nil
	}
}

// Consume returns a channel to receive messages from the specified queue.
func (r *RabbitMQ) Consume(ctx context.Context, queue string) (<-chan []byte, error) {
	r.mu.Lock()
	q, ok := r.queues[queue]
	if !ok {
		q = make(chan []byte, 100)
		r.queues[queue] = q
	}
	r.mu.Unlock()

	out := make(chan []byte)
	go func() {
		defer close(out)
		for {
			select {
			case msg := <-q:
				out <- msg
			case <-ctx.Done():
				return
			}
		}
	}()
	logger.InfoContext(ctx, "Consumer registered", logger.String("queue", queue))
	return out, nil
}

// Close closes all queues.
func (r *RabbitMQ) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for name, ch := range r.queues {
		close(ch)
		delete(r.queues, name)
	}
	logger.Info("RabbitMQ closed")
	return nil
}
