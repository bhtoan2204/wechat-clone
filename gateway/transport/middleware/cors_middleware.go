package middleware

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strings"
)

var (
	corsAllowMethods = []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodOptions,
	}

	// Generic/common request headers
	corsGenericAllowHeaders = []string{
		"Accept",
		"Accept-Language",
		"Authorization",
		"Content-Type",
		"Origin",
		"Referer",
		"User-Agent",
		"X-Requested-With",
		"Idempotency-Key",
	}

	// Device/app-specific headers
	corsDeviceAllowHeaders = []string{
		"X-Device-UID",
		"X-Device-Name",
		"X-Device-Type",
		"X-Device-OS-Name",
		"X-Device-OS-Version",
		"X-Device-App-Version",
	}

	corsExposeHeaders = []string{
		"Content-Length",
		"Content-Type",
	}
)

func ApplyCORSHeaders(header http.Header, origin string) {
	origin = strings.TrimSpace(origin)
	if header == nil || origin == "" {
		return
	}

	allowHeaders := append(
		append([]string{}, corsGenericAllowHeaders...),
		corsDeviceAllowHeaders...,
	)

	header.Set("Access-Control-Allow-Origin", origin)
	header.Set("Vary", "Origin")
	header.Set("Access-Control-Allow-Methods", strings.Join(corsAllowMethods, ", "))
	header.Set("Access-Control-Allow-Headers", strings.Join(allowHeaders, ", "))
	header.Set("Access-Control-Expose-Headers", strings.Join(corsExposeHeaders, ", "))
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
			if r.Method == http.MethodOptions {
				ApplyCORSHeaders(w.Header(), origin)
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(newCORSResponseWriter(w, origin), r)
		})
	}
}

type corsResponseWriter struct {
	http.ResponseWriter
	origin string
}

func newCORSResponseWriter(w http.ResponseWriter, origin string) *corsResponseWriter {
	return &corsResponseWriter{
		ResponseWriter: w,
		origin:         strings.TrimSpace(origin),
	}
}

func (w *corsResponseWriter) ensureCORSHeaders() {
	if strings.TrimSpace(w.Header().Get("Access-Control-Allow-Origin")) != "" {
		return
	}

	ApplyCORSHeaders(w.Header(), w.origin)
}

func (w *corsResponseWriter) WriteHeader(statusCode int) {
	w.ensureCORSHeaders()
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *corsResponseWriter) Write(p []byte) (int, error) {
	w.ensureCORSHeaders()
	return w.ResponseWriter.Write(p)
}

func (w *corsResponseWriter) Flush() {
	w.ensureCORSHeaders()
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *corsResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not support hijacking")
	}
	return hijacker.Hijack()
}

func (w *corsResponseWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := w.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

func (w *corsResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
