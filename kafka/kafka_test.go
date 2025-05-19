package kafka

import (
	"context"
	"testing"

	"github.com/T-Prohmpossadhorn/go-core/config"
	"github.com/stretchr/testify/require"
)

func TestPublishConsume(t *testing.T) {
	cfg, err := config.New(config.WithDefault(map[string]interface{}{}))
	require.NoError(t, err)
	k, err := New(cfg)
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()
	require.NoError(t, k.Publish(ctx, "q", []byte("hi")))
	msgs, err := k.Consume(ctx, "q")
	require.NoError(t, err)
	msg := <-msgs
	require.Equal(t, []byte("hi"), msg)
}

func TestPublishCanceled(t *testing.T) {
	cfg, _ := config.New()
	k, _ := New(cfg)
	defer k.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := k.Publish(ctx, "q", []byte("hi"))
	require.Error(t, err)
}
