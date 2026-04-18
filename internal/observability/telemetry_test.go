package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"
)

func TestEnvOrDefault(t *testing.T) {
	t.Setenv("TEST_KEY", "")
	if got := envOrDefault("TEST_KEY", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}

	t.Setenv("TEST_KEY", " configured ")
	if got := envOrDefault("TEST_KEY", "fallback"); got != "configured" {
		t.Fatalf("expected trimmed value, got %q", got)
	}
}

func TestParseFloatEnv(t *testing.T) {
	t.Setenv("SAMPLE_RATIO", "")
	if got := parseFloatEnv("SAMPLE_RATIO", 0.5); got != 0.5 {
		t.Fatalf("expected fallback 0.5, got %v", got)
	}

	t.Setenv("SAMPLE_RATIO", "0.75")
	if got := parseFloatEnv("SAMPLE_RATIO", 0.5); got != 0.75 {
		t.Fatalf("expected parsed value 0.75, got %v", got)
	}

	t.Setenv("SAMPLE_RATIO", "invalid")
	if got := parseFloatEnv("SAMPLE_RATIO", 0.5); got != 0.5 {
		t.Fatalf("expected fallback for invalid value, got %v", got)
	}
}

func TestRoutePatternOrPathFallbackToURLPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/plain", nil)
	if got := routePatternOrPath(req); got != "/plain" {
		t.Fatalf("expected /plain, got %q", got)
	}
}

func TestRoutePatternOrPathUsesChiRoutePattern(t *testing.T) {
	r := chi.NewRouter()
	var captured string
	r.Get("/items/{id}", func(w http.ResponseWriter, req *http.Request) {
		captured = routePatternOrPath(req)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/items/123", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if captured != "/items/{id}" {
		t.Fatalf("expected route pattern /items/{id}, got %q", captured)
	}
}

func TestMiddlewarePassesThroughStatusCode(t *testing.T) {
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	req := httptest.NewRequest(http.MethodGet, "/brew", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTeapot {
		t.Fatalf("expected status %d, got %d", http.StatusTeapot, rec.Code)
	}
}

func TestShouldSkipTracing(t *testing.T) {
	if !shouldSkipTracing("/metrics", "/metrics") {
		t.Fatal("expected /metrics to be skipped")
	}
	if shouldSkipTracing("/ping", "/ping") {
		t.Fatal("did not expect /ping to be skipped")
	}
}

func TestInitTracerSuccess(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "goapi-test")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318/v1/traces")
	t.Setenv("OTEL_TRACES_SAMPLER_ARG", "0.5")

	shutdown, err := InitTracer(context.Background())
	if err != nil {
		t.Fatalf("expected tracer init success, got %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected shutdown function, got nil")
	}
	if err = shutdown(context.Background()); err != nil {
		t.Fatalf("expected tracer shutdown success, got %v", err)
	}
}

func TestInitTracerInvalidEndpointDoesNotPanic(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "://invalid-url")

	shutdown, err := InitTracer(context.Background())
	if err != nil {
		t.Fatalf("expected tracer init to return gracefully, got %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected shutdown function, got nil")
	}
	if err = shutdown(context.Background()); err != nil {
		t.Fatalf("expected tracer shutdown success, got %v", err)
	}
}
