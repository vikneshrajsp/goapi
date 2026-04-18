package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"

	log "github.com/sirupsen/logrus"
	"goapi/api"
	"goapi/internal/database"
)

func setWebhookURL(repo database.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.URL.Query().Get("username")
		if username == "" {
			api.RequestErrorHandler(w, r, ErrUsernameRequired)
			return
		}

		var req api.SetWebhookRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			api.RequestErrorHandler(w, r, err)
			return
		}
		u, err := url.ParseRequestURI(req.WebhookURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			api.RequestErrorHandler(w, r, errInvalidWebhookURL)
			return
		}

		if err := repo.SetUserWebhookURL(r.Context(), username, req.WebhookURL); err != nil {
			log.Error(err)
			api.RequestErrorHandler(w, r, err)
			return
		}

		resp := api.SetWebhookResponse{
			Code:       http.StatusOK,
			Username:   username,
			WebhookURL: req.WebhookURL,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}
