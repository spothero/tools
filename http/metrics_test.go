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
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/spothero/tools/http/mock"
	"github.com/spothero/tools/http/writer"
	"github.com/spothero/tools/jose"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRegistry struct {
	error error
}

func (s mockRegistry) Register(prometheus.Collector) error {
	return s.error
}

func (s mockRegistry) MustRegister(...prometheus.Collector) {}

func (s mockRegistry) Unregister(prometheus.Collector) bool {
	return true
}

func TestNewMetrics(t *testing.T) {
	tests := []struct {
		name         string
		mustRegister bool
		duplicate    bool
	}{
		{
			"when must register is true and we do not duplicate registration no panic occurs",
			true,
			false,
		},
		{
			"when must register is true and we duplicate registration a panic occurs",
			true,
			true,
		},
		{
			"when must register is false and we do not duplicate registration no panic occurs",
			false,
			false,
		},
		{
			"when must register is false and we duplicate registration a panic occurs",
			false,
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry := prometheus.NewRegistry()
			metrics := NewMetrics(registry, test.mustRegister)
			if test.duplicate {
				if test.mustRegister {
					assert.Panics(t, func() { NewMetrics(registry, test.mustRegister) })
				} else {
					assert.NotPanics(t, func() { _ = NewMetrics(registry, test.mustRegister) })
				}
			}
			assert.NotNil(t, metrics.requestCounter)
			assert.NotNil(t, metrics.clientCounter)
			assert.NotNil(t, metrics.duration)
			assert.NotNil(t, metrics.clientDuration)
			assert.NotNil(t, metrics.contentLength)
			assert.NotNil(t, metrics.clientContentLength)
		})
	}
}

func TestNewMetrics_RegistryErrorDoesNotPanic(t *testing.T) {
	assert.Panics(t, func() { NewMetrics(mockRegistry{errors.New("some error")}, false) })
}

func TestMiddleware(t *testing.T) {
	const statusCode = 666
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(statusCode)
	})

	metrics := NewMetrics(nil, true)
	router := mux.NewRouter()
	router.Handle("/", writer.StatusRecorderMiddleware(metrics.Middleware(testHandler)))
	testServer := httptest.NewServer(router)
	defer testServer.Close()
	res, err := http.Get(testServer.URL)
	require.NoError(t, err)
	require.NotNil(t, res)
	defer res.Body.Close()

	// Check request counter
	labels := prometheus.Labels{
		"path":                 "/",
		"authenticated_client": UNAUTHENTICATED,
		"method":               "GET",
	}
	requestCounter, err := metrics.requestCounter.GetMetricWith(labels)
	assert.NoError(t, err)
	pb := &dto.Metric{}
	assert.NoError(t, requestCounter.Write(pb))
	assert.Equal(t, 1, int(pb.Counter.GetValue()))

	// add status code label for returning counters
	labels["status_code"] = "666"

	// Check duration histogram
	histogram, err := metrics.duration.GetMetricWith(labels)
	assert.NoError(t, err)
	pb = &dto.Metric{}
	assert.NoError(t, histogram.(prometheus.Histogram).Write(pb))
	buckets := pb.Histogram.GetBucket()
	assert.NotEmpty(t, buckets)
	for _, bucket := range pb.Histogram.GetBucket() {
		// Choose a bucket which gives a full second to this test and ensure we have a count of at
		// least one. This just ensures that our timer is working. This request should never take
		// longer than a millisecond, but we hugely increase the threshold to ensure we dont
		// introduce tests that periodically fail for no clear reason.
		if bucket.GetUpperBound() >= 1.0 {
			assert.Equal(t, uint64(1), bucket.GetCumulativeCount())
			break
		}
	}
	prometheus.Unregister(metrics.duration)
	prometheus.Unregister(metrics.clientDuration)

	// Check content-length histogram
	contentLengthHistogram, err := metrics.contentLength.GetMetricWith(labels)
	assert.NoError(t, err)
	pb = &dto.Metric{}
	assert.NoError(t, contentLengthHistogram.(prometheus.Histogram).Write(pb))
	buckets = pb.Histogram.GetBucket()
	assert.NotEmpty(t, buckets)
	prometheus.Unregister(metrics.contentLength)
	prometheus.Unregister(metrics.clientContentLength)

	// Check request returned counter
	responseCounter, err := metrics.responseCounter.GetMetricWith(labels)
	assert.NoError(t, err)
	pb = &dto.Metric{}
	assert.NoError(t, responseCounter.Write(pb))
	assert.Equal(t, 1, int(pb.Counter.GetValue()))

	prometheus.Unregister(metrics.requestCounter)
	prometheus.Unregister(metrics.responseCounter)
	prometheus.Unregister(metrics.clientCounter)
	prometheus.Unregister(metrics.circuitBreakerOpen)
}

func TestMetricsRoundTrip(t *testing.T) {
	tests := []struct {
		roundTripper          http.RoundTripper
		name                  string
		expectErr             bool
		expectCircuitBreakErr bool
		expectPanic           bool
	}{
		{
			name:        "no roundtripper results in a panic",
			expectPanic: true,
		},
		{
			name:         "an error on roundtrip is reported to the caller",
			roundTripper: &mock.RoundTripper{ResponseStatusCodes: []int{http.StatusOK}, CreateErr: true},
			expectErr:    true,
		},
		{
			name:         "http requests are measured and status code is recorded on request",
			roundTripper: &mock.RoundTripper{ResponseStatusCodes: []int{http.StatusOK}, CreateErr: false},
		},
		{
			name: "circuit-breaking errors are recorded correctly in the metrics",
			roundTripper: &mock.RoundTripper{
				ResponseStatusCodes: []int{http.StatusOK},
				CreateErr:           true,
				DesiredErr:          mock.CircuitError{CircuitOpened: true},
			},
			expectErr:             true,
			expectCircuitBreakErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			metricsRT := MetricsRoundTripper{
				RoundTripper: test.roundTripper,
				Metrics:      NewMetrics(nil, true),
			}
			mockReq := httptest.NewRequest("GET", "/path", nil)
			if test.expectPanic {
				assert.Panics(t, func() {
					_, _ = metricsRT.RoundTrip(mockReq)
				})
			} else if test.expectErr {
				resp, err := metricsRT.RoundTrip(mockReq)
				assert.Nil(t, resp)
				assert.Error(t, err)

				if test.expectCircuitBreakErr {
					// Check circuit-breaker counter
					counter, err := metricsRT.Metrics.circuitBreakerOpen.GetMetricWith(prometheus.Labels{"host": ""})
					assert.NoError(t, err)
					pb := &dto.Metric{}
					assert.NoError(t, counter.Write(pb))
					assert.Equal(t, 1, int(pb.Counter.GetValue()))
				}
			} else {
				mockReq.Header.Set("Content-Length", "1")
				resp, err := metricsRT.RoundTrip(mockReq)
				assert.NotNil(t, resp)
				assert.NoError(t, err)

				// Expected prometheus labels after this request
				labels := prometheus.Labels{
					"path":                 "/path",
					"status_code":          "200",
					"authenticated_client": UNAUTHENTICATED,
					"method":               "GET",
				}

				// Check duration histogram
				histogram, err := metricsRT.Metrics.clientDuration.GetMetricWith(labels)
				assert.NoError(t, err)
				pb := &dto.Metric{}
				assert.NoError(t, histogram.(prometheus.Histogram).Write(pb))
				buckets := pb.Histogram.GetBucket()
				assert.NotEmpty(t, buckets)
				for _, bucket := range pb.Histogram.GetBucket() {
					// Choose a bucket which gives a full second to this test and ensure we have a count of at
					// least one. This just ensures that our timer is working. This request should never take
					// longer than a millisecond, but we hugely increase the threshold to ensure we dont
					// introduce tests that periodically fail for no clear reason.
					if bucket.GetUpperBound() >= 1.0 {
						assert.Equal(t, uint64(1), bucket.GetCumulativeCount())
						break
					}
				}

				// Check content-length histogram
				contentLengthHistogram, err := metricsRT.Metrics.clientContentLength.GetMetricWith(labels)
				assert.NoError(t, err)
				pb = &dto.Metric{}
				assert.NoError(t, contentLengthHistogram.(prometheus.Histogram).Write(pb))
				buckets = pb.Histogram.GetBucket()
				assert.NotEmpty(t, buckets)

				// Check request counter
				counter, err := metricsRT.Metrics.clientCounter.GetMetricWith(labels)
				assert.NoError(t, err)
				pb = &dto.Metric{}
				assert.NoError(t, counter.Write(pb))
				assert.Equal(t, 1, int(pb.Counter.GetValue()))

				// Check circuit-breaker counter
				counter, err = metricsRT.Metrics.circuitBreakerOpen.GetMetricWith(prometheus.Labels{"host": ""})
				assert.NoError(t, err)
				pb = &dto.Metric{}
				assert.NoError(t, counter.Write(pb))
				assert.Equal(t, 0, int(pb.Counter.GetValue()))
			}
			prometheus.Unregister(metricsRT.Metrics.duration)
			prometheus.Unregister(metricsRT.Metrics.clientDuration)
			prometheus.Unregister(metricsRT.Metrics.contentLength)
			prometheus.Unregister(metricsRT.Metrics.clientContentLength)
			prometheus.Unregister(metricsRT.Metrics.requestCounter)
			prometheus.Unregister(metricsRT.Metrics.responseCounter)
			prometheus.Unregister(metricsRT.Metrics.clientCounter)
			prometheus.Unregister(metricsRT.Metrics.circuitBreakerOpen)
		})
	}
}

func TestRetrieveAuthenticatedClient(t *testing.T) {
	tests := []struct {
		name       string
		auth0Claim *jose.Auth0Claim
		expected   string
	}{
		{
			name:       "base case - no auth0 claim",
			auth0Claim: nil,
			expected:   UNAUTHENTICATED,
		},
		{
			name: "auth0claim exists",
			auth0Claim: &jose.Auth0Claim{
				ID:         "123",
				ClientName: "client-application",
			},
			expected: "client-application",
		},
	}
	for _, test := range tests {
		request := &http.Request{}

		if test.auth0Claim != nil {
			authenticatedContext := context.WithValue(
				request.Context(),
				jose.Auth0ClaimKey,
				test.auth0Claim,
			)
			request = request.WithContext(authenticatedContext)
		}

		authenticatedClient := retrieveAuthenticatedClient(request)

		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, authenticatedClient)
		})
	}
}
