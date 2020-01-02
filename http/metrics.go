// Copyright 2019 SpotHero
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
	"context"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spothero/tools/http/writer"
	"github.com/spothero/tools/log"
	"go.uber.org/zap"
)

// Metrics is a bundle of prometheus HTTP metrics recorders
type Metrics struct {
	counter        *prometheus.CounterVec
	duration       *prometheus.HistogramVec
	clientCounter  *prometheus.CounterVec
	clientDuration *prometheus.HistogramVec
}

// NewMetrics creates and returns a metrics bundle. The user may optionally
// specify an existing Prometheus Registry. If no Registry is provided, the global Prometheus
// Registry is used. Finally, if mustRegister is true, and a registration error is encountered,
// the application will panic.
func NewMetrics(registry prometheus.Registerer, mustRegister bool) Metrics {
	histogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "Total duration histogram for the HTTP request",
			// Power of 2 time - 1ms, 2ms, 4ms ... 32768ms, +Inf ms
			Buckets: prometheus.ExponentialBuckets(0.001, 2.0, 16),
		},
		[]string{
			// The path recording the request
			"path",
			// The Specific HTTP Status Code
			"status_code",
		},
	)
	clientHistogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_client_request_duration_seconds",
			Help: "Total duration histogram for the HTTP client requests",
			// Power of 2 time - 1ms, 2ms, 4ms ... 32768ms, +Inf ms
			Buckets: prometheus.ExponentialBuckets(0.001, 2.0, 16),
		},
		[]string{
			// The path recording the request
			"path",
			// The Specific HTTP Status Code
			"status_code",
		},
	)
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP Requests received",
		},
		[]string{
			// The path recording the request
			"path",
			// The Specific HTTP Status Code
			"status_code",
		},
	)
	clientCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_client_requests_total",
			Help: "Total number of HTTP Client Requests sent",
		},
		[]string{
			// The path recording the request
			"path",
			// The Specific HTTP Status Code
			"status_code",
		},
	)
	// If the user hasnt provided a Prometheus Registry, use the global Registry
	if registry == nil {
		registry = prometheus.DefaultRegisterer
	}
	if mustRegister {
		registry.MustRegister(histogram)
		registry.MustRegister(clientHistogram)
		registry.MustRegister(counter)
		registry.MustRegister(clientCounter)
	} else {
		if err := registry.Register(histogram); err != nil {
			log.Get(context.Background()).Error("failed to register http histogram", zap.Error(err))
		}
		if err := registry.Register(clientHistogram); err != nil {
			log.Get(context.Background()).Error("failed to register http client histogram", zap.Error(err))
		}
		if err := registry.Register(counter); err != nil {
			log.Get(context.Background()).Error("failed to register http counter", zap.Error(err))
		}
		if err := registry.Register(clientCounter); err != nil {
			log.Get(context.Background()).Error("failed to register http client counter", zap.Error(err))
		}
	}
	return Metrics{
		counter:        counter,
		clientCounter:  clientCounter,
		duration:       histogram,
		clientDuration: clientHistogram,
	}
}

// Middleware provides standard HTTP middleware for recording prometheus metrics on every request.
// Note that this middleware must be attached after writer.StatusRecorderMiddleware
// for HTTP response code tagging to function.
func (m Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timer := prometheus.NewTimer(prometheus.ObserverFunc(func(durationSec float64) {
			labels := prometheus.Labels{"path": writer.FetchRoutePathTemplate(r)}
			if statusRecorder, ok := w.(*writer.StatusRecorder); ok {
				labels["status_code"] = strconv.Itoa(statusRecorder.StatusCode)
			}
			m.counter.With(labels).Inc()
			m.duration.With(labels).Observe(durationSec)
		}))
		defer timer.ObserveDuration()
		next.ServeHTTP(w, r)
	})
}

// ClientMiddleware is middleware for use in HTTP Clients for capturing prometheus metrics
func (m Metrics) ClientMiddleware(r *http.Request) (*http.Request, func(*http.Response) error, error) {
	var receivedResp *http.Response
	observerFunc := func(durationSec float64) {
		labels := prometheus.Labels{
			"path":        r.URL.Path,
			"status_code": strconv.Itoa(receivedResp.StatusCode),
		}
		m.clientCounter.With(labels).Inc()
		m.clientDuration.With(labels).Observe(durationSec)
	}
	timer := prometheus.NewTimer(prometheus.ObserverFunc(observerFunc))
	return r, func(resp *http.Response) error {
		receivedResp = resp
		timer.ObserveDuration()
		return nil
	}, nil
}
