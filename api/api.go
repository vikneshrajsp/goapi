package api

import (
	"encoding/json"
	"net/http"
)

type CoinBalanceParams struct {
	UserName string `json:"username" schema:"username"`
}

type CoinBalanceResponse struct {
	Code    int   `json:"code"`
	Balance int64 `json:"balance"`
}

type UpdateCoinBalanceRequest struct {
	Balance int64 `json:"balance"`
}

type UpdateCoinBalanceResponse struct {
	Code     int    `json:"code"`
	Username string `json:"username"`
	Balance  int64  `json:"balance"`
}

type SetWebhookRequest struct {
	WebhookURL string `json:"webhook_url"`
}

type SetWebhookResponse struct {
	Code       int    `json:"code"`
	Username   string `json:"username"`
	WebhookURL string `json:"webhook_url"`
}

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func writeErrorResponse(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{Code: code, Message: message})
}

var (
	RequestErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		writeErrorResponse(w, http.StatusBadRequest, err.Error())
	}
	ResponseErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		writeErrorResponse(w, http.StatusInternalServerError, err.Error())
	}
	NotFoundErrorHandler = func(w http.ResponseWriter, r *http.Request) {
		writeErrorResponse(w, http.StatusNotFound, "not found")
	}
	MethodNotAllowedErrorHandler = func(w http.ResponseWriter, r *http.Request) {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
	}
	InternalServerErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		writeErrorResponse(w, http.StatusInternalServerError, err.Error())
	}
)
