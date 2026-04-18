package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestErrorHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	RequestErrorHandler(rec, req, errors.New("invalid request"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	var response ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.Message != "invalid request" {
		t.Fatalf("expected invalid request message, got %q", response.Message)
	}
}

func TestInternalServerErrorHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	InternalServerErrorHandler(rec, req, errors.New("boom"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
	var response ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.Message != "boom" {
		t.Fatalf("expected boom message, got %q", response.Message)
	}
}

func TestNotFoundAndMethodNotAllowedHandlers(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	notFoundRec := httptest.NewRecorder()
	NotFoundErrorHandler(notFoundRec, req)
	if notFoundRec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, notFoundRec.Code)
	}

	methodRec := httptest.NewRecorder()
	MethodNotAllowedErrorHandler(methodRec, req)
	if methodRec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, methodRec.Code)
	}
}
