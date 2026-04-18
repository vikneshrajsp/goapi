package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"goapi/api"
	"goapi/internal/database"
)

type webhookRepoErr struct{}

func (webhookRepoErr) Setup(context.Context) error { return nil }
func (webhookRepoErr) GetLoginDetails(context.Context, string) (*database.LoginDetails, error) {
	return nil, errors.New("nope")
}
func (webhookRepoErr) GetCoinDetails(context.Context, string) (*database.CoinDetails, error) {
	return nil, errors.New("nope")
}
func (webhookRepoErr) UpdateCoinDetails(context.Context, string, int64) (*database.CoinDetails, error) {
	return nil, errors.New("nope")
}
func (webhookRepoErr) SetUserWebhookURL(context.Context, string, string) error {
	return errors.New("db error")
}
func (webhookRepoErr) GetUserWebhookURL(context.Context, string) (string, error) {
	return "", errors.New("nope")
}

func TestSetWebhookURLSuccess(t *testing.T) {
	body, _ := json.Marshal(api.SetWebhookRequest{WebhookURL: "https://example.com/hook"})
	req := httptest.NewRequest(http.MethodPut, "/account/webhook?username=alex", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	setWebhookURL(mockRepo(t))(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 got %d", rec.Code)
	}
}

func TestSetWebhookURLValidation(t *testing.T) {
	body, _ := json.Marshal(api.SetWebhookRequest{WebhookURL: "notaurl"})
	req := httptest.NewRequest(http.MethodPut, "/account/webhook?username=alex", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	setWebhookURL(mockRepo(t))(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 got %d", rec.Code)
	}
}

func TestSetWebhookURLMissingUsername(t *testing.T) {
	body, _ := json.Marshal(api.SetWebhookRequest{WebhookURL: "https://example.com/hook"})
	req := httptest.NewRequest(http.MethodPut, "/account/webhook", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	setWebhookURL(mockRepo(t))(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 got %d", rec.Code)
	}
}

func TestSetWebhookURLRepoError(t *testing.T) {
	body, _ := json.Marshal(api.SetWebhookRequest{WebhookURL: "https://example.com/hook"})
	req := httptest.NewRequest(http.MethodPut, "/account/webhook?username=alex", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	setWebhookURL(webhookRepoErr{})(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 got %d", rec.Code)
	}
}
