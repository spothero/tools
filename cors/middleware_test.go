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

package cors

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetHTTPServerMiddleware(t *testing.T) {
	tests := []struct {
		expectedHeaders         map[string]string
		name                    string
		httpMethod              string
		config                  Config
		expectNextHandlerCalled bool
	}{
		{
			name: "GET request",
			config: Config{
				AllowedOrigins: "*",
				AllowedMethods: "POST, GET, OPTIONS, PUT, DELETE",
				AllowedHeaders: "*",
			},
			httpMethod: http.MethodGet,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "POST, GET, OPTIONS, PUT, DELETE",
				"Access-Control-Allow-Headers": "*",
			},
			expectNextHandlerCalled: true,
		}, {
			name:       "OPTIONS request",
			httpMethod: http.MethodOptions,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testHandlerCalled := false
			testHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				testHandlerCalled = true
			})

			corsMiddleware := test.config.GetHTTPServerMiddleware()
			testServer := httptest.NewServer(corsMiddleware(testHandler))
			defer testServer.Close()

			req, err := http.NewRequest(test.httpMethod, testServer.URL, nil)
			require.NoError(t, err)

			httpRespResult, err := (&http.Client{}).Do(req)
			require.NoError(t, err)
			require.NotNil(t, httpRespResult)
			defer httpRespResult.Body.Close()

			for expectedHeader, expectedValue := range test.expectedHeaders {
				fmt.Printf("%+v\n", httpRespResult)
				assert.Equal(t, expectedValue, httpRespResult.Header.Get(expectedHeader))
			}

			assert.Equal(t, test.expectNextHandlerCalled, testHandlerCalled)
		})
	}
}
