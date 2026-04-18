package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	log "github.com/sirupsen/logrus"
	"goapi/api"
	"goapi/internal/database"
)

func updateCoinBalance(repo database.Repository) http.HandlerFunc {
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
