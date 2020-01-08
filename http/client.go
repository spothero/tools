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
	"github.com/cenkalti/backoff/v4"
	"github.com/spothero/tools/jose"
	"github.com/spothero/tools/log"
	"github.com/spothero/tools/tracing"
	"go.uber.org/zap"
)

// RetryRoundTripper wraps a roundtripper with retry logic
type RetryRoundTripper struct {
	RoundTripper         http.RoundTripper
	RetriableStatusCodes map[int]bool
	InitialInterval      time.Duration
	RandomizationFactor  float64
	Multiplier           float64
	MaxInterval          time.Duration
	MaxRetries           uint8
}

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
	retryRoundTripper := RetryRoundTripper{
		RoundTripper: roundTripper,
		RetriableStatusCodes: map[int]bool{
			http.StatusInternalServerError: true,
			http.StatusBadGateway:          true,
			http.StatusServiceUnavailable:  true,
			http.StatusGatewayTimeout:      true,
		},
		InitialInterval:     100 * time.Millisecond,
		Multiplier:          2,
		MaxInterval:         10 * time.Second,
		RandomizationFactor: 0.5,
		MaxRetries:          5,
	}
	tracingRoundTripper := tracing.RoundTripper{RoundTripper: retryRoundTripper}
	loggingRoundTripper := log.RoundTripper{RoundTripper: tracingRoundTripper}
	metricsRoundTripper := MetricsRoundTripper{RoundTripper: loggingRoundTripper}
	joseRoundTripper := jose.RoundTripper{RoundTripper: metricsRoundTripper}
	return http.Client{Transport: joseRoundTripper}
}

// RoundTrip completes the http request round trip but attempts retries for configured error codes
func (rrt RetryRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Ensure the RoundTripper was set on the RetryRoundTripper
	if rrt.RoundTripper == nil {
		panic("no roundtripper provided to retry round tripper")
	}

	var resp *http.Response
	var err error
	makeRequestRetriable := func() error {
		resp, err = rrt.RoundTripper.RoundTrip(req)

		// If an error was encountered, retry. This typically indicates a failure to get a
		// response.
		if err != nil {
			log.Get(req.Context()).Debug("retrying failed http request", zap.Error(err))
			return err
		}

		// If no error was encountered, return immediately
		if resp.StatusCode < http.StatusBadRequest {
			return nil
		}

		// Check to see if this status code is retriable
		if _, ok := rrt.RetriableStatusCodes[resp.StatusCode]; ok {
			log.Get(req.Context()).Debug("retrying retriable http request", zap.Int("http.status_code", resp.StatusCode))
			return fmt.Errorf("status code `%v` is retriable", resp.StatusCode)
		}

		// The status code is not retriable
		log.Get(req.Context()).Debug("could not retry failed http request", zap.Int("http.status_code", resp.StatusCode))
		return nil
	}

	// Each backoff policy contains state, so unfortunately we must create a fresh backoff
	// policy for every request
	expBackOff := backoff.NewExponentialBackOff()
	expBackOff.InitialInterval = rrt.InitialInterval
	expBackOff.Multiplier = rrt.Multiplier
	expBackOff.MaxInterval = rrt.MaxInterval
	expBackOff.RandomizationFactor = rrt.RandomizationFactor
	backoffPolicy := backoff.WithContext(
		backoff.WithMaxRetries(expBackOff, uint64(rrt.MaxRetries)),
		req.Context(),
	)
	if retryErr := backoff.Retry(makeRequestRetriable, backoffPolicy); retryErr != nil {
		log.Get(req.Context()).Debug("failed retrying http request")
	}
	return resp, err
}

// CircuitBreakerRoundTripper wraps a RoundTrapper with circuit-breaker logic
type CircuitBreakerRoundTripper struct {
	RoundTripper http.RoundTripper
	// A map of hostname to Hystrix configuration settings. Default settings will be used if not
	// specified. Using the default is recommended unless you have a good reason to alter the
	// settings.
	HostConfiguration map[string]hystrix.CommandConfig
	registeredHosts   map[string]bool
	defaultConfig     hystrix.CommandConfig
}

// NewDefaultCircuitBreakerRoundTripper constructs and returns the default
// CircuitBreakerRoundTripper configuration.
func NewDefaultCircuitBreakerRoundTripper(roundTripper http.RoundTripper) CircuitBreakerRoundTripper {
	return CircuitBreakerRoundTripper{
		RoundTripper:      roundTripper,
		HostConfiguration: make(map[string]hystrix.CommandConfig),
		registeredHosts:   make(map[string]bool),
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
	var err error
	makeRequestFunc := func(ctx context.Context) error {
		resp, err = cbrt.RoundTripper.RoundTrip(req)
		if err != nil || resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
			return fmt.Errorf("failed request, invoking circuit-breaker")
		}
		return nil
	}

	// Register the host configuration if it is not yet registered
	if _, ok := cbrt.registeredHosts[req.URL.Host]; !ok {
		hystrixConfig, configExists := cbrt.HostConfiguration[req.URL.Host]
		if !configExists {
			hystrixConfig = cbrt.defaultConfig
		}
		hystrix.ConfigureCommand(req.URL.Host, hystrixConfig)
		cbrt.registeredHosts[req.URL.Host] = true
	}

	if cbErr := hystrix.DoC(req.Context(), req.URL.Host, makeRequestFunc, nil); cbErr != nil {
		switch cbErrTyped := cbErr.(type) {
		// In cases where the returned error type is a circuit-breaker error, we want to return the
		// specific error type instead of the HTTP error. This allows upstream calls to
		// appropriately handle the failure
		case hystrix.CircuitError:
			log.Get(req.Context()).Debug("circuit-breaker call failed", zap.String("reason", cbErrTyped.Message))
			err = cbErr
		}
	}
	return resp, err
}
