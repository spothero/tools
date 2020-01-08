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
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spothero/tools/http/roundtrip"
	"github.com/spothero/tools/jose"
	"github.com/spothero/tools/log"
	"github.com/spothero/tools/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultClient(t *testing.T) {
	client := NewDefaultClient(NewMetrics(prometheus.NewRegistry(), true), nil)
	assert.NotNil(t, client)
	jrt, ok := client.Transport.(jose.RoundTripper)
	assert.True(t, ok)

	mrt, ok := jrt.RoundTripper.(MetricsRoundTripper)
	assert.True(t, ok)

	lrt, ok := mrt.RoundTripper.(log.RoundTripper)
	assert.True(t, ok)

	trt, ok := lrt.RoundTripper.(tracing.RoundTripper)
	assert.True(t, ok)

	rrt, ok := trt.RoundTripper.(RetryRoundTripper)
	assert.True(t, ok)
	assert.Equal(t, http.DefaultTransport, rrt.RoundTripper)
}

func TestRetryRoundTrip(t *testing.T) {
	tests := []struct {
		name               string
		roundTripper       http.RoundTripper
		expectedStatusCode int
		numRetries         uint8
		expectErr          bool
		expectPanic        bool
	}{
		{
			"no round tripper results in a panic",
			nil,
			http.StatusOK, // doesn't matter
			0,
			false,
			true,
		},
		{
			"round tripper with no error invokes middleware correctly",
			&roundtrip.MockRoundTripper{ResponseStatusCodes: []int{http.StatusOK}, CreateErr: false},
			http.StatusOK,
			0,
			false,
			false,
		},
		{
			"round tripper with an unresolved error returns an error",
			&roundtrip.MockRoundTripper{
				ResponseStatusCodes: []int{
					http.StatusInternalServerError,
					http.StatusInternalServerError,
				},
				CreateErr: false,
			},
			http.StatusInternalServerError,
			1,
			false,
			false,
		},
		{
			"round tripper with an unretriable error returns an error",
			&roundtrip.MockRoundTripper{
				ResponseStatusCodes: []int{
					http.StatusNotImplemented,
				},
				CreateErr: false,
			},
			http.StatusNotImplemented,
			1,
			false,
			false,
		},
		{
			"round tripper that encounters an http err is retried",
			&roundtrip.MockRoundTripper{
				ResponseStatusCodes: []int{
					http.StatusBadRequest,
					http.StatusBadRequest,
				},
				CreateErr: true,
			},
			http.StatusBadRequest,
			1,
			true,
			false,
		},
		{
			"retries are stopped when a successful or non-retriable status code is given",
			&roundtrip.MockRoundTripper{
				ResponseStatusCodes: []int{
					http.StatusInternalServerError,
					http.StatusOK,
					http.StatusInternalServerError,
				},
				CreateErr: true,
			},
			http.StatusOK,
			2,
			true,
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rrt := RetryRoundTripper{
				RetriableStatusCodes: map[int]bool{http.StatusInternalServerError: true},
				MaxRetries:           test.numRetries,
				InitialInterval:      1 * time.Nanosecond,
				RoundTripper:         test.roundTripper,
			}
			mockReq := httptest.NewRequest("GET", "/path", nil)
			if !test.expectPanic {
				resp, err := rrt.RoundTrip(mockReq)
				if test.expectErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, resp)
					assert.Equal(t, test.expectedStatusCode, resp.StatusCode)
				}
			} else {
				assert.Panics(t, func() {
					_, _ = rrt.RoundTrip(mockReq)
				})
			}
		})
	}
}

func TestNewDefaultCircuitBreakerRoundTripper(t *testing.T) {
	rt := &roundtrip.MockRoundTripper{ResponseStatusCodes: []int{http.StatusOK}, CreateErr: false}
	cbrt := NewDefaultCircuitBreakerRoundTripper(rt)
	assert.Equal(t, rt, cbrt.RoundTripper)
	assert.Equal(t, make(map[string]hystrix.CommandConfig), cbrt.HostConfiguration)
	assert.Equal(t, make(map[string]bool), cbrt.registeredHosts)
	assert.Equal(
		t,
		hystrix.CommandConfig{
			Timeout:                int((30 * time.Second).Milliseconds()), // 30 second timeout
			MaxConcurrentRequests:  int(math.MaxInt32),                     // Do not limit concurrent requests by default
			RequestVolumeThreshold: hystrix.DefaultVolumeThreshold,
			SleepWindow:            hystrix.DefaultSleepWindow,
			ErrorPercentThreshold:  hystrix.DefaultErrorPercentThreshold,
		},
		cbrt.defaultConfig,
	)
}

func TestCircuitBreakerRoundTrip(t *testing.T) {
	tests := []struct {
		name                string
		roundTripper        http.RoundTripper
		expectedStatusCodes []int
		hystrixConfig       map[string]hystrix.CommandConfig
		numRequests         int
		expectErr           []bool
		expectPanic         bool
	}{
		{
			"no round tripper results in a panic",
			nil,
			[]int{http.StatusOK},
			map[string]hystrix.CommandConfig{},
			1,
			[]bool{false},
			true,
		},
		{
			"round tripper with no error invokes middleware correctly",
			&roundtrip.MockRoundTripper{ResponseStatusCodes: []int{http.StatusOK}, CreateErr: false},
			[]int{http.StatusOK},
			map[string]hystrix.CommandConfig{},
			1,
			[]bool{false},
			false,
		},
		{
			"round tripper opens the circuit breaker when enough errors are encountered",
			&roundtrip.MockRoundTripper{
				ResponseStatusCodes: []int{
					http.StatusBadRequest,
					http.StatusBadRequest,
				},
				CreateErr: false,
			},
			[]int{
				http.StatusBadRequest,
				http.StatusBadRequest,
			},
			map[string]hystrix.CommandConfig{
				"": hystrix.CommandConfig{
					Timeout:                1,
					MaxConcurrentRequests:  1,
					RequestVolumeThreshold: 1,
					SleepWindow:            1,
					ErrorPercentThreshold:  1,
				},
			},
			2,
			[]bool{false, true},
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cbrt := CircuitBreakerRoundTripper{
				RoundTripper:    test.roundTripper,
				registeredHosts: make(map[string]bool),
				defaultConfig: hystrix.CommandConfig{
					Timeout:                1,
					MaxConcurrentRequests:  1,
					RequestVolumeThreshold: 1,
					SleepWindow:            1,
					ErrorPercentThreshold:  1,
				},
				HostConfiguration: test.hystrixConfig,
			}
			if !test.expectPanic {
				for i := 0; i < test.numRequests; i++ {
					mockReq := httptest.NewRequest("GET", "/path", nil)
					resp, err := cbrt.RoundTrip(mockReq)
					if test.expectErr[i] {
						assert.Error(t, err)
					} else {
						assert.NoError(t, err)
						require.NotNil(t, resp)
						assert.Equal(t, test.expectedStatusCodes[i], resp.StatusCode)
					}
				}
			} else {
				assert.Panics(t, func() {
					_, _ = cbrt.RoundTrip(httptest.NewRequest("GET", "/path", nil))
				})
			}
		})
	}
}
