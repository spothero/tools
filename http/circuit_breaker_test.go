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
	"net/http/httptest"
	"testing"

	"github.com/cep21/circuit/v3"
	"github.com/spothero/tools/http/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultCircuitBreakerRoundTripper(t *testing.T) {
	tests := []struct {
		roundTripper http.RoundTripper
		config       map[string]circuit.Config
		name         string
		expectPanic  bool
	}{
		{
			name:        "no round tripper leads to a panic",
			expectPanic: true,
		},
		{
			name:         "the retry default round tripper is correctly created",
			roundTripper: &mock.RoundTripper{ResponseStatusCodes: []int{http.StatusOK}, CreateErr: false},
		},
		{
			name:         "host configuration is honored",
			roundTripper: &mock.RoundTripper{ResponseStatusCodes: []int{http.StatusOK}, CreateErr: false},
			config: map[string]circuit.Config{
				"host": {},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.expectPanic {
				assert.Panics(t, func() {
					_ = NewDefaultCircuitBreakerRoundTripper(test.roundTripper, nil)
				})
			} else {
				cbrt := NewDefaultCircuitBreakerRoundTripper(test.roundTripper, test.config)
				assert.Equal(t, test.roundTripper, cbrt.RoundTripper)
				if test.config != nil {
					cb := cbrt.manager.GetCircuit("host")
					assert.NotNil(t, cb)
				}
			}
		})
	}
}

func TestCircuitBreakerRoundTrip(t *testing.T) {
	tests := []struct {
		roundTripper       http.RoundTripper
		name               string
		expectedStatusCode int
		circuitOpened      bool
		expectErr          bool
		expectPanic        bool
	}{
		{
			name:               "no round tripper results in a panic",
			expectedStatusCode: http.StatusOK,
			expectPanic:        true,
		},
		{
			name:               "round tripper with no error invokes middleware correctly",
			roundTripper:       &mock.RoundTripper{ResponseStatusCodes: []int{http.StatusOK}, CreateErr: false},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "bad requests are counted in the circuit breaker",
			roundTripper:       &mock.RoundTripper{ResponseStatusCodes: []int{http.StatusInternalServerError}, CreateErr: false},
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name: "round tripper opens the circuit breaker when enough errors are encountered",
			roundTripper: &mock.RoundTripper{
				ResponseStatusCodes: []int{http.StatusInternalServerError},
				CreateErr:           false,
			},
			expectedStatusCode: http.StatusInternalServerError,
			circuitOpened:      true,
			expectErr:          true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cbrt := CircuitBreakerRoundTripper{
				RoundTripper: test.roundTripper,
				manager:      circuit.Manager{},
			}
			if test.circuitOpened {
				cb := cbrt.manager.MustCreateCircuit("")
				cb.OpenCircuit()
			}
			if !test.expectPanic {
				mockReq := httptest.NewRequest("GET", "/path", nil)
				resp, err := cbrt.RoundTrip(mockReq)
				if test.expectErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					require.NotNil(t, resp)
					assert.Equal(t, test.expectedStatusCode, resp.StatusCode)
				}
			} else {
				assert.Panics(t, func() {
					_, _ = cbrt.RoundTrip(httptest.NewRequest("GET", "/path", nil))
				})
			}
		})
	}
}
