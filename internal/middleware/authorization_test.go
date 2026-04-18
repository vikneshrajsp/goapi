package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthorizeHandlerUnauthorizedWithoutHeaders(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/account/coins", nil)
	rec := httptest.NewRecorder()

	Authorize(mockRepo(t))(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if nextCalled {
		t.Fatal("next handler should not be called when request is unauthorized")
	}
}

func TestAuthorizeHandlerAuthorized(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/account/coins?username=alex", nil)
	req.Header.Set("Authorization", "123AL100")
	rec := httptest.NewRecorder()

	Authorize(mockRepo(t))(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !nextCalled {
		t.Fatal("next handler should be called for authorized request")
	}
}

func TestAuthorizeHandlerInvalidToken(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/account/coins?username=alex", nil)
	req.Header.Set("Authorization", "bad-token")
	rec := httptest.NewRecorder()

	Authorize(mockRepo(t))(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if nextCalled {
		t.Fatal("next handler should not be called with invalid token")
	}
}

func TestAuthorizeHandlerUnknownUser(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/account/coins?username=ghost", nil)
	req.Header.Set("Authorization", "123AL100")
	rec := httptest.NewRecorder()

	Authorize(mockRepo(t))(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if nextCalled {
		t.Fatal("next handler should not be called for unknown user")
	}
}
