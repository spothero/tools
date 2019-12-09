// Copyright 2019 SpotHero
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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetHTTPMiddleware(t *testing.T) {
	tests := []struct {
		name                    string
		authHeaderPresent       bool
		authHeader              string
		jwt                     string
		parseJWTError           bool
		expectClaim             bool
		authRequired            bool
		expectedStatusCode      int
		expectedHeaders         map[string]string
		expectNextHandlerCalled bool
	}{
		{
			"no auth header results in no claim, auth not required, next handler called",
			false,
			"",
			"",
			false,
			false,
			false,
			-1,
			nil,
			true,
		}, {
			"malformed auth headers are rejected, auth not required, next handler called",
			true,
			"bearer fake.jwt.header",
			"",
			false,
			false,
			false,
			-1,
			nil,
			true,
		}, {
			"failed jwt parsings are rejected, auth not required, next handler called",
			true,
			"Bearer fake.jwt.header",
			"fake.jwt.header",
			true,
			false,
			false,
			-1,
			nil,
			true,
		}, {
			"with auth: no auth header results in no claim and a 401",
			false,
			"",
			"",
			false,
			false,
			true,
			401,
			map[string]string{"WWW-Authenticate": "Bearer"},
			false,
		}, {
			"malformed auth headers are rejected, auth required",
			true,
			"bearer fake.jwt.header",
			"",
			false,
			false,
			true,
			401,
			map[string]string{"WWW-Authenticate": "Bearer"},
			false,
		}, {
			"failed jwt parsings are rejected, auth required",
			true,
			"Bearer fake.jwt.header",
			"fake.jwt.header",
			true,
			false,
			true,
			403,
			nil,
			false,
		}, {
			"failed jwt parsings, auth not required, next handler called",
			true,
			"Bearer fake.jwt.header",
			"fake.jwt.header",
			true,
			false,
			false,
			-1,
			nil,
			true,
		}, {
			"jwt tokens are parsed and placed in context when present",
			true,
			"Bearer fake.jwt.header",
			"fake.jwt.header",
			false,
			true,
			false,
			-1,
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

			joseMiddleware := GetHTTPMiddleware(handler, test.authRequired)
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

			if test.authRequired {
				assert.Equal(t, test.expectedStatusCode, httpRespResult.StatusCode)
				for expectedHeader, expectedValue := range test.expectedHeaders {
					fmt.Printf("%+v\n", httpRespResult)
					assert.Equal(t, expectedValue, httpRespResult.Header.Get(expectedHeader))
				}
			}

			assert.Equal(t, test.expectNextHandlerCalled, testHandlerCalled)
		})
	}
}
