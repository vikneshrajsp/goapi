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
	"goapi/internal/database"
	mkafka "goapi/internal/messaging/kafka"
	"goapi/internal/observability"
	"goapi/internal/workers/notifier"
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

	repo, err := database.New(initCtx)
	if err != nil {
		log.Fatalf("database init: %v", err)
	}
	defer closeRepository(repo)

	topic := os.Getenv("KAFKA_TOPIC_COINBALANCE")
	if topic == "" {
		topic = mkafka.DefaultTopic
	}
	consumer := mkafka.NewConsumer(os.Getenv("KAFKA_BROKERS"), topic, "coinbalance-notifier")
	defer consumer.Close()

	worker := notifier.New(repo)
	go func() {
		if err := consumer.Run(ctx, worker.Handle); err != nil {
			log.Fatalf("consumer run: %v", err)
		}
	}()

	httpServer := &http.Server{
		Addr: ":8091",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			healthCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := consumer.HealthCheck(healthCtx); err != nil {
				http.Error(w, "kafka unavailable", http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("health server: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
}

func closeRepository(repo database.Repository) {
	type closer interface{ Close() }
	if c, ok := repo.(closer); ok {
		c.Close()
	}
}
