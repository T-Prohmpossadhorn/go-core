package rabbitmq

import (
	"context"
	"errors"
	"testing"

	"github.com/T-Prohmpossadhorn/go-core/config"
	amqp "github.com/rabbitmq/amqp091-go"
)

type errChannel struct{}

func (e *errChannel) QueueDeclare(string, bool, bool, bool, bool, amqp.Table) (amqp.Queue, error) {
	return amqp.Queue{}, errors.New("decl")
}
func (e *errChannel) PublishWithContext(context.Context, string, string, bool, bool, amqp.Publishing) error {
	return nil
}
func (e *errChannel) ConsumeWithContext(context.Context, string, string, bool, bool, bool, bool, amqp.Table) (<-chan amqp.Delivery, error) {
	return nil, errors.New("consume")
}
func (e *errChannel) Close() error { return nil }

type errConnConsume struct{}

func (e *errConnConsume) Channel() (amqpChannel, error) { return &errChannel{}, nil }
func (e *errConnConsume) Close() error                  { return nil }

type badStruct struct{ Fn func() }

// TestPublishJSONMarshalError verifies PublishJSON returns marshal errors.
func TestPublishJSONMarshalError(t *testing.T) {
	cfg, _ := config.New()
	ch := &mockChannel{consumeCh: make(chan amqp.Delivery)}
	origDial := dialFunc
	dialFunc = func(string) (amqpConn, error) { return &mockConn{ch: ch}, nil }
	defer func() { dialFunc = origDial }()
	rmq, _ := New(cfg)
	err := PublishJSON(context.Background(), rmq, "q", badStruct{})
	if err == nil {
		t.Fatal("expected marshal error")
	}
}

// TestConsumeWithChannelError ensures Consume handles ConsumeWithContext errors.
func TestConsumeWithChannelError(t *testing.T) {
	origDial := dialFunc
	dialFunc = func(string) (amqpConn, error) { return &errConnConsume{}, nil }
	defer func() { dialFunc = origDial }()
	cfg, _ := config.New()
	rmq, _ := New(cfg)
	ch, err := rmq.Consume(context.Background(), "q")
	if err == nil {
		t.Fatalf("expected error, got channel %v", ch)
	}
}
