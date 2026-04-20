package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"goapi/api"
	"goapi/internal/database"
	"goapi/internal/messaging/events"

	"github.com/google/uuid"
)

func updateCoinBalance(repo database.Repository, publisher EventPublisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.URL.Query().Get("username")
		if username == "" {
			api.RequestErrorHandler(w, r, ErrUsernameRequired)
			return
		}

		var request api.UpdateCoinBalanceRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			log.Error(err.Error())
			api.RequestErrorHandler(w, r, err)
			return
		}

		if request.Balance < 0 {
			api.RequestErrorHandler(w, r, errors.New("balance cannot be negative"))
			return
		}

		prev, err := repo.GetCoinDetails(r.Context(), username)
		if err != nil {
			log.Error(err.Error())
			api.RequestErrorHandler(w, r, err)
			return
		}

		coinDetails, err := repo.UpdateCoinDetails(r.Context(), username, request.Balance)
		if err != nil {
			log.Error(err.Error())
			if errors.Is(err, database.ErrUserNotFound) {
				api.RequestErrorHandler(w, r, err)
				return
			}
			api.RequestErrorHandler(w, r, err)
			return
		}

		if publisher != nil {
			event := events.CoinBalanceChanged{
				SchemaVersion: 1,
				EventID:       uuid.NewString(),
				EventType:     events.CoinBalanceChangedType,
				Username:      coinDetails.Username,
				Previous:      prev.Coins,
				Current:       coinDetails.Coins,
				Delta:         coinDetails.Coins - prev.Coins,
				OccurredAt:    time.Now().UTC(),
			}
			publishCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			if err = publisher.PublishCoinBalanceChanged(publishCtx, event); err != nil {
				log.Errorf("failed to publish coin balance event: %v", err)
				api.InternalServerErrorHandler(w, r, err)
				return
			}
		}

		response := api.UpdateCoinBalanceResponse{
			Code:     http.StatusOK,
			Username: coinDetails.Username,
			Balance:  coinDetails.Coins,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error(err.Error())
			api.InternalServerErrorHandler(w, r, err)
		}
	}
}
