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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spothero/tools/http/roundtrip"
	"github.com/stretchr/testify/assert"
)

func TestNewDefaultClient(t *testing.T) {
	client := NewDefaultClient(NewMetrics(prometheus.NewRegistry(), true), nil)
	assert.NotNil(t, client)
	mrt, ok := client.Transport.(MiddlewareRoundTripper)
	assert.True(t, ok)
	// TODO: FIX THIS
	//assert.Equal(t, 4, len(mrt.Middleware))

	rrt, ok := mrt.RoundTripper.(RetryRoundTripper)
	assert.True(t, ok)
	assert.Equal(t, http.DefaultTransport, rrt.RoundTripper)
}

func TestMiddlewareRoundTrip(t *testing.T) {
	tests := []struct {
		name           string
		roundTripper   http.RoundTripper
		middlewareErr  bool
		respHandlerErr bool
		expectErr      bool
		expectPanic    bool
	}{
		{
			"no round tripper results in a panic",
			nil,
			false,
			false,
			false,
			true,
		},
		{
			"round tripper with no error invokes middleware correctly",
			&roundtrip.MockRoundTripper{ResponseStatusCodes: []int{http.StatusOK}, CreateErr: false},
			false,
			false,
			false,
			false,
		},
		{
			"round tripper with middleware error returns an error",
			&roundtrip.MockRoundTripper{ResponseStatusCodes: []int{http.StatusOK}, CreateErr: false},
			true,
			false,
			true,
			false,
		},
		{
			"round tripper with internal roundtripper error returns an error",
			&roundtrip.MockRoundTripper{ResponseStatusCodes: []int{http.StatusOK}, CreateErr: true},
			false,
			false,
			true,
			false,
		},
		{
			"round tripper with resp handler error returns an error",
			&roundtrip.MockRoundTripper{ResponseStatusCodes: []int{http.StatusOK}, CreateErr: false},
			false,
			true,
			true,
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			middlewareCalled := false
			respHandlerCalled := false
			mockMiddleware := func(req *http.Request) (*http.Request, func(*http.Response) error, error) {
				middlewareCalled = true
				var mwErr error
				if test.middlewareErr {
					mwErr = fmt.Errorf("middleware error")
				}
				return req,
					func(*http.Response) error {
						respHandlerCalled = true
						var rhErr error
						if test.respHandlerErr {
							rhErr = fmt.Errorf("response handler error")
						}
						return rhErr
					}, mwErr
			}
			mrt := MiddlewareRoundTripper{
				RoundTripper: test.roundTripper,
				Middleware: []ClientMiddleware{
					mockMiddleware,
				},
			}
			mockReq := httptest.NewRequest("GET", "/path", nil)
			if !test.expectPanic {
				resp, err := mrt.RoundTrip(mockReq)
				if test.expectErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, resp)
					assert.True(t, middlewareCalled)
					assert.True(t, respHandlerCalled)
				}
			} else {
				assert.Panics(t, func() {
					_, _ = mrt.RoundTrip(mockReq)
				})
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
