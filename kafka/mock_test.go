package kafka

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"testing"

	kafka_go "github.com/segmentio/kafka-go"

	"github.com/T-Prohmpossadhorn/go-core/config"
	"github.com/T-Prohmpossadhorn/go-core/otel"
	"github.com/stretchr/testify/require"
)

type mockWriter struct{ msgs []kafka_go.Message }

func (m *mockWriter) WriteMessages(ctx context.Context, msgs ...kafka_go.Message) error {
	m.msgs = append(m.msgs, msgs...)
	return nil
}

func (m *mockWriter) Close() error { return nil }

type mockReader struct{ ch chan kafka_go.Message }

func (m *mockReader) ReadMessage(ctx context.Context) (kafka_go.Message, error) {
	msg, ok := <-m.ch
	if !ok {
		return kafka_go.Message{}, io.EOF
	}
	return msg, nil
}

func (m *mockReader) Close() error { return nil }

func TestKafkaPublishConsumeMock(t *testing.T) {
	mw := &mockWriter{}
	mr := &mockReader{ch: make(chan kafka_go.Message, 1)}
	mr.ch <- kafka_go.Message{Value: []byte("consumed")}
	close(mr.ch)

	origW, origR := writerFactoryFunc, readerFactoryFunc
	writerFactoryFunc = func([]string, string) writer { return mw }
	readerFactoryFunc = func([]string, string) reader { return mr }
	defer func() { writerFactoryFunc, readerFactoryFunc = origW, origR }()

	cfg, _ := config.New(config.WithDefault(map[string]interface{}{}))
	k, err := New(cfg)
	require.NoError(t, err)

	out, err := k.Consume(context.Background(), "t1")
	require.NoError(t, err)

	require.NoError(t, k.Publish(context.Background(), "t1", []byte("hello")))
	require.Len(t, mw.msgs, 1)
	require.Equal(t, []byte("hello"), mw.msgs[0].Value)

	msg := <-out
	require.Equal(t, []byte("consumed"), msg)
}

func TestKafkaPublishConsumeJSONMock(t *testing.T) {
	type msg struct {
		Name string `json:"name"`
	}

	mw := &mockWriter{}
	mr := &mockReader{ch: make(chan kafka_go.Message, 1)}
	b, _ := json.Marshal(msg{Name: "consumed"})
	mr.ch <- kafka_go.Message{Value: b}
	close(mr.ch)

	origW, origR := writerFactoryFunc, readerFactoryFunc
	writerFactoryFunc = func([]string, string) writer { return mw }
	readerFactoryFunc = func([]string, string) reader { return mr }
	defer func() { writerFactoryFunc, readerFactoryFunc = origW, origR }()

	cfg, _ := config.New(config.WithDefault(map[string]interface{}{}))
	k, err := New(cfg)
	require.NoError(t, err)

	out, err := ConsumeJSON[msg](context.Background(), k, "t1")
	require.NoError(t, err)

	require.NoError(t, PublishJSON(context.Background(), k, "t1", msg{Name: "hello"}))
	require.Len(t, mw.msgs, 1)
	var sent msg
	_ = json.Unmarshal(mw.msgs[0].Value, &sent)
	require.Equal(t, "hello", sent.Name)

	m := <-out
	require.Equal(t, "consumed", m.Name)
}

func TestKafkaCloseMock(t *testing.T) {
	mw := &mockWriter{}
	mr := &mockReader{ch: make(chan kafka_go.Message)}

	origW, origR := writerFactoryFunc, readerFactoryFunc
	writerFactoryFunc = func([]string, string) writer { return mw }
	readerFactoryFunc = func([]string, string) reader { return mr }
	defer func() { writerFactoryFunc, readerFactoryFunc = origW, origR }()

	cfg, _ := config.New(config.WithDefault(map[string]interface{}{}))
	k, err := New(cfg)
	require.NoError(t, err)

	_, err = k.Consume(context.Background(), "t1")
	require.NoError(t, err)
	require.NoError(t, k.Publish(context.Background(), "t1", []byte("x")))

	require.NoError(t, k.Close())
}

func TestKafkaPublishCanceledMock(t *testing.T) {
	mw := &mockWriter{}

	origW := writerFactoryFunc
	writerFactoryFunc = func([]string, string) writer { return mw }
	defer func() { writerFactoryFunc = origW }()

	cfg, _ := config.New(config.WithDefault(map[string]interface{}{}))
	k, err := New(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = k.Publish(ctx, "t1", []byte("x"))
	require.Error(t, err)
}

func TestKafkaPublishInjectsTraceContext(t *testing.T) {
	mw := &mockWriter{}
	mr := &mockReader{ch: make(chan kafka_go.Message)}
	origW, origR := writerFactoryFunc, readerFactoryFunc
	writerFactoryFunc = func([]string, string) writer { return mw }
	readerFactoryFunc = func([]string, string) reader { return mr }
	defer func() { writerFactoryFunc, readerFactoryFunc = origW, origR }()

	cfg, _ := config.New(config.WithDefault(map[string]interface{}{
		"otel_enabled": true,
	}))

	os.Setenv("OTEL_TEST_MOCK_EXPORTER", "true")
	defer os.Unsetenv("OTEL_TEST_MOCK_EXPORTER")
	require.NoError(t, otel.Init(cfg))
	defer otel.Shutdown(context.Background())

	k, err := New(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, k.Publish(ctx, "t1", []byte("msg")))
	require.Len(t, mw.msgs, 1)

	found := false
	for _, h := range mw.msgs[0].Headers {
		if h.Key == "traceparent" {
			found = true
			break
		}
	}
	require.True(t, found, "traceparent header not found")
}
