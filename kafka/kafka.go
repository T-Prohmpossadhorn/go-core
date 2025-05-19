package kafka

import (
	"context"
	"fmt"
	"sync"

	"github.com/T-Prohmpossadhorn/go-core/config"
	"github.com/T-Prohmpossadhorn/go-core/logger"
	"github.com/T-Prohmpossadhorn/go-core/otel"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Config defines Kafka settings.
type Config struct {
	OtelEnabled bool   `mapstructure:"otel_enabled" default:"false"`
	Brokers     string `mapstructure:"kafka_brokers" default:"localhost:9092"`
	Topic       string `mapstructure:"kafka_topic" default:"default"`
}

// Kafka is an in-memory message queue used for demonstration.
type Kafka struct {
	mu         sync.RWMutex
	topics     map[string]chan []byte
	cfg        Config
	tracerName string
}

// New creates a new Kafka instance with the provided config.
func New(c *config.Config) (*Kafka, error) {
	cfg := Config{
		OtelEnabled: c.GetBool("otel_enabled"),
		Brokers:     c.GetStringWithDefault("kafka_brokers", "localhost:9092"),
		Topic:       c.GetStringWithDefault("kafka_topic", "default"),
	}

	k := &Kafka{
		topics:     make(map[string]chan []byte),
		cfg:        cfg,
		tracerName: "kafka",
	}
	logger.Info("Kafka initialized", logger.String("brokers", cfg.Brokers), logger.String("topic", cfg.Topic))
	return k, nil
}

// Publish sends a message to the specified topic.
func (k *Kafka) Publish(ctx context.Context, topic string, body []byte) error {
	var span oteltrace.Span
	if k.cfg.OtelEnabled {
		ctx, span = otel.StartSpan(ctx, k.tracerName, "Publish")
		defer span.End()
	}
	if ctx.Err() != nil {
		return fmt.Errorf("publish canceled: %w", ctx.Err())
	}

	k.mu.Lock()
	q, ok := k.topics[topic]
	if !ok {
		q = make(chan []byte, 100)
		k.topics[topic] = q
	}
	k.mu.Unlock()

	select {
	case <-ctx.Done():
		return fmt.Errorf("publish canceled: %w", ctx.Err())
	case q <- body:
		logger.InfoContext(ctx, "Message published", logger.String("topic", topic))
		return nil
	}
}

// Consume returns a channel to receive messages from the specified topic.
func (k *Kafka) Consume(ctx context.Context, topic string) (<-chan []byte, error) {
	var span oteltrace.Span
	if k.cfg.OtelEnabled {
		ctx, span = otel.StartSpan(ctx, k.tracerName, "Consume")
		defer span.End()
	}

	k.mu.Lock()
	q, ok := k.topics[topic]
	if !ok {
		q = make(chan []byte, 100)
		k.topics[topic] = q
	}
	k.mu.Unlock()

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
	logger.InfoContext(ctx, "Consumer registered", logger.String("topic", topic))
	return out, nil
}

// Close closes all topics.
func (k *Kafka) Close() error {
	k.mu.Lock()
	defer k.mu.Unlock()
	for name, ch := range k.topics {
		close(ch)
		delete(k.topics, name)
	}
	logger.Info("Kafka closed")
	return nil
}
