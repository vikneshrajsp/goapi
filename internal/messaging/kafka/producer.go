package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"goapi/internal/messaging/events"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otrace "go.opentelemetry.io/otel/trace"
)

const (
	DefaultBrokers = "kafka:9092"
	DefaultTopic   = "coinbalance_change"
)

type Producer interface {
	PublishCoinBalanceChanged(context.Context, events.CoinBalanceChanged) error
	HealthCheck(context.Context) error
	Close() error
}

type producer struct {
	brokers []string
	topic   string
	writer  *kafka.Writer
}

func NewProducer(brokersCSV string, topic string) Producer {
	brokers := splitBrokers(brokersCSV)
	if topic == "" {
		topic = DefaultTopic
	}
	return &producer{
		brokers: brokers,
		topic:   topic,
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Topic:                  topic,
			Balancer:               &kafka.Hash{},
			RequiredAcks:           kafka.RequireAll,
			AllowAutoTopicCreation: false,
			BatchTimeout:           100 * time.Millisecond,
			BatchSize:              100,
			WriteTimeout:           8 * time.Second,
			ReadTimeout:            8 * time.Second,
			MaxAttempts:            10,
			Async:                  false,
		},
	}
}

func (p *producer) PublishCoinBalanceChanged(ctx context.Context, event events.CoinBalanceChanged) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal coin balance event: %w", err)
	}

	tracer := otel.Tracer("goapi/kafka")
	ctx, span := tracer.Start(ctx, p.topic+" publish",
		otrace.WithSpanKind(otrace.SpanKindProducer),
		otrace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination.name", p.topic),
			attribute.String("messaging.operation", "publish"),
		),
	)
	defer span.End()

	headers := []kafka.Header{
		{Key: "event_type", Value: []byte(event.EventType)},
		{Key: "schema_version", Value: []byte("1")},
	}
	carrier := newKafkaHeaderCarrier(&headers)
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:     []byte(event.Username),
		Value:   body,
		Time:    event.OccurredAt,
		Headers: headers,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("kafka write event: %w", err)
	}
	return nil
}

func (p *producer) HealthCheck(ctx context.Context) error {
	conn, err := (&kafka.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "tcp", p.brokers[0])
	if err != nil {
		return fmt.Errorf("kafka dial: %w", err)
	}
	defer conn.Close()
	_, err = conn.ReadPartitions(p.topic)
	if err != nil {
		return fmt.Errorf("kafka read partitions: %w", err)
	}
	return nil
}

func (p *producer) Close() error {
	return p.writer.Close()
}

type noopProducer struct{}

func NewNoopProducer() Producer {
	return noopProducer{}
}

func (noopProducer) PublishCoinBalanceChanged(context.Context, events.CoinBalanceChanged) error {
	return nil
}
func (noopProducer) HealthCheck(context.Context) error { return nil }
func (noopProducer) Close() error                      { return nil }

func splitBrokers(v string) []string {
	if strings.TrimSpace(v) == "" {
		return []string{DefaultBrokers}
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return []string{DefaultBrokers}
	}
	return out
}
