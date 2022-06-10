// Copyright 2022 SpotHero
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
	"fmt"
	"net/http"

	"github.com/cep21/circuit/v3"
)

// CircuitBreakerRoundTripper wraps a RoundTrapper with circuit-breaker logic
type CircuitBreakerRoundTripper struct {
	RoundTripper http.RoundTripper
	manager      circuit.Manager
}

// NewDefaultCircuitBreakerRoundTripper constructs and returns the default
// CircuitBreakerRoundTripper configuration.
//
// By default, circuit-breaking is configured on a host-by-host basis, meaning that errors for a
// given host will be accumulated on that host for determining whether or not to open the circuit
// breaker.
//
// The host configuration is a map of hostname to Circuit configuration settings. Use of this
// function is discouraged unless the caller has established reasons to modify the configuration
// for a particular host.
//
// **IMPORTANT**: If you decide to set the hostConfiguration, the key in the map **must be the host
//                name of the server you intend to call (eg req.URL.Host)**
//
// Additional Default:
// * Requests will timeout after 30 seconds
// * No ceiling is placed on the number of concurrent requests per host (set to MaxInt32)
// * 50% of requests must fail for the circuit-breaker to trip
// * If the circuit-breaker opens, no requests will be attempted until after 5 seconds have passed
// * At least 20 requests must be recorded to a host before the circuit breaker can be tripped
func NewDefaultCircuitBreakerRoundTripper(
	roundTripper http.RoundTripper,
	hostConfiguration map[string]circuit.Config,
) *CircuitBreakerRoundTripper {
	// Ensure the RoundTripper was set on the CircuitBreakerRoundTripper
	if roundTripper == nil {
		panic("no roundtripper provided to circuit-breaker round tripper")
	}
	cbrt := &CircuitBreakerRoundTripper{
		RoundTripper: roundTripper,
		manager:      circuit.Manager{},
	}
	for circuitName, circuitConfig := range hostConfiguration {
		_ = cbrt.manager.MustCreateCircuit(circuitName, circuitConfig)
	}
	return cbrt
}

// RoundTrip completes the http request round trip but wraps the call in circuit-breaking logic
// using the Netflix Hystrix approach.
func (cbrt *CircuitBreakerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Ensure the RoundTripper was set on the CircuitBreakerRoundTripper
	if cbrt.RoundTripper == nil {
		panic("no roundtripper provided to circuit-breaker round tripper")
	}

	var resp *http.Response
	var requestErr error
	makeRequestFunc := func(ctx context.Context) error {
		resp, requestErr = cbrt.RoundTripper.RoundTrip(req)
		if requestErr != nil || resp.StatusCode >= http.StatusInternalServerError {
			return fmt.Errorf("failed request, invoking circuit-breaker")
		}
		return nil
	}

	// Fetch or register the circuit breaker for this host
	circuitBreaker := cbrt.manager.GetCircuit(req.URL.Host)
	if circuitBreaker == nil {
		circuitBreaker = cbrt.manager.MustCreateCircuit(req.URL.Host)
	}

	var err error
	if cbErr := circuitBreaker.Execute(req.Context(), makeRequestFunc, nil); cbErr != nil {
		err = requestErr
		// In cases where the returned error type is a circuit-breaker error, we want to return the
		// specific error type instead of the HTTP error. This allows upstream calls to
		// appropriately handle the failure
		var circuitError circuit.Error
		if errors.As(cbErr, &circuitError) {
			err = cbErr
		}
	}
	return resp, err
}
