package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"goapi/internal/config"
	"goapi/internal/observability"
	"goapi/internal/server"

	log "github.com/sirupsen/logrus"
)

func main() {
	ctx := context.Background()

	log.SetReportCaller(true)
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(config.ResolveLogLevel())
	config.ConfigureLogOutput()

	shutdownTracer, err := observability.InitTracer(ctx)
	if err != nil {
		log.Fatalf("failed to initialize OpenTelemetry tracer: %v", err)
	}
	defer func() {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownErr := shutdownTracer(timeoutCtx); shutdownErr != nil {
			log.Errorf("failed to shutdown OpenTelemetry tracer: %v", shutdownErr)
		}
	}()

	fmt.Println("Starting server on port 8080")
	log.Info("Starting server on port 8080")

	httpServer := &http.Server{
		Addr:              ":8080",
		Handler:           server.NewRouter(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if serveErr := httpServer.ListenAndServe(); serveErr != nil && serveErr != http.ErrServerClosed {
			log.Fatalf("server failed: %v", serveErr)
		}
	}()

	waitForShutdown(httpServer)
}

func waitForShutdown(server *http.Server) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Errorf("server shutdown failed: %v", err)
	}
}
