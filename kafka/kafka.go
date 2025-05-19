package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	kafka_go "github.com/segmentio/kafka-go"

	otelglobal "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

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
// writer defines the minimal interface needed from kafka-go writers.
type writer interface {
	WriteMessages(context.Context, ...kafka_go.Message) error
	Close() error
}

// reader defines the minimal interface needed from kafka-go readers.
type reader interface {
	ReadMessage(context.Context) (kafka_go.Message, error)
	Close() error
}

// writerFactoryFunc creates a writer for a topic.
var writerFactoryFunc = func(brokers []string, topic string) writer {
	return &kafka_go.Writer{
		Addr:     kafka_go.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka_go.LeastBytes{},
	}
}

// readerFactoryFunc creates a reader for a topic.
var readerFactoryFunc = func(brokers []string, topic string) reader {
	return kafka_go.NewReader(kafka_go.ReaderConfig{
		Brokers: brokers,
		Topic:   topic,
		GroupID: "",
	})
}

// Kafka wraps kafka-go writers and readers to talk to a real Kafka broker.
type Kafka struct {
	mu         sync.RWMutex
	writers    map[string]writer
	readers    map[string]reader
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
		writers:    make(map[string]writer),
		readers:    make(map[string]reader),
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
		w = writerFactoryFunc(k.brokers, topic)
		k.writers[topic] = w
	}
	k.mu.Unlock()

	var headers []kafka_go.Header
	if k.cfg.OtelEnabled {
		carrier := propagation.MapCarrier{}
		otelglobal.GetTextMapPropagator().Inject(ctx, carrier)
		headers = make([]kafka_go.Header, 0, len(carrier))
		for k, v := range carrier {
			headers = append(headers, kafka_go.Header{Key: k, Value: []byte(v)})
		}
	}

	err := w.WriteMessages(ctx, kafka_go.Message{Value: body, Headers: headers})
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
		r = readerFactoryFunc(k.brokers, topic)
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
			if k.cfg.OtelEnabled {
				carrier := propagation.MapCarrier{}
				for _, h := range m.Headers {
					carrier[h.Key] = string(h.Value)
				}
				msgCtx := otelglobal.GetTextMapPropagator().Extract(ctx, carrier)
				_, span := otel.StartSpan(msgCtx, k.tracerName, "ConsumeMessage")
				span.End()
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
	k.writers = map[string]writer{}
	k.readers = map[string]reader{}
	logger.Info("Kafka closed")
	return nil
}

// PublishJSON marshals v as JSON and publishes it to the specified topic.
func PublishJSON[T any](ctx context.Context, k *Kafka, topic string, v T) error {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}
	return k.Publish(ctx, topic, b)
}

// ConsumeJSON consumes messages from the topic and unmarshals them into type T.
func ConsumeJSON[T any](ctx context.Context, k *Kafka, topic string) (<-chan T, error) {
	byteCh, err := k.Consume(ctx, topic)
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
