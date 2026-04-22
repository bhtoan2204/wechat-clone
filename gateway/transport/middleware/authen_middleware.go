package middleware

import (
	"crypto/ed25519"
	"net/http"
	"strings"

	"github.com/o1egl/paseto"
)

type publicRouteRule struct {
	exact  string
	prefix string
}

var publicRouteRules = []publicRouteRule{
	{exact: "/api/v1/account/verify-email/confirm"},
	{exact: "/api/v1/auth/login"},
	{exact: "/api/v1/auth/login-google"},
	{exact: "/api/v1/auth/login-google/callback"},
	{exact: "/api/v1/auth/refresh"},
	{exact: "/api/v1/auth/register"},
	{prefix: "/api/v1/payment/webhooks/"},
}

func extractToken(r *http.Request) string {
	token := r.Header.Get("Authorization")
	if token != "" {
		token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))
		if token != "" {
			return token
		}
	}

	token = strings.TrimSpace(r.URL.Query().Get("authorization"))
	if token != "" {
		return token
	}

	return ""
}

func isWhitelisted(path string) bool {
	for _, rule := range publicRouteRules {
		if rule.exact != "" && path == rule.exact {
			return true
		}
		if rule.prefix != "" && strings.HasPrefix(path, rule.prefix) {
			return true
		}
	}
	return false
}

func AuthMiddleware(publicKey ed25519.PublicKey) func(http.Handler) http.Handler {
	parser := paseto.NewV2()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isWhitelisted(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			token := extractToken(r)
			if token == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"Unauthorized"}`))
				return
			}

			if err := parser.Verify(token, publicKey, nil, nil); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"Unauthorized"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
