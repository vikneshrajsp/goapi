package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"goapi/api"
	"goapi/internal/database"
)

type repoTransientErr struct{}

func (repoTransientErr) Setup(context.Context) error { return nil }

func (repoTransientErr) GetLoginDetails(context.Context, string) (*database.LoginDetails, error) {
	return nil, fmt.Errorf("login unavailable")
}

func (repoTransientErr) GetCoinDetails(context.Context, string) (*database.CoinDetails, error) {
	return nil, fmt.Errorf("db unavailable")
}

func (repoTransientErr) UpdateCoinDetails(context.Context, string, int64) (*database.CoinDetails, error) {
	return nil, fmt.Errorf("db unavailable")
}

func (repoTransientErr) SetUserWebhookURL(context.Context, string, string) error {
	return fmt.Errorf("db unavailable")
}

func (repoTransientErr) GetUserWebhookURL(context.Context, string) (string, error) {
	return "", fmt.Errorf("db unavailable")
}

func TestGetCoinBalanceInternalServerError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/account/coins?username=alex", nil)
	rec := httptest.NewRecorder()

	getCoinBalance(repoTransientErr{})(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestUpdateCoinBalanceRepoError(t *testing.T) {
	body, err := json.Marshal(api.UpdateCoinBalanceRequest{Balance: 10})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPut, "/account/coins?username=alex", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	updateCoinBalance(repoTransientErr{}, testDeps().Publisher)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d (RequestErrorHandler), got %d", http.StatusBadRequest, rec.Code)
	}
}
