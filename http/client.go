package http

import (
	"fmt"
	"net/http"
)

// ClientMiddleware defines an HTTP Client middleware function. The function is called prior to
// invoking the transport roundtrip, and the returned response function is called after the
// response has been received from the client.
type ClientMiddleware func(*http.Request) (func(*http.Response) error, error)

// MiddlewareRoundTripper implements a proxied net/http RoundTripper so that http requests may be decorated
// with middleware
type MiddlewareRoundTripper struct {
	roundTripper http.RoundTripper
	Middleware   []ClientMiddleware
}

// RoundTrip completes the http request round trip and is responsible for invoking HTTP Client
// Middleware
func (mrt MiddlewareRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Call all middleware
	responseHandlers := make([]func(*http.Response) error, len(mrt.Middleware))
	for idx, middleware := range mrt.Middleware {
		callback, err := middleware(req)
		if err != nil {
			return nil, fmt.Errorf("error invoking http client middleware: %w", err)
		}
		responseHandlers[idx] = callback
	}

	// Make the request
	resp, err := mrt.roundTripper.RoundTrip(req)
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

// BackoffClientMiddleware is middleware for use in HTTP Clients for automatically performing
// exponential backoff and retries on failed HTTP requests
func BackoffClientMiddleware(r *http.Request) (func(*http.Response) error, error) {
	// TODO: Middleware!
	return func(resp *http.Response) error {
		return nil
	}, nil
}
