package middleware

import (
	"context"
	"crypto/ed25519"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestCORSMiddlewareHandlesPreflightBeforeAuth(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate ed25519 key: %v", err)
	}

	handler := CORSMiddleware()(AuthMiddleware(publicKey)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/notification/list?limit=20", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	req.Header.Set("Access-Control-Request-Headers", "authorization,content-type")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected preflight to return 204, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("expected allow origin header, got %q", got)
	}
}

func TestCORSMiddlewareAllowsAccountDeviceHeaders(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate ed25519 key: %v", err)
	}

	handler := CORSMiddleware()(AuthMiddleware(publicKey)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "content-type,x-device-uid,x-device-name,x-device-type,x-device-os-name,x-device-os-version,x-device-app-version")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected preflight to return 204, got %d", rec.Code)
	}

	got := rec.Header().Get("Access-Control-Allow-Headers")
	for _, expected := range []string{
		"X-Device-UID",
		"X-Device-Name",
		"X-Device-Type",
		"X-Device-OS-Name",
		"X-Device-OS-Version",
		"X-Device-App-Version",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("expected allow headers to contain %s, got %q", expected, got)
		}
	}
}

func TestCORSMiddlewareAddsHeadersToUnauthorizedResponse(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate ed25519 key: %v", err)
	}

	handler := CORSMiddleware()(AuthMiddleware(publicKey)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notification/list?limit=20", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized response, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("expected allow origin header, got %q", got)
	}
}

func TestCORSMiddlewareDoesNotDuplicateOriginWhenDownstreamAlreadySetCORS(t *testing.T) {
	handler := CORSMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		ApplyCORSHeaders(w.Header(), "http://localhost:5173")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notification/list?limit=20", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	values := rec.Header().Values("Access-Control-Allow-Origin")
	if len(values) != 1 {
		t.Fatalf("expected a single allow origin header value, got %v", values)
	}
	if values[0] != "http://localhost:5173" {
		t.Fatalf("expected allow origin header to match request origin, got %q", values[0])
	}
}

func TestStripCORSHeadersRemovesUpstreamCORSValues(t *testing.T) {
	header := http.Header{}
	header.Set("Access-Control-Allow-Origin", "*")
	header.Set("Access-Control-Allow-Methods", "GET")
	header.Set("Access-Control-Allow-Headers", "Content-Type")
	header.Set("Access-Control-Expose-Headers", "X-Test")
	header.Set("Access-Control-Max-Age", "60")

	StripCORSHeaders(header)

	if got := header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected allow origin to be removed, got %q", got)
	}
	if got := header.Get("Access-Control-Allow-Methods"); got != "" {
		t.Fatalf("expected allow methods to be removed, got %q", got)
	}
	if got := header.Get("Access-Control-Allow-Headers"); got != "" {
		t.Fatalf("expected allow headers to be removed, got %q", got)
	}
	if got := header.Get("Access-Control-Expose-Headers"); got != "" {
		t.Fatalf("expected expose headers to be removed, got %q", got)
	}
	if got := header.Get("Access-Control-Max-Age"); got != "" {
		t.Fatalf("expected max age to be removed, got %q", got)
	}
}
