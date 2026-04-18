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
	"goapi/internal/database"
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

	repo, err := database.New(ctx)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer closeRepository(repo)

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
		Handler:           server.NewRouter(repo),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if serveErr := httpServer.ListenAndServe(); serveErr != nil && serveErr != http.ErrServerClosed {
			log.Fatalf("server failed: %v", serveErr)
		}
	}()

	waitForShutdown(httpServer)
}

func closeRepository(repo database.Repository) {
	type closer interface {
		Close()
	}
	if c, ok := repo.(closer); ok {
		c.Close()
		return
	}
	type ioCloser interface {
		Close() error
	}
	if c, ok := repo.(ioCloser); ok {
		if err := c.Close(); err != nil {
			log.Errorf("database close: %v", err)
		}
	}
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
