package observability

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	otrace "go.opentelemetry.io/otel/trace"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "goapi_http_requests_total",
			Help: "Total number of HTTP requests by route, method and status code.",
		},
		[]string{"route", "method", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "goapi_http_request_duration_seconds",
			Help:    "HTTP request duration by route and method.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"route", "method"},
	)
)

func InitTracer(ctx context.Context) (func(context.Context) error, error) {
	serviceName := envOrDefault("OTEL_SERVICE_NAME", "goapi")
	endpoint := envOrDefault("OTEL_EXPORTER_OTLP_ENDPOINT", "http://jaeger:4318")

	exporter, err := otlptracehttp.New(
		ctx,
		otlptracehttp.WithEndpointURL(endpoint),
	)
	if err != nil {
		return nil, err
	}

	sampleRatio := parseFloatEnv("OTEL_TRACES_SAMPLER_ARG", 1.0)
	traceProvider := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithSampler(trace.ParentBased(trace.TraceIDRatioBased(sampleRatio))),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		)),
	)

	otel.SetTracerProvider(traceProvider)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return traceProvider.Shutdown, nil
}

func Middleware(next http.Handler) http.Handler {
	tracer := otel.Tracer("goapi/http")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := routePatternOrPath(r)
		if shouldSkipTracing(route, r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		spanName := fmt.Sprintf("%s %s", r.Method, route)
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		ctx, span := tracer.Start(
			ctx,
			spanName,
			otrace.WithSpanKind(otrace.SpanKindServer),
			otrace.WithAttributes(
				semconv.HTTPRequestMethodKey.String(r.Method),
				semconv.URLPathKey.String(route),
			),
		)
		defer span.End()

		r = r.WithContext(ctx)
		recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(recorder, r)
		duration := time.Since(start).Seconds()
		statusCode := strconv.Itoa(recorder.statusCode)

		span.SetAttributes(semconv.HTTPResponseStatusCodeKey.Int(recorder.statusCode))
		if recorder.statusCode >= http.StatusInternalServerError {
			span.SetStatus(codes.Error, http.StatusText(recorder.statusCode))
		}

		httpRequestsTotal.WithLabelValues(route, r.Method, statusCode).Inc()
		httpRequestDuration.WithLabelValues(route, r.Method).Observe(duration)
	})
}

func shouldSkipTracing(route string, path string) bool {
	normalizedRoute := strings.TrimSpace(route)
	normalizedPath := strings.TrimSpace(path)
	return normalizedRoute == "/metrics" || normalizedPath == "/metrics"
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.statusCode = code
	s.ResponseWriter.WriteHeader(code)
}

func routePatternOrPath(r *http.Request) string {
	routePattern := ""
	if routeContext := chi.RouteContext(r.Context()); routeContext != nil {
		routePattern = routeContext.RoutePattern()
	}
	routePattern = strings.TrimSpace(routePattern)
	if routePattern == "" {
		return r.URL.Path
	}
	return routePattern
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func parseFloatEnv(key string, fallback float64) float64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}

	return parsed
}
