//go:build testcontainers

package kafka

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	segkafka "github.com/segmentio/kafka-go"
	tckafka "github.com/testcontainers/testcontainers-go/modules/kafka"
	"goapi/internal/messaging/events"
)

func TestProducerConsumerWithKafkaContainer(t *testing.T) {
	ctx := context.Background()
	kafkaContainer, err := tckafka.Run(ctx, "confluentinc/confluent-local:7.5.0")
	if err != nil {
		t.Fatalf("start kafka container: %v", err)
	}
	t.Cleanup(func() {
		_ = kafkaContainer.Terminate(ctx)
	})

	brokers, err := kafkaContainer.Brokers(ctx)
	if err != nil {
		t.Fatalf("brokers: %v", err)
	}
	broker := brokers[0]

	conn, err := segkafka.DialContext(ctx, "tcp", broker)
	if err != nil {
		t.Fatalf("dial broker: %v", err)
	}
	if err = conn.CreateTopics(segkafka.TopicConfig{
		Topic:             DefaultTopic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	}); err != nil {
		_ = conn.Close()
		t.Fatalf("create topic: %v", err)
	}
	_ = conn.Close()

	producer := NewProducer(strings.Join(brokers, ","), DefaultTopic)
	defer producer.Close()
	consumer := NewConsumer(strings.Join(brokers, ","), DefaultTopic, "coinbalance-test")
	defer consumer.Close()
	if err = producer.HealthCheck(ctx); err != nil {
		t.Fatalf("producer health check: %v", err)
	}
	if err = consumer.HealthCheck(ctx); err != nil {
		t.Fatalf("consumer health check: %v", err)
	}

	event := events.CoinBalanceChanged{
		SchemaVersion: 1,
		EventID:       "tc-1",
		EventType:     events.CoinBalanceChangedType,
		Username:      "alex",
		Previous:      100,
		Current:       120,
		Delta:         20,
		OccurredAt:    time.Now().UTC(),
	}

	received := make(chan struct{}, 1)
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		_ = consumer.Run(runCtx, func(_ context.Context, msg segkafka.Message) error {
			if len(msg.Value) > 0 {
				received <- struct{}{}
				cancel()
			}
			return nil
		})
	}()

	if err = producer.PublishCoinBalanceChanged(ctx, event); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case <-received:
	case <-time.After(20 * time.Second):
		t.Fatal("timed out waiting for consumed message")
	}

	t.Run("handler error bubbles", func(t *testing.T) {
		consumerErr := NewConsumer(strings.Join(brokers, ","), DefaultTopic, "coinbalance-test-error")
		defer consumerErr.Close()

		runCtxErr, cancelErr := context.WithCancel(ctx)
		defer cancelErr()
		result := make(chan error, 1)
		go func() {
			result <- consumerErr.Run(runCtxErr, func(_ context.Context, _ segkafka.Message) error {
				return errors.New("handler failure")
			})
		}()

		if err = producer.PublishCoinBalanceChanged(ctx, event); err != nil {
			t.Fatalf("publish for error flow: %v", err)
		}

		select {
		case runErr := <-result:
			if runErr == nil {
				t.Fatal("expected handler error from consumer run")
			}
		case <-time.After(20 * time.Second):
			t.Fatal("timed out waiting for handler error")
		}
	})
}
