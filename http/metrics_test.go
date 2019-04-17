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

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func TestNewMetrics(t *testing.T) {
	metrics := NewMetrics("test")
	assert.Equal(t, "test", metrics.serverName)
	assert.NotNil(t, metrics.counter)
	prometheus.Unregister(metrics.counter)
	assert.NotNil(t, metrics.duration)
	prometheus.Unregister(metrics.duration)
}

func TestMiddleware(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)
	httpRec := httptest.NewRecorder()

	metrics := NewMetrics("test")
	handler := func(w http.ResponseWriter, r *http.Request) {
		sr := &StatusRecorder{w, http.StatusOK}
		deferableFunc, r := metrics.Middleware(sr, r)
		defer deferableFunc()
	}
	http.HandlerFunc(handler).ServeHTTP(httpRec, req)

	// Expected prometheus labels after this request
	labels := prometheus.Labels{
		"path":        "/",
		"status_code": "200",
	}

	// Check duration histogram
	histogram, err := metrics.duration.GetMetricWith(labels)
	assert.NoError(t, err)
	pb := &dto.Metric{}
	histogram.(prometheus.Histogram).Write(pb)
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

	// Check request counter
	counter, err := metrics.counter.GetMetricWith(labels)
	assert.NoError(t, err)
	pb = &dto.Metric{}
	counter.Write(pb)
	assert.Equal(t, 1, int(pb.Counter.GetValue()))
	prometheus.Unregister(metrics.counter)
}
