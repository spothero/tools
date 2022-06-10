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

package jose

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spothero/tools/http/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetHTTPServerMiddleware(t *testing.T) {
	tests := []struct {
		name                    string
		authHeaderPresent       bool
		authHeader              string
		jwt                     string
		parseJWTError           bool
		expectClaim             bool
		expectedStatusCode      int
		expectedHeaders         map[string]string
		expectNextHandlerCalled bool
	}{
		{
			"no auth header results in no claim, next handler called",
			false,
			"",
			"",
			false,
			false,
			http.StatusOK,
			nil,
			true,
		}, {
			"malformed auth headers are rejected",
			true,
			"bearer fake.jwt.header",
			"",
			false,
			false,
			401,
			map[string]string{"WWW-Authenticate": "Bearer"},
			false,
		}, {
			"failed jwt parsings are rejected",
			true,
			"Bearer fake.jwt.header",
			"fake.jwt.header",
			true,
			false,
			403,
			nil,
			false,
		}, {
			"jwt tokens are parsed and placed in context when present",
			true,
			"Bearer fake.jwt.header",
			"fake.jwt.header",
			false,
			true,
			http.StatusOK,
			nil,
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := &MockHandler{
				claimGenerators: []ClaimGenerator{MockGenerator{}},
			}

			var parseErr error
			if test.parseJWTError {
				parseErr = fmt.Errorf("a jwt parsing error occurred in this test")
			}
			handler.On(
				"ParseValidateJWT",
				test.jwt,
				handler.GetClaims(),
			).Return(parseErr)

			testHandlerCalled := false
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				testHandlerCalled = true
				value, ok := r.Context().Value(MockClaimKey).(*MockClaim)
				if test.expectClaim {
					assert.True(t, ok)
					assert.Equal(t, &MockClaim{}, value)
				} else {
					assert.False(t, ok)
					assert.Nil(t, value)
				}
			})

			joseMiddleware := GetHTTPServerMiddleware(handler)
			testServer := httptest.NewServer(joseMiddleware(testHandler))
			defer testServer.Close()

			req, err := http.NewRequest(http.MethodGet, testServer.URL, nil)
			require.NoError(t, err)
			if test.authHeaderPresent {
				req.Header.Add(authHeader, test.authHeader)
			}
			httpRespResult, err := (&http.Client{}).Do(req)
			require.NoError(t, err)
			require.NotNil(t, httpRespResult)
			defer httpRespResult.Body.Close()

			assert.Equal(t, test.expectedStatusCode, httpRespResult.StatusCode)
			for expectedHeader, expectedValue := range test.expectedHeaders {
				fmt.Printf("%+v\n", httpRespResult)
				assert.Equal(t, expectedValue, httpRespResult.Header.Get(expectedHeader))
			}

			assert.Equal(t, test.expectNextHandlerCalled, testHandlerCalled)
		})
	}
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name         string
		roundTripper http.RoundTripper
		expectPanic  bool
	}{
		{
			"no round tripper results in a panic",
			nil,
			true,
		},
		{
			"if auth data is present in the context it is set on outbound requests",
			&mock.RoundTripper{ResponseStatusCodes: []int{http.StatusOK}, CreateErr: false},
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rt := RoundTripper{RoundTripper: test.roundTripper}
			if test.expectPanic {
				assert.Panics(t, func() {
					_, _ = rt.RoundTrip(nil)
				})
			} else {
				mockReq := httptest.NewRequest("GET", "/path", nil)
				mockReq = mockReq.WithContext(context.WithValue(mockReq.Context(), JWTClaimKey, "jwt"))
				resp, err := rt.RoundTrip(mockReq)
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			}
		})
	}
}

func TestEnforceAuthentication(t *testing.T) {
	tests := []struct {
		name                   string
		requestIsAuthenticated bool
		expectedAuthSuccess    bool
	}{
		{
			name:                   "authentication provided",
			requestIsAuthenticated: true,
			expectedAuthSuccess:    true,
		},
		{
			name:                   "authentication omitted",
			requestIsAuthenticated: false,
			expectedAuthSuccess:    false,
		},
	}

	for _, test := range tests {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}

		authenticatedHandler := EnforceAuthentication(handler)

		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			reqCtx := context.Background()

			if test.requestIsAuthenticated {
				reqCtx = context.WithValue(reqCtx, JWTClaimKey, true)
			}

			request, err := http.NewRequestWithContext(reqCtx, "GET", "url", nil)
			assert.NoError(err)
			responseRecorder := httptest.NewRecorder()
			authenticatedHandler(responseRecorder, request)
			actualResponse := responseRecorder.Result()

			if test.expectedAuthSuccess {
				assert.Equal(http.StatusOK, actualResponse.StatusCode)
			} else {
				assert.Equal(http.StatusUnauthorized, actualResponse.StatusCode)
			}
		})
	}
}

func TestEnforceAuthenticationWithAuthorization(t *testing.T) {
	tests := []struct {
		name                   string
		requestIsAuthenticated bool
		requestHasClaim        bool
		expectedAuthSuccess    bool
		authParams             AuthParams
		authClaim              Auth0Claim
	}{
		{
			name:                   "no authorization needed",
			requestIsAuthenticated: true,
			requestHasClaim:        true,
			expectedAuthSuccess:    true,
			authParams:             AuthParams{},
			authClaim:              Auth0Claim{},
		},
		{
			name:                   "cannot find claim",
			requestIsAuthenticated: true,
			requestHasClaim:        false,
			expectedAuthSuccess:    false,
			authParams:             AuthParams{RequiredScopes: []string{"update:service"}},
			authClaim:              Auth0Claim{},
		},
		{
			name:                   "missing required scope",
			requestIsAuthenticated: true,
			requestHasClaim:        true,
			expectedAuthSuccess:    false,
			authParams:             AuthParams{RequiredScopes: []string{"update:service"}},
			authClaim:              Auth0Claim{Scope: "read:service"},
		},
		{
			name:                   "has partial scope",
			requestIsAuthenticated: true,
			requestHasClaim:        true,
			expectedAuthSuccess:    false,
			authParams:             AuthParams{RequiredScopes: []string{"update:service read:service"}},
			authClaim:              Auth0Claim{Scope: "read:service"},
		},
		{
			name:                   "has required scope",
			requestIsAuthenticated: true,
			requestHasClaim:        true,
			expectedAuthSuccess:    true,
			authParams:             AuthParams{RequiredScopes: []string{"update:service"}},
			authClaim:              Auth0Claim{Scope: "update:service"},
		},
	}

	for _, test := range tests {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}

		authenticatedHandler := EnforceAuthenticationWithAuthorization(handler, test.authParams)

		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			reqCtx := context.Background()

			if test.requestIsAuthenticated {
				reqCtx = context.WithValue(reqCtx, JWTClaimKey, true)
			}

			if test.requestHasClaim {
				reqCtx = test.authClaim.NewContext(reqCtx)
			}

			request, err := http.NewRequestWithContext(reqCtx, "GET", "url", nil)
			assert.NoError(err)
			responseRecorder := httptest.NewRecorder()
			authenticatedHandler(responseRecorder, request)
			actualResponse := responseRecorder.Result()

			if test.expectedAuthSuccess {
				assert.Equal(http.StatusOK, actualResponse.StatusCode)
			} else {
				assert.Equal(http.StatusForbidden, actualResponse.StatusCode)
			}
		})
	}
}
