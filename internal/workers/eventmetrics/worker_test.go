package eventmetrics

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/segmentio/kafka-go"
	"goapi/internal/messaging/events"
)

type failingHealth struct{}

func (f failingHealth) HealthCheck(context.Context) error { return errors.New("down") }

func TestHandleEvent(t *testing.T) {
	w := New()
	event := events.CoinBalanceChanged{
		SchemaVersion: 1,
		EventID:       "e-1",
		EventType:     events.CoinBalanceChangedType,
		Username:      "alex",
		Previous:      10,
		Current:       20,
		Delta:         -10,
	}
	body, _ := json.Marshal(event)
	if err := w.Handle(context.Background(), kafka.Message{Value: body}); err != nil {
		t.Fatalf("handle event: %v", err)
	}
}

func TestHandleEventZeroDelta(t *testing.T) {
	w := New()
	event := events.CoinBalanceChanged{
		SchemaVersion: 1,
		EventID:       "e-0",
		EventType:     events.CoinBalanceChangedType,
		Username:      "alex",
		Previous:      50,
		Current:       50,
		Delta:         0,
	}
	body, _ := json.Marshal(event)
	if err := w.Handle(context.Background(), kafka.Message{Value: body}); err != nil {
		t.Fatalf("handle event: %v", err)
	}
}

func TestHandleEventInvalidPayload(t *testing.T) {
	w := New()
	if err := w.Handle(context.Background(), kafka.Message{Value: []byte("{")}); err == nil {
		t.Fatal("expected unmarshal error")
	}
}

func TestHealthHandler(t *testing.T) {
	h := HealthHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rec.Code)
	}
}

func TestHealthHandlerFailure(t *testing.T) {
	h := HealthHandler(failingHealth{})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 got %d", rec.Code)
	}
}
