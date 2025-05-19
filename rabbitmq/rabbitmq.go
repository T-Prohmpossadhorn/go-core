package rabbitmq

import (
	"context"
	"fmt"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/T-Prohmpossadhorn/go-core/config"
	"github.com/T-Prohmpossadhorn/go-core/logger"
	"github.com/T-Prohmpossadhorn/go-core/otel"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Config defines RabbitMQ settings.
type Config struct {
	OtelEnabled bool   `mapstructure:"otel_enabled" default:"false"`
	URL         string `mapstructure:"rabbitmq_url" default:"amqp://guest:guest@localhost:5672/"`
}

// RabbitMQ wraps a real RabbitMQ connection using the amqp091-go client.
type RabbitMQ struct {
	mu          sync.RWMutex
	conn        *amqp.Connection
	channel     *amqp.Channel
	otelEnabled bool
	url         string
	tracerName  string
}

// New creates a new RabbitMQ instance with the provided config.
func New(c *config.Config) (*RabbitMQ, error) {
	cfg := Config{
		OtelEnabled: c.GetBool("otel_enabled"),
		URL:         c.GetStringWithDefault("rabbitmq_url", "amqp://guest:guest@localhost:5672/"),
	}

	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("connect rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("open channel: %w", err)
	}

	rmq := &RabbitMQ{
		conn:        conn,
		channel:     ch,
		otelEnabled: cfg.OtelEnabled,
		url:         cfg.URL,
		tracerName:  "rabbitmq",
	}
	logger.Info("RabbitMQ initialized", logger.String("url", cfg.URL))
	return rmq, nil
}

// Publish sends a message to the specified queue.
func (r *RabbitMQ) Publish(ctx context.Context, queue string, body []byte) error {
	var span oteltrace.Span
	if r.otelEnabled {
		ctx, span = otel.StartSpan(ctx, r.tracerName, "Publish")
		defer span.End()
	}
	if ctx.Err() != nil {
		return fmt.Errorf("publish canceled: %w", ctx.Err())
	}

	_, err := r.channel.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	err = r.channel.PublishWithContext(ctx, "", queue, false, false, amqp.Publishing{
		ContentType: "application/octet-stream",
		Body:        body,
	})
	if err != nil {
		return fmt.Errorf("publish message: %w", err)
	}
	logger.InfoContext(ctx, "Message published", logger.String("queue", queue))
	return nil
}

// Consume returns a channel to receive messages from the specified queue.
func (r *RabbitMQ) Consume(ctx context.Context, queue string) (<-chan []byte, error) {
	var span oteltrace.Span
	if r.otelEnabled {
		ctx, span = otel.StartSpan(ctx, r.tracerName, "Consume")
		defer span.End()
	}

	_, err := r.channel.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("declare queue: %w", err)
	}

	deliveries, err := r.channel.ConsumeWithContext(ctx, queue, "", true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("consume: %w", err)
	}

	out := make(chan []byte)
	go func() {
		defer close(out)
		for d := range deliveries {
			out <- d.Body
		}
	}()
	logger.InfoContext(ctx, "Consumer registered", logger.String("queue", queue))
	return out, nil
}

// Close shuts down the channel and connection.
func (r *RabbitMQ) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.channel != nil {
		_ = r.channel.Close()
	}
	if r.conn != nil {
		_ = r.conn.Close()
	}
	logger.Info("RabbitMQ closed")
	return nil
}
