package kafka

import (
	"context"
	"errors"
	"testing"

	"github.com/T-Prohmpossadhorn/go-core/config"
	kafka_go "github.com/segmentio/kafka-go"
)

type errReader struct{}

func (e *errReader) ReadMessage(context.Context) (kafka_go.Message, error) {
	return kafka_go.Message{}, errors.New("boom")
}
func (e *errReader) Close() error { return nil }

type badType struct{ Ch chan int }
type noWriter struct{}

func (n *noWriter) WriteMessages(context.Context, ...kafka_go.Message) error { return nil }
func (n *noWriter) Close() error                                             { return nil }

// TestPublishJSONMarshalError ensures PublishJSON returns marshal errors.
func TestPublishJSONMarshalError(t *testing.T) {
	origWriter := writerFactoryFunc
	writerFactoryFunc = func([]string, string, Config) writer { return &noWriter{} }
	defer func() { writerFactoryFunc = origWriter }()
	cfg, _ := config.New()
	k, _ := New(cfg)
	err := PublishJSON(context.Background(), k, "t", badType{})
	if err == nil {
		t.Fatal("expected marshal error")
	}
}

// TestConsumeReaderError verifies Consume stops on reader error.
func TestConsumeReaderError(t *testing.T) {
	origReader := readerFactoryFunc
	readerFactoryFunc = func([]string, string, Config) reader { return &errReader{} }
	defer func() { readerFactoryFunc = origReader }()

	cfg, _ := config.New()
	k, _ := New(cfg)
	ch, err := k.Consume(context.Background(), "t")
	if err != nil {
		t.Fatalf("consume returned error: %v", err)
	}
	if _, ok := <-ch; ok {
		t.Fatal("expected channel to close on error")
	}
}
