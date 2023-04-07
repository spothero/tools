package sentry

import (
	"go.opentelemetry.io/otel/trace"
	"net/http"

	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/spothero/tools/log"
)

// Middleware contains a Sentry handler
type Middleware struct {
	sentryHandler *sentryhttp.Handler
}

// NewMiddleware creates a new Sentry middleware object
func NewMiddleware() Middleware {
	return Middleware{
		sentryhttp.New(sentryhttp.Options{
			Repanic:         true,
			WaitForDelivery: true,
			Timeout:         flushTimeout,
		}),
	}
}

// HTTP is a wrapper around the sentry-go library's middleware that attaches a Sentry hub that
// from the request context to the logger. That way, if the logger ever writes an error log, instead of just sending
// the log message and fields provided to the logger to Sentry, Sentry is able to capture the entire request context
// i.e. the request path, headers present, etc. If this middleware is attached after the tracing middleware,
// the corresponding Trace ID will be added to the Sentry scope.
func (m Middleware) HTTP(next http.Handler) http.Handler {
	return m.sentryHandler.HandleFunc(func(w http.ResponseWriter, r *http.Request) {
		hub := sentry.GetHubFromContext(r.Context())
		hub.ConfigureScope(func(scope *sentry.Scope) {
			if span := trace.SpanFromContext(r.Context()); span != nil {
				sc := span.SpanContext()
				scope.SetTag("correlation_id", sc.TraceID().String())
			}
		})
		ctx := log.NewContext(r.Context(), log.Get(r.Context()).With(Hub(hub)))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
