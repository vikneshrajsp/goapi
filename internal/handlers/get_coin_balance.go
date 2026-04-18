package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/schema"
	log "github.com/sirupsen/logrus"
	"goapi/api"
	"goapi/internal/database"
)

var ErrUsernameRequired = errors.New("username is required")

func getCoinBalance(repo database.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		coinBalanceParams := api.CoinBalanceParams{}
		decoder := schema.NewDecoder()
		if err := decoder.Decode(&coinBalanceParams, r.URL.Query()); err != nil {
			log.Error(err.Error())
			api.InternalServerErrorHandler(w, r, err)
			return
		}
		if coinBalanceParams.UserName == "" {
			api.RequestErrorHandler(w, r, ErrUsernameRequired)
			return
		}

		coinDetails, err := repo.GetCoinDetails(r.Context(), coinBalanceParams.UserName)
		if err != nil {
			log.Error(err.Error())
			if errors.Is(err, database.ErrUserNotFound) {
				api.RequestErrorHandler(w, r, err)
				return
			}
			api.InternalServerErrorHandler(w, r, err)
			return
		}

		response := api.CoinBalanceResponse{
			Code:    200,
			Balance: coinDetails.Coins,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err = json.NewEncoder(w).Encode(response); err != nil {
			log.Error(err.Error())
			api.InternalServerErrorHandler(w, r, err)
		}
	}
}
