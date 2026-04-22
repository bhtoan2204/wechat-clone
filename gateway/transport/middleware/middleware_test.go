package middleware

import (
	"context"
	"crypto/ed25519"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gateway/pkg/idempotency"

	"github.com/o1egl/paseto"
)

type stubIdempotencyStore struct {
	tryLockCalls int
	lockedKeys   []string
}

func (s *stubIdempotencyStore) TryLock(_ context.Context, key string, _ time.Duration) (bool, error) {
	s.tryLockCalls++
	s.lockedKeys = append(s.lockedKeys, key)
	return true, nil
}

func (s *stubIdempotencyStore) MarkDone(context.Context, string, time.Duration) error {
	return nil
}

func (s *stubIdempotencyStore) Release(context.Context, string) error {
	return nil
}

func TestAuthMiddlewareAllowsPaymentWebhookWithoutToken(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate ed25519 key: %v", err)
	}

	handler := AuthMiddleware(publicKey)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment/webhooks/stripe", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected payment webhook to bypass auth, got status %d", rec.Code)
	}
}

func TestAuthMiddlewareRejectsProtectedRouteWithoutToken(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate ed25519 key: %v", err)
	}

	handler := AuthMiddleware(publicKey)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/account/profile", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected protected route to require auth, got status %d", rec.Code)
	}
}

func TestAuthMiddlewareAllowsProtectedRouteWithValidToken(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate ed25519 key: %v", err)
	}

	token, err := paseto.NewV2().Sign(privateKey, paseto.JSONToken{}, nil)
	if err != nil {
		t.Fatalf("sign paseto token: %v", err)
	}

	handler := AuthMiddleware(publicKey)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/account/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected valid token to pass auth, got status %d", rec.Code)
	}
}

func TestIdempotencyMiddlewareSkipsWhenHeaderMissing(t *testing.T) {
	store := &stubIdempotencyStore{}
	manager := idempotency.NewManager(store, time.Minute, time.Hour)

	handler := IdempotencyMiddleware(manager)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/messages", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected request without idempotency key to pass through, got status %d", rec.Code)
	}
	if store.tryLockCalls != 0 {
		t.Fatalf("expected no idempotency lock attempt when header is missing, got %d", store.tryLockCalls)
	}
}

func TestClientIPUsesRemoteAddrInsteadOfForwardedHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health-check", nil)
	req.RemoteAddr = "198.51.100.7:43210"
	req.Header.Set("X-Forwarded-For", "203.0.113.9")
	req.Header.Set("X-Real-IP", "203.0.113.10")

	got := clientIP(req)

	if got != "198.51.100.7" {
		t.Fatalf("expected remote addr IP, got %q", got)
	}
}
