package http

import (
	"fmt"
	"net/http"

	"github.com/spothero/tools/jose"
	"github.com/spothero/tools/log"
	"github.com/spothero/tools/tracing"
)

// ClientMiddleware defines an HTTP Client middleware function. The function is called prior to
// invoking the transport roundtrip, and the returned response function is called after the
// response has been received from the client.
type ClientMiddleware func(*http.Request) (*http.Request, func(*http.Response) error, error)

// MiddlewareRoundTripper implements a proxied net/http RoundTripper so that http requests may be decorated
// with middleware
type MiddlewareRoundTripper struct {
	RoundTripper http.RoundTripper
	Middleware   []ClientMiddleware
}

// NewDefaultClient constructs the default HTTP Client with middleware. Providing an HTTP
// RoundTripper is optional. If `nil` is received, the DefaultClient will be used.
func NewDefaultClient(metrics Metrics, roundTripper http.RoundTripper) http.Client {
	if roundTripper == nil {
		roundTripper = http.DefaultTransport
	}
	return http.Client{
		Transport: MiddlewareRoundTripper{
			Middleware: []ClientMiddleware{
				tracing.HTTPClientMiddleware,
				log.HTTPClientMiddleware,
				metrics.ClientMiddleware,
				jose.HTTPClientMiddleware,
			},
			RoundTripper: roundTripper,
		},
	}
}

// RoundTrip completes the http request round trip and is responsible for invoking HTTP Client
// Middleware
func (mrt MiddlewareRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Ensure the RoundTripper was set on the MiddlewareRoundTripper
	if mrt.RoundTripper == nil {
		return nil, fmt.Errorf("no roundtripper provided to middleware round tripper")
	}

	// Call all middleware
	responseHandlers := make([]func(*http.Response) error, len(mrt.Middleware))
	for idx, middleware := range mrt.Middleware {
		updatedReq, callback, err := middleware(req)
		if err != nil {
			return nil, fmt.Errorf("error invoking http client middleware: %w", err)
		}
		responseHandlers[idx] = callback
		req = updatedReq
	}

	// Make the request
	resp, err := mrt.RoundTripper.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("http client request failed: %w", err)
	}

	// Call response handler callbacks with the response
	for _, callback := range responseHandlers {
		if err := callback(resp); err != nil {
			return nil, fmt.Errorf("error invoking http client response handler middleware: %w", err)
		}
	}
	return resp, err
}
