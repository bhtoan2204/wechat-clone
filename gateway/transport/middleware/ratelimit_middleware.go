package middleware

import (
	"gateway/pkg/cache"
	"gateway/pkg/ratelimit"
	"net"
	"net/http"
	"time"
)

func RateLimitMiddleware(ca cache.Cache) func(http.Handler) http.Handler {
	limiter := ratelimit.NewSlidingWindowLimiter(ca, 60, time.Minute)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ca == nil {
				next.ServeHTTP(w, r)
				return
			}

			ip := clientIP(r)

			ok, err := limiter.Allow(r.Context(), ip)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"rate limit exceeded"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}

	return r.RemoteAddr
}
