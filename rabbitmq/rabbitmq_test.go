package rabbitmq

import (
	"context"
	"testing"

	"github.com/T-Prohmpossadhorn/go-core/config"
	"github.com/stretchr/testify/require"
)

func TestPublishConsume(t *testing.T) {
	cfg, err := config.New(config.WithDefault(map[string]interface{}{}))
	require.NoError(t, err)
	rmq, err := New(cfg)
	require.NoError(t, err)
	defer rmq.Close()

	ctx := context.Background()
	require.NoError(t, rmq.Publish(ctx, "q", []byte("hi")))
	msgs, err := rmq.Consume(ctx, "q")
	require.NoError(t, err)
	msg := <-msgs
	require.Equal(t, []byte("hi"), msg)
}

func TestPublishCanceled(t *testing.T) {
	cfg, _ := config.New()
	rmq, _ := New(cfg)
	defer rmq.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := rmq.Publish(ctx, "q", []byte("hi"))
	require.Error(t, err)
}
