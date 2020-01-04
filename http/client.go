package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/spothero/tools/jose"
	"github.com/spothero/tools/log"
	"github.com/spothero/tools/tracing"
	"go.uber.org/zap"
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

// RetryRoundTripper wraps a roundtripper with retry logic
type RetryRoundTripper struct {
	RoundTripper         http.RoundTripper
	RetriableStatusCodes []int
	InitialInterval      time.Duration
	RandomizationFactor  float64
	Multiplier           float64
	MaxInterval          time.Duration
	MaxRetries           uint64
}

// NewDefaultClient constructs the default HTTP Client with middleware. Providing an HTTP
// RoundTripper is optional. If `nil` is received, the DefaultClient will be used.
func NewDefaultClient(metrics Metrics, roundTripper http.RoundTripper) http.Client {
	if roundTripper == nil {
		roundTripper = http.DefaultTransport
	}
	retryRoundTripper := RetryRoundTripper{
		RoundTripper: roundTripper,
		RetriableStatusCodes: []int{
			http.StatusInternalServerError,
			http.StatusNotImplemented,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout,
			http.StatusHTTPVersionNotSupported,
			http.StatusVariantAlsoNegotiates,
			http.StatusInsufficientStorage,
			http.StatusLoopDetected,
			http.StatusNotExtended,
			http.StatusNetworkAuthenticationRequired,
		},
		InitialInterval:     100 * time.Millisecond,
		Multiplier:          2,
		MaxInterval:         30 * time.Second,
		RandomizationFactor: 0.5,
		MaxRetries:          5,
	}
	return http.Client{
		Transport: MiddlewareRoundTripper{
			Middleware: []ClientMiddleware{
				tracing.HTTPClientMiddleware,
				log.HTTPClientMiddleware,
				metrics.ClientMiddleware,
				jose.HTTPClientMiddleware,
			},
			RoundTripper: retryRoundTripper,
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

// RoundTrip completes the http request round trip but attempts retries for configured error codes
func (rrt RetryRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Ensure the RoundTripper was set on the MiddlewareRoundTripper
	if rrt.RoundTripper == nil {
		panic("no roundtripper provided to middleware round tripper")
	}

	var resp *http.Response
	var err error
	makeRequestRetriable := func() error {
		resp, err = rrt.RoundTripper.RoundTrip(req)

		// If an error was encountered, retry. This typically indicates a failure to get a
		// response.
		if err != nil {
			log.Get(req.Context()).Debug("retrying failed http request", zap.Error(err))
			return err
		}

		// If no error was encountered, return immediately
		if resp.StatusCode < http.StatusBadRequest {
			return nil
		}

		// Check to see if this status code is retriable
		for _, retriableCode := range rrt.RetriableStatusCodes {
			if resp.StatusCode == retriableCode {
				// Return an error indicating a retriable error condition
				log.Get(req.Context()).Debug("retrying retriable http request", zap.Int("http.status_code", resp.StatusCode))
				return fmt.Errorf("status code `%v` is retriable", resp.StatusCode)
			}
		}

		// The status code is not retriable
		log.Get(req.Context()).Debug("could not retry failed http request", zap.Int("http.status_code", resp.StatusCode))
		return nil
	}

	// Each backoff policy contains state, so unfortunately we must create a fresh backoff
	// policy for every request
	expBackOff := backoff.NewExponentialBackOff()
	expBackOff.InitialInterval = rrt.InitialInterval
	expBackOff.Multiplier = rrt.Multiplier
	expBackOff.MaxInterval = rrt.MaxInterval
	expBackOff.RandomizationFactor = rrt.RandomizationFactor
	backoffPolicy := backoff.WithContext(
		backoff.WithMaxRetries(expBackOff, rrt.MaxRetries),
		req.Context(),
	)
	if retryErr := backoff.Retry(makeRequestRetriable, backoffPolicy); retryErr != nil {
		log.Get(req.Context()).Debug("failed retrying http request")
	}
	return resp, err
}
