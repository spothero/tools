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
	// Middleware consist of a function which is called prior to the execution of a request. This
	// function returns the potentially modified request, the post-response handler, and an error,
	// if any. The response handler is invoked after the HTTP request has been made.
	//
	// Middleware are called in the order they are specified. In otherwords, the first item in the
	// slice is the first middleware applied, and the last item in the slice is the last middleware
	// applied. Each response handler is called in the reverse order of the middleware. Meaning,
	// the last middleware called will be the first to have its response handler called, and
	// likewise, the first middleware called will be the last to have its handler called.
	Middleware []ClientMiddleware
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
		panic("no roundtripper provided to middleware round tripper")
	}

	// Call all middleware
	numMiddleware := len(mrt.Middleware)
	responseHandlers := make([]func(*http.Response) error, numMiddleware)
	for idx, middleware := range mrt.Middleware {
		updatedReq, callback, err := middleware(req)
		if err != nil {
			return nil, fmt.Errorf("error invoking http client middleware: %w", err)
		}
		// Append handlers in reverse order so that nesting is reverse on response handling.
		// Always call the last middleware's response handler first, the second to last
		// middleware's response handler second, and so on.
		responseHandlers[numMiddleware-idx-1] = callback
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
