//go:build integration

package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"goapi/api"

	"github.com/go-chi/chi"
)

func setupIntegrationServer() *httptest.Server {
	r := chi.NewRouter()
	NewHandler(r)
	return httptest.NewServer(r)
}

func TestIntegrationGetCoinBalance(t *testing.T) {
	server := setupIntegrationServer()
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/account/coins?username=alex", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "123AL100")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to call endpoint: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.StatusCode)
	}

	var response api.CoinBalanceResponse
	if err = json.NewDecoder(res.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Balance <= 0 {
		t.Fatalf("expected balance greater than 0, got %d", response.Balance)
	}
}

func TestIntegrationUpdateCoinBalance(t *testing.T) {
	server := setupIntegrationServer()
	defer server.Close()

	payload := api.UpdateCoinBalanceRequest{Balance: 777}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest(http.MethodPut, server.URL+"/account/coins?username=alex", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "123AL100")
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to call update endpoint: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.StatusCode)
	}

	var updateResponse api.UpdateCoinBalanceResponse
	if err = json.NewDecoder(res.Body).Decode(&updateResponse); err != nil {
		t.Fatalf("failed to decode update response: %v", err)
	}
	if updateResponse.Balance != 777 {
		t.Fatalf("expected updated balance 777, got %d", updateResponse.Balance)
	}

	// Reset state so integration tests remain deterministic.
	resetPayload := api.UpdateCoinBalanceRequest{Balance: 100}
	resetBody, _ := json.Marshal(resetPayload)
	resetReq, _ := http.NewRequest(http.MethodPut, server.URL+"/account/coins?username=alex", bytes.NewReader(resetBody))
	resetReq.Header.Set("Authorization", "123AL100")
	resetReq.Header.Set("Content-Type", "application/json")
	_, _ = http.DefaultClient.Do(resetReq)
}

func TestIntegrationPingHandler(t *testing.T) {
	server := setupIntegrationServer()
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/ping", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to call ping endpoint: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.StatusCode)
	}

	body, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		t.Fatalf("failed to read ping response body: %v", readErr)
	}
	if strings.TrimSpace(string(body)) != "pong" {
		t.Fatalf("expected pong response, got %q", string(body))
	}
}

func TestIntegrationHealthHandler(t *testing.T) {
	server := setupIntegrationServer()
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/health", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to call health endpoint: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.StatusCode)
	}

	body, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		t.Fatalf("failed to read health response body: %v", readErr)
	}
	if strings.TrimSpace(string(body)) != "ok" {
		t.Fatalf("expected ok response, got %q", string(body))
	}
}
