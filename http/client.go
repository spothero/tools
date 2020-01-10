// Copyright 2020 SpotHero
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

	"github.com/spothero/tools/jose"
	"github.com/spothero/tools/log"
	"github.com/spothero/tools/tracing"
)

// NewDefaultClient constructs the default HTTP Client with a series of HTTP RoundTrippers that
// provide additional features, such as exponential backoff, metrics, tracing, authentication
// passthrough, and logging. Providing the base HTTP RoundTripper is optional.
// If `nil` is received, the net/http DefaultClient will be used.
//
// By default, the client provides exponential backoff on [500-504] errors. The default
// configuration for exponential backoff is to start with an interval of 100 milliseconds, a
// multiplier of two, a randomization factor of up to 0.5 milliseconds (for jitter), a max
// interval of 10 seconds, and finally, the retry will attempt 5 times before failing if the
// error is retriable.
func NewDefaultClient(metrics Metrics, roundTripper http.RoundTripper) http.Client {
	if roundTripper == nil {
		roundTripper = http.DefaultTransport
	}
	circuitBreakerRoundTripper := NewDefaultCircuitBreakerRoundTripper(roundTripper)
	retryRoundTripper := NewDefaultRetryRoundTripper(circuitBreakerRoundTripper)
	tracingRoundTripper := tracing.RoundTripper{RoundTripper: retryRoundTripper}
	loggingRoundTripper := log.RoundTripper{RoundTripper: tracingRoundTripper}
	metricsRoundTripper := MetricsRoundTripper{RoundTripper: loggingRoundTripper}
	joseRoundTripper := jose.RoundTripper{RoundTripper: metricsRoundTripper}
	return http.Client{Transport: joseRoundTripper}
}
