package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type failingKafkaHealth struct{}

func (failingKafkaHealth) HealthCheck(context.Context) error { return errors.New("down") }

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	HealthHandler(testDeps().KafkaHealth)(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rec.Code)
	}
}

func TestHealthHandlerKafkaFailure(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	HealthHandler(failingKafkaHealth{})(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 got %d", rec.Code)
	}
}
