// Copyright 2023 SpotHero
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/cep21/circuit/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spothero/tools/http/writer"
	"github.com/spothero/tools/jose"
)

// UNAUTHENTICATED is the string used when the client is unknown
const UNAUTHENTICATED = "unauthenticated"

// Metrics is a bundle of prometheus HTTP metrics recorders
type Metrics struct {
	requestCounter      *prometheus.CounterVec
	responseCounter     *prometheus.CounterVec
	duration            *prometheus.HistogramVec
	contentLength       *prometheus.HistogramVec
	clientCounter       *prometheus.CounterVec
	clientDuration      *prometheus.HistogramVec
	clientContentLength *prometheus.HistogramVec
	circuitBreakerOpen  *prometheus.CounterVec
}

// registerCollector will register the passed collector
func registerCollector(registry prometheus.Registerer, collector prometheus.Collector, mustRegister bool) prometheus.Collector {
	if mustRegister {
		registry.MustRegister(collector)
	} else {
		if err := registry.Register(collector); err != nil {
			if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
				// metric has been registered before so use existing metric
				return are.ExistingCollector
			}
			// Something else went wrong so panic
			panic(err)
		}
	}
	return collector
}

// NewMetrics creates and returns a metrics bundle. The user may optionally
// specify an existing Prometheus Registry. If no Registry is provided, the global Prometheus
// Registry is used.
//
// Finally, if mustRegister is true and a registration error is encountered,
// the application will panic.
//
// If mustRegister is false and registration failed due to the collector already being registered
// then the existing collector will be returned.  But if registration failed for any other reason then
// the application will panic.
func NewMetrics(registry prometheus.Registerer, mustRegister bool) Metrics {
	labels := []string{"path", "authenticated_client", "method"}

	// If the user has not provided a Prometheus Registry, use the global Registry
	if registry == nil {
		registry = prometheus.DefaultRegisterer
	}

	requestCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP Requests received",
		},
		labels,
	)
	requestCounter = registerCollector(registry, requestCounter, mustRegister).(*prometheus.CounterVec)

	// add status code label for metrics tracked on request return
	labels = append(labels, "status_code")

	histogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "Total duration histogram for the HTTP request",
			// Power of 2 time - 1ms, 2ms, 4ms ... 32768ms, +Inf ms
			Buckets: prometheus.ExponentialBuckets(0.001, 2.0, 16),
		},
		labels,
	)
	histogram = registerCollector(registry, histogram, mustRegister).(*prometheus.HistogramVec)

	clientHistogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_client_request_duration_seconds",
			Help: "Total duration histogram for the HTTP client requests",
			// Power of 2 time - 1ms, 2ms, 4ms ... 32768ms, +Inf ms
			Buckets: prometheus.ExponentialBuckets(0.001, 2.0, 16),
		},
		labels,
	)
	clientHistogram = registerCollector(registry, clientHistogram, mustRegister).(*prometheus.HistogramVec)

	responseCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_responses_total",
			Help: "Total number of HTTP responses",
		},
		labels,
	)
	responseCounter = registerCollector(registry, responseCounter, mustRegister).(*prometheus.CounterVec)

	clientCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_client_requests_total",
			Help: "Total number of HTTP Client Requests sent",
		},
		labels,
	)
	clientCounter = registerCollector(registry, clientCounter, mustRegister).(*prometheus.CounterVec)

	contentLength := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_content_length_bytes",
			Help: "HTTP Request content length histogram, buckets range from 1B to 16MB",
			// Power of 2 bytes, starts at 1 byte and works up to 16MB
			Buckets: prometheus.ExponentialBuckets(1, 2.0, 24),
		},
		labels,
	)
	contentLength = registerCollector(registry, contentLength, mustRegister).(*prometheus.HistogramVec)

	clientContentLength := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_client_content_length_bytes",
			Help: "HTTP Client Request content length histogram, buckets range from 1B to 16MB",
			// Power of 2 bytes, starts at 1 byte and works up to 16MB
			Buckets: prometheus.ExponentialBuckets(1, 2.0, 24),
		},
		labels,
	)
	clientContentLength = registerCollector(registry, clientContentLength, mustRegister).(*prometheus.HistogramVec)

	circuitBreakerOpen := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_client_circuit_breaker_open_total",
			Help: "Total number of times the HTTP client circuit breaker has been open when a request was attempted",
		},
		[]string{"host"},
	)
	circuitBreakerOpen = registerCollector(registry, circuitBreakerOpen, mustRegister).(*prometheus.CounterVec)

	return Metrics{
		requestCounter:      requestCounter,
		responseCounter:     responseCounter,
		clientCounter:       clientCounter,
		duration:            histogram,
		clientDuration:      clientHistogram,
		contentLength:       contentLength,
		clientContentLength: clientContentLength,
		circuitBreakerOpen:  circuitBreakerOpen,
	}
}

// Middleware provides standard HTTP middleware for recording prometheus metrics on every request.
// Note that this middleware must be attached after writer.StatusRecorderMiddleware
// for HTTP response code tagging to function.
func (m Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		labels := prometheus.Labels{
			"path":                 writer.FetchRoutePathTemplate(r),
			"authenticated_client": retrieveAuthenticatedClient(r),
			"method": 				r.Method,
		}
		m.requestCounter.With(labels).Inc()

		timer := prometheus.NewTimer(prometheus.ObserverFunc(func(durationSec float64) {
			if statusRecorder, ok := w.(*writer.StatusRecorder); ok {
				labels["status_code"] = strconv.Itoa(statusRecorder.StatusCode)
			}
			m.responseCounter.With(labels).Inc()
			if contentLengthStr := r.Header.Get("Content-Length"); len(contentLengthStr) > 0 {
				if contentLength, err := strconv.Atoi(contentLengthStr); err == nil {
					m.contentLength.With(labels).Observe(float64(contentLength))
				}
			}
			m.duration.With(labels).Observe(durationSec)
		}))
		defer timer.ObserveDuration()
		next.ServeHTTP(w, r)
	})
}

// MetricsRoundTripper implements a proxied net/http RoundTripper so that http requests may be
// measured with metrics
type MetricsRoundTripper struct {
	RoundTripper http.RoundTripper
	Metrics      Metrics // An instantiated http.Metrics bundle for measuring timings and status codes
}

// RoundTrip measures HTTP client call duration and status codes
func (metricsRT MetricsRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	// Ensure the RoundTripper was set on the MiddlewareRoundTripper
	if metricsRT.RoundTripper == nil {
		panic("no roundtripper provided to metrics round tripper")
	}

	// Make the request
	var resp *http.Response
	var err error
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(durationSec float64) {
		if resp != nil {
			labels := prometheus.Labels{
				"path":                 r.URL.Path,
				"status_code":          strconv.Itoa(resp.StatusCode),
				"authenticated_client": retrieveAuthenticatedClient(r),
				"method":				r.Method,
			}
			metricsRT.Metrics.clientCounter.With(labels).Inc()
			if contentLengthStr := r.Header.Get("Content-Length"); len(contentLengthStr) > 0 {
				if contentLength, contentLengthErr := strconv.Atoi(contentLengthStr); contentLengthErr == nil {
					metricsRT.Metrics.clientContentLength.With(labels).Observe(float64(contentLength))
				}
			}
			metricsRT.Metrics.clientDuration.With(labels).Observe(durationSec)
		}
		var circuitError circuit.Error
		if err != nil && errors.As(err, &circuitError) && circuitError.CircuitOpen() {
			metricsRT.Metrics.circuitBreakerOpen.With(prometheus.Labels{"host": r.URL.Host}).Inc()
		}
	}))
	defer timer.ObserveDuration()
	resp, err = metricsRT.RoundTripper.RoundTrip(r)
	return resp, err
}

func retrieveAuthenticatedClient(r *http.Request) string {
	claim, err := jose.FromContext(r.Context())
	if err != nil {
		return UNAUTHENTICATED
	}

	return claim.ClientName
}
