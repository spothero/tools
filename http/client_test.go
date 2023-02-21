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
	"net/http"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spothero/tools/jose"
	"github.com/spothero/tools/log"
	"github.com/spothero/tools/tracing"
	"github.com/stretchr/testify/assert"
)

func TestNewDefaultClient(t *testing.T) {
	metrics := NewMetrics(prometheus.NewRegistry(), true)
	client := NewDefaultClient(metrics, nil)
	assert.NotNil(t, client)
	jrt, ok := client.Transport.(jose.RoundTripper)
	assert.True(t, ok)

	mrt, ok := jrt.RoundTripper.(MetricsRoundTripper)
	assert.True(t, ok)
	assert.Equal(t, metrics, mrt.Metrics)

	lrt, ok := mrt.RoundTripper.(log.RoundTripper)
	assert.True(t, ok)

	trt, ok := lrt.RoundTripper.(tracing.RoundTripper)
	assert.True(t, ok)

	rrt, ok := trt.RoundTripper.(RetryRoundTripper)
	assert.True(t, ok)

	cbrt, ok := rrt.RoundTripper.(*CircuitBreakerRoundTripper)
	assert.True(t, ok)

	assert.Equal(t, http.DefaultTransport, cbrt.RoundTripper)
}
