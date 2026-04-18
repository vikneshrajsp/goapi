package handlers

import (
	"context"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

func HealthHandler(kafka HealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if kafka != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
			defer cancel()
			if err := kafka.HealthCheck(ctx); err != nil {
				log.Errorf("kafka health check failed: %v", err)
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte("kafka unavailable"))
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}
