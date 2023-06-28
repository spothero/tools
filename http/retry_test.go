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
	"time"

	"github.com/spothero/tools/http/mock"
	"github.com/stretchr/testify/assert"
)

func TestNewDefaultRetryRoundTripper(t *testing.T) {
	tests := []struct {
		roundTripper http.RoundTripper
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
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.expectPanic {
				assert.Panics(t, func() {
					NewDefaultRetryRoundTripper(test.roundTripper)
				})
			} else {
				assert.Equal(
					t,
					RetryRoundTripper{
						RoundTripper: test.roundTripper,
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
					},
					NewDefaultRetryRoundTripper(test.roundTripper),
				)
			}
		})
	}
}

func TestRetryRoundTrip(t *testing.T) {
	tests := []struct {
		roundTripper       http.RoundTripper
		name               string
		expectedStatusCode int
		numRetries         uint8
		expectErr          bool
		expectPanic        bool
	}{
		{
			name:               "no round tripper results in a panic",
			expectedStatusCode: http.StatusOK, // doesn't matter
			expectPanic:        true,
		},
		{
			name:               "round tripper with no error invokes middleware correctly",
			roundTripper:       &mock.RoundTripper{ResponseStatusCodes: []int{http.StatusOK}, CreateErr: false},
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "round tripper with an unresolved error returns an error",
			roundTripper: &mock.RoundTripper{
				ResponseStatusCodes: []int{
					http.StatusInternalServerError,
					http.StatusInternalServerError,
				},
				CreateErr: false,
			},
			expectedStatusCode: http.StatusInternalServerError,
			numRetries:         1,
		},
		{
			name: "round tripper with an unretriable error returns an error",
			roundTripper: &mock.RoundTripper{
				ResponseStatusCodes: []int{
					http.StatusNotImplemented,
				},
				CreateErr: false,
			},
			expectedStatusCode: http.StatusNotImplemented,
			numRetries:         1,
		},
		{
			name: "round tripper that encounters an http err is retried",
			roundTripper: &mock.RoundTripper{
				ResponseStatusCodes: []int{
					http.StatusBadRequest,
					http.StatusBadRequest,
				},
				CreateErr: true,
			},
			expectedStatusCode: http.StatusBadRequest,
			numRetries:         1,
			expectErr:          true,
		},
		{
			name: "retries are stopped when a successful or non-retriable status code is given",
			roundTripper: &mock.RoundTripper{
				ResponseStatusCodes: []int{
					http.StatusInternalServerError,
					http.StatusOK,
					http.StatusInternalServerError,
				},
				CreateErr: true,
			},
			expectedStatusCode: http.StatusOK,
			numRetries:         2,
			expectErr:          true,
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
