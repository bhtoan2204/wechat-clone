package middleware

import (
	"net/http"
	"strings"
)

const (
	corsAllowMethods  = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	corsAllowHeaders  = "Authorization, Content-Type, Accept, Idempotency-Key, X-Requested-With, X-Device-UID, X-Device-Name, X-Device-Type, X-Device-OS-Name, X-Device-OS-Version, X-Device-App-Version"
	corsExposeHeaders = "Content-Length, Content-Type"
)

func ApplyCORSHeaders(header http.Header, origin string) {
	origin = strings.TrimSpace(origin)
	if header == nil || origin == "" {
		return
	}

	header.Set("Access-Control-Allow-Origin", origin)
	header.Set("Vary", "Origin")
	header.Set("Access-Control-Allow-Methods", corsAllowMethods)
	header.Set("Access-Control-Allow-Headers", corsAllowHeaders)
	header.Set("Access-Control-Expose-Headers", corsExposeHeaders)
	header.Set("Access-Control-Max-Age", "600")
}

func StripCORSHeaders(header http.Header) {
	if header == nil {
		return
	}

	header.Del("Access-Control-Allow-Origin")
	header.Del("Access-Control-Allow-Methods")
	header.Del("Access-Control-Allow-Headers")
	header.Del("Access-Control-Expose-Headers")
	header.Del("Access-Control-Max-Age")
}

func CORSMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			ApplyCORSHeaders(w.Header(), origin)

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
