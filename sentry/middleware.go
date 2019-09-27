package sentry

import (
	"net/http"

	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/spothero/tools/log"
)

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
