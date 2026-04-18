package server

import (
	"goapi/internal/handlers"
	"goapi/internal/observability"

	"github.com/go-chi/chi"
	chiMiddleware "github.com/go-chi/chi/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(chiMiddleware.StripSlashes)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(observability.Middleware)

	handlers.NewHandler(r)
	r.Handle("/metrics", promhttp.Handler())

	return r
}
