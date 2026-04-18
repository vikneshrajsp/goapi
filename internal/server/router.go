package server

import (
	"goapi/internal/database"
	"goapi/internal/handlers"
	"goapi/internal/observability"

	"github.com/go-chi/chi"
	chiMiddleware "github.com/go-chi/chi/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewRouter wires Chi with shared dependencies.
func NewRouter(repo database.Repository) *chi.Mux {
	r := chi.NewRouter()
	r.Use(chiMiddleware.StripSlashes)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(observability.Middleware)

	handlers.NewHandler(r, repo)
	r.Handle("/metrics", promhttp.Handler())

	return r
}
