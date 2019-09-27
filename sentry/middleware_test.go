package sentry

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getsentry/sentry-go"
	"github.com/spothero/tools/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPMiddleware(t *testing.T) {
	testHandler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		assert.True(t, sentry.HasHubOnContext(r.Context()))
		hub := sentry.GetHubFromContext(r.Context())
		// there's no way to get the contents of scope, so just make sure the hub isn't empty to verify the
		// trace id is on it
		clone := hub.Clone()
		clone.Scope().Clear()
		assert.NotEqual(t, clone.Scope(), hub.Scope())
	})

	testServer := httptest.NewServer(tracing.HTTPMiddleware(HTTPMiddleware(testHandler)))
	defer testServer.Close()
	res, err := http.Get(testServer.URL)
	require.NoError(t, err)
	require.NotNil(t, res)
	res.Body.Close()
}
