package handlers

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
	"goapi/api"
	tools "goapi/internal/tools"
)

func UpdateCoinBalance(w http.ResponseWriter, r *http.Request) {
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

	database, err := tools.NewDatabase()
	if err != nil {
		log.Error(err.Error())
		api.InternalServerErrorHandler(w, r, err)
		return
	}

	coinDetails, err := database.UpdateCoinDetails(username, request.Balance)
	if err != nil {
		log.Error(err.Error())
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
	if err = json.NewEncoder(w).Encode(response); err != nil {
		log.Error(err.Error())
		api.InternalServerErrorHandler(w, r, err)
		return
	}
}
