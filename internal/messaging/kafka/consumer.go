package kafka

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otrace "go.opentelemetry.io/otel/trace"
)

type MessageHandler func(context.Context, kafka.Message) error

type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(brokersCSV string, topic string, groupID string) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:                splitBrokers(brokersCSV),
			Topic:                  topic,
			GroupID:                groupID,
			MinBytes:               1,
			MaxBytes:               10e6,
			MaxWait:                500 * time.Millisecond,
			CommitInterval:         time.Second,
			HeartbeatInterval:      3 * time.Second,
			SessionTimeout:         30 * time.Second,
			RebalanceTimeout:       30 * time.Second,
			ReadLagInterval:        -1,
			WatchPartitionChanges:  true,
			PartitionWatchInterval: 5 * time.Second,
		}),
	}
}

func (c *Consumer) Run(ctx context.Context, handler MessageHandler) error {
	if handler == nil {
		return errors.New("kafka message handler is nil")
	}
	cfg := c.reader.Config()
	tracer := otel.Tracer("goapi/kafka")
	prop := otel.GetTextMapPropagator()

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "context canceled") {
				return nil
			}
			return fmt.Errorf("fetch kafka message: %w", err)
		}

		carrier := newKafkaHeaderCarrier(&msg.Headers)
		extCtx := prop.Extract(ctx, carrier)
		procCtx, span := tracer.Start(extCtx, cfg.Topic+" receive",
			otrace.WithSpanKind(otrace.SpanKindConsumer),
			otrace.WithAttributes(
				attribute.String("messaging.system", "kafka"),
				attribute.String("messaging.destination.name", cfg.Topic),
				attribute.String("messaging.operation", "receive"),
				attribute.String("messaging.consumer.group.id", cfg.GroupID),
			),
		)

		err = handler(procCtx, msg)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return fmt.Errorf("handle kafka message: %w", err)
		}
		span.End()

		if err = c.reader.CommitMessages(ctx, msg); err != nil {
			return fmt.Errorf("commit kafka message: %w", err)
		}
	}
}

func (c *Consumer) HealthCheck(ctx context.Context) error {
	conn, err := (&kafka.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "tcp", c.reader.Config().Brokers[0])
	if err != nil {
		return fmt.Errorf("kafka dial: %w", err)
	}
	defer conn.Close()
	if _, err = conn.ReadPartitions(c.reader.Config().Topic); err != nil {
		return fmt.Errorf("kafka read partitions: %w", err)
	}
	return nil
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
