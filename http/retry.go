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
	"fmt"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/spothero/tools/log"
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

// NewDefaultRetryRoundTripper constructs and returns the default RetryRoundTripper configuration.
//
// By default, the round tripper provides exponential backoff on [500-504] errors. The default
// configuration for exponential backoff is to start with an interval of 100 milliseconds, a
// multiplier of two, a randomization factor of up to 0.5 milliseconds (for jitter), a max
// interval of 10 seconds, and finally, the retry will attempt 5 times before failing if the
// error is retriable.
func NewDefaultRetryRoundTripper(roundTripper http.RoundTripper) RetryRoundTripper {
	// Ensure the RoundTripper was set on the CircuitBreakerRoundTripper
	if roundTripper == nil {
		panic("no roundtripper provided to retry round tripper")
	}
	return RetryRoundTripper{
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
