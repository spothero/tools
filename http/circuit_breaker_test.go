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
	"sync"
	"testing"
	"time"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/spothero/tools/http/roundtrip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultCircuitBreakerRoundTripper(t *testing.T) {
	tests := []struct {
		name         string
		roundTripper http.RoundTripper
		expectPanic  bool
	}{
		{
			"no round tripper leads to a panic",
			nil,
			true,
		},
		{
			"the retry default round tripper is correctly created",
			&roundtrip.MockRoundTripper{ResponseStatusCodes: []int{http.StatusOK}, CreateErr: false},
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.expectPanic {
				assert.Panics(t, func() {
					_ = NewDefaultCircuitBreakerRoundTripper(test.roundTripper)
				})
			} else {
				cbrt := NewDefaultCircuitBreakerRoundTripper(test.roundTripper)
				assert.Equal(t, test.roundTripper, cbrt.RoundTripper)
				assert.Equal(t, make(map[string]hystrix.CommandConfig), cbrt.hostConfiguration)
				assert.Equal(t, make(map[string]bool), cbrt.registeredHostsSet)
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
		})
	}
}

func TestWithHostConfiguration(t *testing.T) {
	cbrt := &CircuitBreakerRoundTripper{configMutex: sync.RWMutex{}}
	assert.Equal(
		t,
		cbrt.WithHostConfiguration(map[string]hystrix.CommandConfig{
			"host": hystrix.CommandConfig{},
		}).hostConfiguration,
		map[string]hystrix.CommandConfig{
			"host": hystrix.CommandConfig{},
		})

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
					http.StatusInternalServerError,
					http.StatusInternalServerError,
				},
				CreateErr: false,
			},
			[]int{
				http.StatusInternalServerError,
				http.StatusInternalServerError,
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
				RoundTripper:       test.roundTripper,
				registeredHostsSet: make(map[string]bool),
				defaultConfig: hystrix.CommandConfig{
					Timeout:                1,
					MaxConcurrentRequests:  1,
					RequestVolumeThreshold: 1,
					SleepWindow:            1,
					ErrorPercentThreshold:  1,
				},
				hostConfiguration: test.hystrixConfig,
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
