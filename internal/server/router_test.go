package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRouterRegistersHealthEndpoints(t *testing.T) {
	r := NewRouter(mockRepo(t))

	pingReq := httptest.NewRequest(http.MethodGet, "/ping", nil)
	pingRec := httptest.NewRecorder()
	r.ServeHTTP(pingRec, pingReq)
	if pingRec.Code != http.StatusOK {
		t.Fatalf("expected ping 200, got %d", pingRec.Code)
	}

	healthReq := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthRec := httptest.NewRecorder()
	r.ServeHTTP(healthRec, healthReq)
	if healthRec.Code != http.StatusOK {
		t.Fatalf("expected health 200, got %d", healthRec.Code)
	}
}

func TestNewRouterRegistersMetricsEndpoint(t *testing.T) {
	r := NewRouter(mockRepo(t))
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected metrics endpoint 200, got %d", rec.Code)
	}
}
