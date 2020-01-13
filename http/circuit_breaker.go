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
	"sync"
	"time"

	"github.com/afex/hystrix-go/hystrix"
)

// CircuitBreakerRoundTripper wraps a RoundTrapper with circuit-breaker logic
type CircuitBreakerRoundTripper struct {
	RoundTripper       http.RoundTripper
	hostConfiguration  map[string]hystrix.CommandConfig
	registeredHostsSet map[string]bool
	defaultConfig      hystrix.CommandConfig
	registrationMutex  sync.RWMutex
	configMutex        sync.RWMutex
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
func NewDefaultCircuitBreakerRoundTripper(roundTripper http.RoundTripper) *CircuitBreakerRoundTripper {
	// Ensure the RoundTripper was set on the CircuitBreakerRoundTripper
	if roundTripper == nil {
		panic("no roundtripper provided to circuit-breaker round tripper")
	}
	return &CircuitBreakerRoundTripper{
		RoundTripper:       roundTripper,
		hostConfiguration:  make(map[string]hystrix.CommandConfig),
		registeredHostsSet: make(map[string]bool),
		defaultConfig: hystrix.CommandConfig{
			Timeout:                int((30 * time.Second).Milliseconds()), // 30 second timeout
			MaxConcurrentRequests:  int(math.MaxInt32),                     // Do not limit concurrent requests by default
			RequestVolumeThreshold: hystrix.DefaultVolumeThreshold,
			SleepWindow:            hystrix.DefaultSleepWindow,
			ErrorPercentThreshold:  hystrix.DefaultErrorPercentThreshold,
		},
		registrationMutex: sync.RWMutex{},
		configMutex:       sync.RWMutex{},
	}
}

// WithHostConfiguration sets the host configuration on the CircuitBreakerRoundTripper and returns
// the modified configuration.
//
// The host configuration is a map of hostname to Hystrix configuration settings. Use of this
// function is discouraged unless the caller has established reasons to modify the configuration
// for a particular host.
func (cbrt *CircuitBreakerRoundTripper) WithHostConfiguration(hostConfiguration map[string]hystrix.CommandConfig) *CircuitBreakerRoundTripper {
	cbrt.configMutex.Lock()
	cbrt.hostConfiguration = hostConfiguration
	cbrt.configMutex.Unlock()
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

	// Register the host configuration if it is not yet registered
	cbrt.registrationMutex.RLock()
	_, ok := cbrt.registeredHostsSet[req.URL.Host]
	cbrt.registrationMutex.RUnlock()
	if !ok {
		cbrt.configMutex.RLock()
		hystrixConfig, configExists := cbrt.hostConfiguration[req.URL.Host]
		cbrt.configMutex.RUnlock()
		if !configExists {
			hystrixConfig = cbrt.defaultConfig
		}
		hystrix.ConfigureCommand(req.URL.Host, hystrixConfig)
		cbrt.registrationMutex.Lock()
		cbrt.registeredHostsSet[req.URL.Host] = true
		cbrt.registrationMutex.Unlock()
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