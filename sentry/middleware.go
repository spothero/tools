package sentry

import (
	"net/http"

	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/spothero/tools/log"
)

// This middleware is a wrapper around the sentry-go library's middleware that attaches a the Sentry hub that
// from the request context to the logger. That way, if the logger ever writes an error log, instead of just sending
// the log message and fields provided to the logger to Sentry, Sentry is able to capture the entire request context
// i.e. the request path, headers present, etc.
func HTTPMiddleware(next http.Handler) http.Handler {
	sentryHandler := sentryhttp.New(sentryhttp.Options{
		Repanic:         true,
		WaitForDelivery: true,
		Timeout:         flushTimeout,
	})
	return sentryHandler.HandleFunc(func(w http.ResponseWriter, r *http.Request) {
		hub := sentry.GetHubFromContext(r.Context())
		ctx := log.NewContext(r.Context(), log.Get(r.Context()).With(Hub(hub)))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
