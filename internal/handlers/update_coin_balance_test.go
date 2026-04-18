package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"goapi/api"
	"goapi/internal/tools"
)

func TestUpdateCoinBalanceMissingUsername(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/account/coins", bytes.NewBufferString(`{"balance":100}`))
	rec := httptest.NewRecorder()

	UpdateCoinBalance(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestUpdateCoinBalanceInvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/account/coins?username=alex", bytes.NewBufferString("{"))
	rec := httptest.NewRecorder()

	UpdateCoinBalance(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestUpdateCoinBalanceSuccess(t *testing.T) {
	db, err := tools.NewDatabase()
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	original, err := db.GetCoinDetails("alex")
	if err != nil {
		t.Fatalf("failed to read original balance: %v", err)
	}

	request := api.UpdateCoinBalanceRequest{Balance: 555}
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/account/coins?username=alex", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	UpdateCoinBalance(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response api.UpdateCoinBalanceResponse
	if err = json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.Balance != 555 {
		t.Fatalf("expected updated balance 555, got %d", response.Balance)
	}

	_, restoreErr := db.UpdateCoinDetails("alex", original.Coins)
	if restoreErr != nil {
		t.Fatalf("failed to restore original balance: %v", restoreErr)
	}
}

func TestUpdateCoinBalanceNegativeBalance(t *testing.T) {
	request := api.UpdateCoinBalanceRequest{Balance: -1}
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/account/coins?username=alex", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	UpdateCoinBalance(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}
