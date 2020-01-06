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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/spothero/tools/http/roundtrip"
	"github.com/spothero/tools/http/writer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			assert.NotNil(t, metrics.counter)
			assert.NotNil(t, metrics.clientCounter)
			assert.NotNil(t, metrics.duration)
			assert.NotNil(t, metrics.clientDuration)
			assert.NotNil(t, metrics.contentLength)
			assert.NotNil(t, metrics.clientContentLength)
		})
	}
}

func TestMiddleware(t *testing.T) {
	const statusCode = 666
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	// Expected prometheus labels after this request
	labels := prometheus.Labels{
		"path":        "/",
		"status_code": "666",
	}

	// Check duration histogram
	histogram, err := metrics.duration.GetMetricWith(labels)
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

	// Check request counter
	counter, err := metrics.counter.GetMetricWith(labels)
	assert.NoError(t, err)
	pb = &dto.Metric{}
	assert.NoError(t, counter.Write(pb))
	assert.Equal(t, 1, int(pb.Counter.GetValue()))
	prometheus.Unregister(metrics.counter)
	prometheus.Unregister(metrics.clientCounter)
}

func TestClientMiddleware(t *testing.T) {
	tests := []struct {
		name         string
		roundTripper http.RoundTripper
		expectPanic  bool
	}{
		{
			"no roundtripper results in a panic",
			nil,
			true,
		},
		{
			"http requests are measured and status code is recorded on request",
			&roundtrip.MockRoundTripper{ResponseStatusCodes: []int{http.StatusOK}, CreateErr: false},
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			metricsRT := MetricsRoundTripper{
				RoundTripper: test.roundTripper,
				metrics:      NewMetrics(nil, true),
			}
			mockReq := httptest.NewRequest("GET", "/path", nil)
			if test.expectPanic {
				assert.Panics(t, func() {
					_, _ = metricsRT.RoundTrip(mockReq)
				})
			} else {
				resp, err := metricsRT.RoundTrip(mockReq)
				assert.NotNil(t, resp)
				assert.NoError(t, err)

				// Expected prometheus labels after this request
				labels := prometheus.Labels{
					"path":        "/path",
					"status_code": "200",
				}

				// Check duration histogram
				histogram, err := metricsRT.metrics.clientDuration.GetMetricWith(labels)
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
				contentLengthHistogram, err := metricsRT.metrics.clientContentLength.GetMetricWith(labels)
				assert.NoError(t, err)
				pb = &dto.Metric{}
				assert.NoError(t, contentLengthHistogram.(prometheus.Histogram).Write(pb))
				buckets = pb.Histogram.GetBucket()
				assert.NotEmpty(t, buckets)

				// Check request counter
				counter, err := metricsRT.metrics.clientCounter.GetMetricWith(labels)
				assert.NoError(t, err)
				pb = &dto.Metric{}
				assert.NoError(t, counter.Write(pb))
				assert.Equal(t, 1, int(pb.Counter.GetValue()))
			}
			prometheus.Unregister(metricsRT.metrics.duration)
			prometheus.Unregister(metricsRT.metrics.clientDuration)
			prometheus.Unregister(metricsRT.metrics.contentLength)
			prometheus.Unregister(metricsRT.metrics.clientContentLength)
			prometheus.Unregister(metricsRT.metrics.counter)
			prometheus.Unregister(metricsRT.metrics.clientCounter)
		})
	}
}
