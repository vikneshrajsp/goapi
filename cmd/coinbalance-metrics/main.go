package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"goapi/internal/config"
	mkafka "goapi/internal/messaging/kafka"
	"goapi/internal/observability"
	"goapi/internal/workers/eventmetrics"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	initCtx := context.Background()
	shutdownTracer, err := observability.InitTracer(initCtx)
	if err != nil {
		log.Fatalf("OpenTelemetry tracer: %v", err)
	}
	defer func() {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownErr := shutdownTracer(timeoutCtx); shutdownErr != nil {
			log.Errorf("OpenTelemetry shutdown: %v", shutdownErr)
		}
	}()

	log.SetReportCaller(true)
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(config.ResolveLogLevel())
	config.ConfigureLogOutput()

	topic := os.Getenv("KAFKA_TOPIC_COINBALANCE")
	if topic == "" {
		topic = mkafka.DefaultTopic
	}
	consumer := mkafka.NewConsumer(os.Getenv("KAFKA_BROKERS"), topic, "coinbalance-metrics")
	defer consumer.Close()

	worker := eventmetrics.New()
	go func() {
		if err := consumer.Run(ctx, worker.Handle); err != nil {
			log.Fatalf("consumer run: %v", err)
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", eventmetrics.HealthHandler(consumer))

	httpServer := &http.Server{
		Addr:              ":8092",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("metrics server: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
}
