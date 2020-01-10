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
	"context"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/afex/hystrix-go/hystrix"
)

// CircuitBreakerRoundTripper wraps a RoundTrapper with circuit-breaker logic
type CircuitBreakerRoundTripper struct {
	RoundTripper http.RoundTripper
	// A map of hostname to Hystrix configuration settings. Default settings will be used if not
	// specified. Using the default is recommended unless you have a good reason to alter the
	// settings.
	HostConfiguration  map[string]hystrix.CommandConfig
	registeredHostsSet map[string]bool
	defaultConfig      hystrix.CommandConfig
}

// NewDefaultCircuitBreakerRoundTripper constructs and returns the default
// CircuitBreakerRoundTripper configuration.
//
// By default, circuit-breaking is configured on a host-by-host basis, meaning that errors for a
// given host will be accumulated on that host for determining whether or not to open the circuit
// breaker.
//
// Additional Default:
// * Requests will timeout after 30 seconds
// * No ceiling is placed on the number of concurrent requests per host (set to MaxInt32)
// * 50% of requests must fail for the circuit-breaker to trip
// * If the circuit-breaker opens, no requests will be attempted until after 5 seconds have passed
// * At least 20 requests must be recorded to a host before the circuit breaker can be tripped
func NewDefaultCircuitBreakerRoundTripper(roundTripper http.RoundTripper) CircuitBreakerRoundTripper {
	// Ensure the RoundTripper was set on the CircuitBreakerRoundTripper
	if roundTripper == nil {
		panic("no roundtripper provided to circuit-breaker round tripper")
	}
	return CircuitBreakerRoundTripper{
		RoundTripper:       roundTripper,
		HostConfiguration:  make(map[string]hystrix.CommandConfig),
		registeredHostsSet: make(map[string]bool),
		defaultConfig: hystrix.CommandConfig{
			Timeout:                int((30 * time.Second).Milliseconds()), // 30 second timeout
			MaxConcurrentRequests:  int(math.MaxInt32),                     // Do not limit concurrent requests by default
			RequestVolumeThreshold: hystrix.DefaultVolumeThreshold,
			SleepWindow:            hystrix.DefaultSleepWindow,
			ErrorPercentThreshold:  hystrix.DefaultErrorPercentThreshold,
		},
	}
}

// RoundTrip completes the http request round trip but wraps the call in circuit-breaking logic
// using the Netflix Hystrix approach.
func (cbrt CircuitBreakerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
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

	// Register the host configuration if it is not yet registered
	if _, ok := cbrt.registeredHostsSet[req.URL.Host]; !ok {
		hystrixConfig, configExists := cbrt.HostConfiguration[req.URL.Host]
		if !configExists {
			hystrixConfig = cbrt.defaultConfig
		}
		hystrix.ConfigureCommand(req.URL.Host, hystrixConfig)
		cbrt.registeredHostsSet[req.URL.Host] = true
	}

	var err error
	if cbErr := hystrix.DoC(req.Context(), req.URL.Host, makeRequestFunc, nil); cbErr != nil {
		err = requestErr
		switch cbErr.(type) {
		// In cases where the returned error type is a circuit-breaker error, we want to return the
		// specific error type instead of the HTTP error. This allows upstream calls to
		// appropriately handle the failure
		case hystrix.CircuitError:
			err = cbErr
		}
	}
	return resp, err
}
