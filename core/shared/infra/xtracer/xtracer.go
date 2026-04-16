package xtracer

import (
	"context"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "go-socket"

func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.Tracer(instrumentationName).Start(ctx, name, opts...)
}

func WrapTransport(t http.RoundTripper) http.RoundTripper {
	return otelhttp.NewTransport(t)
}

func NewTransport() http.RoundTripper {
	return WrapTransport(http.DefaultTransport)
}
