package kafka

import (
	"context"
	"strings"
	"testing"
	"time"

	"goapi/internal/messaging/events"

	segkafka "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestKafkaHeaderCarrierInjectAddsTraceparent(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	prevTP := otel.GetTracerProvider()
	prevProp := otel.GetTextMapPropagator()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	defer otel.SetTracerProvider(prevTP)
	defer otel.SetTextMapPropagator(prevProp)
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
	})

	ctx := context.Background()
	tracer := tp.Tracer("kafka-test")
	ctx, sp := tracer.Start(ctx, "root")
	defer sp.End()

	headers := []segkafka.Header{{Key: "event_type", Value: []byte("coinbalance_change")}}
	carrier := newKafkaHeaderCarrier(&headers)
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	var traceparent string
	for _, h := range headers {
		if strings.EqualFold(h.Key, "traceparent") {
			traceparent = string(h.Value)
			break
		}
	}
	if traceparent == "" {
		t.Fatal("expected traceparent header after inject")
	}
}

func TestKafkaHeaderCarrierGetKeys(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	prevTP := otel.GetTracerProvider()
	prevProp := otel.GetTextMapPropagator()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	defer otel.SetTracerProvider(prevTP)
	defer otel.SetTextMapPropagator(prevProp)
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
	})

	ctx := context.Background()
	tracer := tp.Tracer("kafka-test")
	ctx, sp := tracer.Start(ctx, "root")
	defer sp.End()

	headers := []segkafka.Header{}
	carrier := newKafkaHeaderCarrier(&headers)
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	if carrier.Get("traceparent") == "" {
		t.Fatal("expected Get to return traceparent")
	}
	keys := carrier.Keys()
	if len(keys) < 1 {
		t.Fatalf("expected Keys() non-empty, got %#v", keys)
	}
}

func TestKafkaHeaderCarrierExtractRoundTrip(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	prevTP := otel.GetTracerProvider()
	prevProp := otel.GetTextMapPropagator()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	defer otel.SetTracerProvider(prevTP)
	defer otel.SetTextMapPropagator(prevProp)
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
	})

	tracer := tp.Tracer("kafka-test")
	ctx, sp := tracer.Start(context.Background(), "producer")
	defer sp.End()

	headers := []segkafka.Header{}
	otel.GetTextMapPropagator().Inject(ctx, newKafkaHeaderCarrier(&headers))

	childCtx := otel.GetTextMapPropagator().Extract(context.Background(), newKafkaHeaderCarrier(&headers))
	_, child := tracer.Start(childCtx, "consumer")
	defer child.End()
}

func TestProducerPublishCoinBalanceChangedWriteFails(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	prevTP := otel.GetTracerProvider()
	prevProp := otel.GetTextMapPropagator()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	defer otel.SetTracerProvider(prevTP)
	defer otel.SetTextMapPropagator(prevProp)
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
	})

	p := NewProducer("127.0.0.1:1", "missing-topic")
	defer p.Close()

	err := p.PublishCoinBalanceChanged(context.Background(), events.CoinBalanceChanged{
		SchemaVersion: 1,
		EventType:     events.CoinBalanceChangedType,
		Username:      "u",
		OccurredAt:    time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("expected kafka write to fail")
	}
}

func TestSplitBrokers(t *testing.T) {
	out := splitBrokers("kafka:9092, kafka2:9092")
	if len(out) != 2 {
		t.Fatalf("expected 2 brokers got %d", len(out))
	}
}

func TestSplitBrokersDefault(t *testing.T) {
	out := splitBrokers(" ")
	if len(out) != 1 || out[0] != DefaultBrokers {
		t.Fatalf("unexpected default brokers: %#v", out)
	}
}

func TestNoopProducer(t *testing.T) {
	p := NewNoopProducer()
	err := p.PublishCoinBalanceChanged(context.Background(), events.CoinBalanceChanged{
		SchemaVersion: 1,
		EventType:     events.CoinBalanceChangedType,
		Username:      "alex",
	})
	if err != nil {
		t.Fatalf("noop publish returned error: %v", err)
	}
	if err = p.HealthCheck(context.Background()); err != nil {
		t.Fatalf("noop health returned error: %v", err)
	}
	if err = p.Close(); err != nil {
		t.Fatalf("noop close returned error: %v", err)
	}
}

func TestProducerHealthCheckFailure(t *testing.T) {
	p := NewProducer("127.0.0.1:1", "missing-topic")
	err := p.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("expected health check to fail for invalid broker")
	}
}

func TestConsumerNilHandler(t *testing.T) {
	c := NewConsumer("127.0.0.1:1", "t", "g")
	defer c.Close()
	err := c.Run(context.Background(), nil)
	if err == nil || err.Error() != "kafka message handler is nil" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConsumerHealthCheckFailure(t *testing.T) {
	c := NewConsumer("127.0.0.1:1", "t", "g")
	defer c.Close()
	err := c.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("expected health check failure")
	}
}

func TestConsumerClose(t *testing.T) {
	c := NewConsumer("127.0.0.1:1", "t", "g")
	if err := c.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
}
