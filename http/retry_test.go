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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spothero/tools/http/mock"
	"github.com/stretchr/testify/assert"
)

func TestNewDefaultRetryRoundTripper(t *testing.T) {
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
			&mock.RoundTripper{ResponseStatusCodes: []int{http.StatusOK}, CreateErr: false},
			false,
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
			&mock.RoundTripper{ResponseStatusCodes: []int{http.StatusOK}, CreateErr: false},
			http.StatusOK,
			0,
			false,
			false,
		},
		{
			"round tripper with an unresolved error returns an error",
			&mock.RoundTripper{
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
			&mock.RoundTripper{
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
			&mock.RoundTripper{
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
			&mock.RoundTripper{
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
