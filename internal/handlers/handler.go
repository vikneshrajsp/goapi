package handlers

import (
	"goapi/internal/database"
	middleware "goapi/internal/middleware"

	"github.com/go-chi/chi"
)

// NewHandler registers HTTP routes. Pass a fully configured [database.Repository].
func NewHandler(r *chi.Mux, repo database.Repository, deps Deps) {
	r.Route("/account", func(r chi.Router) {
		r.Use(middleware.Authorize(repo))
		r.Get("/coins", getCoinBalance(repo))
		r.Put("/coins", updateCoinBalance(repo, deps.Publisher))
		r.Put("/webhook", setWebhookURL(repo))
	})

	r.Get("/ping", PingHandler)
	r.Get("/health", HealthHandler(deps.KafkaHealth))
}
