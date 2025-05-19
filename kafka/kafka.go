package kafka

import (
	"context"
	"fmt"
	"strings"
	"sync"

	kafka_go "github.com/segmentio/kafka-go"

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

// Kafka wraps kafka-go writers and readers to talk to a real Kafka broker.
type Kafka struct {
	mu         sync.RWMutex
	writers    map[string]*kafka_go.Writer
	readers    map[string]*kafka_go.Reader
	brokers    []string
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

	brokers := strings.Split(cfg.Brokers, ",")
	k := &Kafka{
		writers:    make(map[string]*kafka_go.Writer),
		readers:    make(map[string]*kafka_go.Reader),
		brokers:    brokers,
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
	w, ok := k.writers[topic]
	if !ok {
		w = &kafka_go.Writer{
			Addr:     kafka_go.TCP(k.brokers...),
			Topic:    topic,
			Balancer: &kafka_go.LeastBytes{},
		}
		k.writers[topic] = w
	}
	k.mu.Unlock()

	err := w.WriteMessages(ctx, kafka_go.Message{Value: body})
	if err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	logger.InfoContext(ctx, "Message published", logger.String("topic", topic))
	return nil
}

// Consume returns a channel to receive messages from the specified topic.
func (k *Kafka) Consume(ctx context.Context, topic string) (<-chan []byte, error) {
	var span oteltrace.Span
	if k.cfg.OtelEnabled {
		ctx, span = otel.StartSpan(ctx, k.tracerName, "Consume")
		defer span.End()
	}

	k.mu.Lock()
	r, ok := k.readers[topic]
	if !ok {
		r = kafka_go.NewReader(kafka_go.ReaderConfig{
			Brokers: k.brokers,
			Topic:   topic,
			GroupID: "",
		})
		k.readers[topic] = r
	}
	k.mu.Unlock()

	out := make(chan []byte)
	go func() {
		defer close(out)
		for {
			m, err := r.ReadMessage(ctx)
			if err != nil {
				return
			}
			out <- m.Value
		}
	}()
	logger.InfoContext(ctx, "Consumer registered", logger.String("topic", topic))
	return out, nil
}

// Close shuts down all readers and writers.
func (k *Kafka) Close() error {
	k.mu.Lock()
	defer k.mu.Unlock()
	for _, w := range k.writers {
		_ = w.Close()
	}
	for _, r := range k.readers {
		_ = r.Close()
	}
	k.writers = map[string]*kafka_go.Writer{}
	k.readers = map[string]*kafka_go.Reader{}
	logger.Info("Kafka closed")
	return nil
}
