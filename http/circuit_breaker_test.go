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
		name         string
		roundTripper http.RoundTripper
		config       map[string]circuit.Config
		expectPanic  bool
	}{
		{
			"no round tripper leads to a panic",
			nil,
			nil,
			true,
		},
		{
			"the retry default round tripper is correctly created",
			&mock.RoundTripper{ResponseStatusCodes: []int{http.StatusOK}, CreateErr: false},
			nil,
			false,
		},
		{
			"host configuration is honored",
			&mock.RoundTripper{ResponseStatusCodes: []int{http.StatusOK}, CreateErr: false},
			map[string]circuit.Config{
				"host": {},
			},
			false,
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
		name               string
		roundTripper       http.RoundTripper
		expectedStatusCode int
		circuitOpened      bool
		expectErr          bool
		expectPanic        bool
	}{
		{
			"no round tripper results in a panic",
			nil,
			http.StatusOK,
			false,
			false,
			true,
		},
		{
			"round tripper with no error invokes middleware correctly",
			&mock.RoundTripper{ResponseStatusCodes: []int{http.StatusOK}, CreateErr: false},
			http.StatusOK,
			false,
			false,
			false,
		},
		{
			"bad requests are counted in the circuit breaker",
			&mock.RoundTripper{ResponseStatusCodes: []int{http.StatusInternalServerError}, CreateErr: false},
			http.StatusInternalServerError,
			false,
			false,
			false,
		},
		{
			"round tripper opens the circuit breaker when enough errors are encountered",
			&mock.RoundTripper{
				ResponseStatusCodes: []int{http.StatusInternalServerError},
				CreateErr:           false,
			},
			http.StatusInternalServerError,
			true,
			true,
			false,
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
