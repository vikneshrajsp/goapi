package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"goapi/api"
)

func TestGetCoinBalanceMissingUsername(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/account/coins", nil)
	rec := httptest.NewRecorder()

	GetCoinBalance(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestGetCoinBalanceSuccess(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/account/coins?username=alex", nil)
	rec := httptest.NewRecorder()

	GetCoinBalance(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response api.CoinBalanceResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.Balance <= 0 {
		t.Fatalf("expected positive balance, got %d", response.Balance)
	}
}

func TestGetCoinBalanceUnknownUser(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/account/coins?username=ghost", nil)
	rec := httptest.NewRecorder()

	GetCoinBalance(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}
