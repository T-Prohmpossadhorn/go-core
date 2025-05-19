package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"

	otelglobal "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/T-Prohmpossadhorn/go-core/config"
	"github.com/T-Prohmpossadhorn/go-core/logger"
	"github.com/T-Prohmpossadhorn/go-core/otel"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Config defines RabbitMQ settings.
type Config struct {
	OtelEnabled bool   `mapstructure:"otel_enabled" default:"false"`
	URL         string `mapstructure:"rabbitmq_url" default:"amqp://guest:guest@localhost:5672/"`
	EnableTLS   bool   `mapstructure:"rabbitmq_enable_tls" default:"false"`
	AutoAck     bool   `mapstructure:"rabbitmq_auto_ack" default:"true"`
}

// RabbitMQ wraps a real RabbitMQ connection using the amqp091-go client.
type amqpChannel interface {
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	ConsumeWithContext(ctx context.Context, queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
	Close() error
}

type amqpConn interface {
	Channel() (amqpChannel, error)
	Close() error
}

type realConn struct{ *amqp.Connection }

func (rc *realConn) Channel() (amqpChannel, error) { return rc.Connection.Channel() }
func (rc *realConn) Close() error                  { return rc.Connection.Close() }

var dialFunc = func(url string) (amqpConn, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	return &realConn{conn}, nil
}

// RabbitMQ wraps a real RabbitMQ connection using the amqp091-go client.
type RabbitMQ struct {
	mu          sync.RWMutex
	conn        amqpConn
	channel     amqpChannel
	otelEnabled bool
	url         string
	enableTLS   bool
	autoAck     bool
	tracerName  string
}

// New creates a new RabbitMQ instance with the provided config.
func New(c *config.Config) (*RabbitMQ, error) {
	cfg := Config{
		OtelEnabled: c.GetBool("otel_enabled"),
		URL:         c.GetStringWithDefault("rabbitmq_url", "amqp://guest:guest@localhost:5672/"),
		EnableTLS:   c.GetBool("rabbitmq_enable_tls"),
	}
	autoAck := c.GetBool("rabbitmq_auto_ack")
	if c.Get("rabbitmq_auto_ack") == nil {
		autoAck = true
	}
	cfg.AutoAck = autoAck

	if cfg.EnableTLS && strings.HasPrefix(cfg.URL, "amqp://") {
		cfg.URL = "amqps://" + strings.TrimPrefix(cfg.URL, "amqp://")
	}

	conn, err := dialFunc(cfg.URL)
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
		enableTLS:   cfg.EnableTLS,
		autoAck:     cfg.AutoAck,
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

	headers := amqp.Table{}
	if r.otelEnabled {
		carrier := propagation.MapCarrier{}
		otelglobal.GetTextMapPropagator().Inject(ctx, carrier)
		for k, v := range carrier {
			headers[k] = v
		}
	}

	err = r.channel.PublishWithContext(ctx, "", queue, false, false, amqp.Publishing{
		ContentType: "application/octet-stream",
		Body:        body,
		Headers:     headers,
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

	deliveries, err := r.channel.ConsumeWithContext(ctx, queue, "", r.autoAck, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("consume: %w", err)
	}

	out := make(chan []byte)
	go func() {
		defer close(out)
		for d := range deliveries {
			if r.otelEnabled {
				carrier := propagation.MapCarrier{}
				for k, v := range d.Headers {
					switch val := v.(type) {
					case string:
						carrier[k] = val
					case []byte:
						carrier[k] = string(val)
					}
				}
				msgCtx := otelglobal.GetTextMapPropagator().Extract(ctx, carrier)
				_, span := otel.StartSpan(msgCtx, r.tracerName, "ConsumeMessage")
				span.End()
			}
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

// PublishJSON marshals v as JSON and publishes it to the specified queue.
func PublishJSON[T any](ctx context.Context, r *RabbitMQ, queue string, v T) error {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}
	return r.Publish(ctx, queue, b)
}

// ConsumeJSON consumes messages from the queue and unmarshals them into type T.
func ConsumeJSON[T any](ctx context.Context, r *RabbitMQ, queue string) (<-chan T, error) {
	byteCh, err := r.Consume(ctx, queue)
	if err != nil {
		return nil, err
	}
	out := make(chan T)
	go func() {
		defer close(out)
		for b := range byteCh {
			var v T
			if err := json.Unmarshal(b, &v); err != nil {
				_ = logger.ErrorContext(ctx, "Failed to unmarshal message", logger.ErrField(err))
				continue
			}
			out <- v
		}
	}()
	return out, nil
}
