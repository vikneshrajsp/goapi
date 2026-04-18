package handlers

import (
	middleware "goapi/internal/middleware"

	"github.com/go-chi/chi"
)

func NewHandler(r *chi.Mux) {
	r.Route("/account", func(r chi.Router) {
		r.Use(middleware.AuthorizeHandler)
		r.Get("/coins", GetCoinBalance)
		r.Put("/coins", UpdateCoinBalance)
	})

	r.Get("/ping", PingHandler)
	r.Get("/health", HealthHandler)
}
