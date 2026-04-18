package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/schema"
	log "github.com/sirupsen/logrus"
	"goapi/api"
	tools "goapi/internal/tools"
)

var ErrUsernameRequired = errors.New("username is required")

func GetCoinBalance(w http.ResponseWriter, r *http.Request) {
	coinBalanceParams := api.CoinBalanceParams{}
	var decoder *schema.Decoder = schema.NewDecoder()
	error := decoder.Decode(&coinBalanceParams, r.URL.Query())
	if error != nil {
		log.Error(error.Error())
		api.InternalServerErrorHandler(w, r, error)
		return
	}
	if coinBalanceParams.UserName == "" {
		api.RequestErrorHandler(w, r, ErrUsernameRequired)
		return
	}

	database, err := tools.NewDatabase()
	if err != nil {
		log.Error(err.Error())
		api.InternalServerErrorHandler(w, r, err)
		return
	}

	coinDetails, error := database.GetCoinDetails(coinBalanceParams.UserName)
	if error != nil {
		log.Error(error.Error())
		api.InternalServerErrorHandler(w, r, error)
		return
	}

	var response api.CoinBalanceResponse = api.CoinBalanceResponse{
		Code:    200,
		Balance: coinDetails.Coins,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	error = json.NewEncoder(w).Encode(response)
	if error != nil {
		log.Error(error.Error())
		api.InternalServerErrorHandler(w, r, error)
		return
	}
}
