package middleware

import (
	"gateway/pkg/idempotency"
	"net/http"
	"strings"
)

const idempotencyHeader = "Idempotency-Key"

func IdempotencyMiddleware(manager *idempotency.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if manager == nil {
				next.ServeHTTP(w, r)
				return
			}

			if !isWriteMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			key := strings.TrimSpace(r.Header.Get(idempotencyHeader))
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			ok, err := manager.Begin(r.Context(), key)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"` + err.Error() + `"}`))
				return
			}

			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"error":"duplicate request"}`))
				return
			}

			rw := &statusRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(rw, r)

			status := rw.statusCode
			success := status >= http.StatusOK && status < http.StatusMultipleChoices
			_ = manager.End(r.Context(), key, success)
		})
	}
}

func isWriteMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.statusCode == 0 {
		r.statusCode = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}
