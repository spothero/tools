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

package log

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spothero/tools/http/mock"
	"github.com/spothero/tools/http/writer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestGetFields(t *testing.T) {
	mockReq := httptest.NewRequest("GET", "/path", nil)
	mockReq.Header.Set("Content-Length", "1")
	fields := getFields(mockReq)
	assert.Equal(t, 5, len(fields))
	keys := make([]string, 5)
	for idx, field := range fields {
		keys[idx] = field.Key
	}
	assert.ElementsMatch(t, keys, []string{"http.method", "http.url", "http.path", "http.user_agent", "http.content_length"})
}

func TestHTTPServerMiddleware(t *testing.T) {
	recordedLogs := makeLoggerObservable(t, zapcore.DebugLevel)

	// setup a test server with logging middleware and a handler that sets the status code
	const statusCode = 666
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		verifyLogContext(r.Context(), t)
	})
	testServer := httptest.NewServer(writer.StatusRecorderMiddleware(HTTPServerMiddleware(testHandler)))
	defer testServer.Close()
	res, err := http.Post(testServer.URL, "text/plain", nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	defer res.Body.Close()

	// Test that request parameters are appropriately logged to our standards
	currLogs := recordedLogs.All()
	assert.Len(t, currLogs, 2)
	foundLogKeysRequest := make([]string, len(currLogs[0].Context))
	for idx, field := range currLogs[0].Context {
		foundLogKeysRequest[idx] = field.Key
	}
	assert.ElementsMatch(t, []string{"http.method", "http.url", "http.path", "http.user_agent", "http.content_length"}, foundLogKeysRequest)

	// Test that response parameters are appropriately logged to our standards
	foundLogKeysResponse := make([]string, len(currLogs[1].Context))
	for idx, field := range currLogs[1].Context {
		foundLogKeysResponse[idx] = field.Key
	}
	assert.ElementsMatch(t, []string{"http.url", "http.method", "http.path", "http.status_code", "http.duration", "http.user_agent", "http.content_length"}, foundLogKeysResponse)
	assert.Equal(t, currLogs[1].Context[len(currLogs[1].Context)-2].Integer, int64(statusCode))
}

func TestRoundTripper(t *testing.T) {
	tests := []struct {
		roundTripper http.RoundTripper
		name         string
		expectErr    bool
		expectPanic  bool
	}{
		{
			name:        "no round tripper results in a panic",
			expectPanic: true,
		},
		{
			name:         "roundtripper errors are returned to the caller",
			roundTripper: &mock.RoundTripper{ResponseStatusCodes: []int{http.StatusOK}, CreateErr: true},
			expectErr:    true,
		},
		{
			name:         "requests are logged appropriately in client calls",
			roundTripper: &mock.RoundTripper{ResponseStatusCodes: []int{http.StatusOK}, CreateErr: false},
		},
		{
			name: "circuit-breaking errors are logged",
			roundTripper: &mock.RoundTripper{
				ResponseStatusCodes: []int{http.StatusOK},
				CreateErr:           true,
				DesiredErr:          mock.CircuitError{CircuitOpened: true},
			},
			expectErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rt := RoundTripper{RoundTripper: test.roundTripper}
			if test.expectPanic {
				assert.Panics(t, func() {
					_, _ = rt.RoundTrip(nil)
				})
			} else if test.expectErr {
				resp, err := rt.RoundTrip(httptest.NewRequest("GET", "/path", nil))
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				recordedLogs := makeLoggerObservable(t, zapcore.DebugLevel)

				mockReq := httptest.NewRequest("GET", "/path", nil)
				mockReq.Header.Set("Content-Length", "1")
				resp, err := rt.RoundTrip(mockReq)
				assert.NoError(t, err)
				assert.NotNil(t, resp)

				// Test that request parameters are appropriately logged to our standards
				currLogs := recordedLogs.All()
				assert.Len(t, currLogs, 2)
				foundLogKeysRequest := make([]string, len(currLogs[0].Context))
				for idx, field := range currLogs[0].Context {
					foundLogKeysRequest[idx] = field.Key
				}
				assert.ElementsMatch(t, []string{"http.method", "http.url", "http.path", "http.user_agent", "http.content_length"}, foundLogKeysRequest)

				// Test that response parameters are appropriately logged to our standards
				foundLogKeysResponse := make([]string, len(currLogs[1].Context))
				for idx, field := range currLogs[1].Context {
					foundLogKeysResponse[idx] = field.Key
				}
				assert.ElementsMatch(t, []string{"http.url", "http.path", "http.method", "http.status_code", "http.user_agent", "http.duration", "http.content_length"}, foundLogKeysResponse)
				assert.Equal(t, currLogs[1].Context[len(currLogs[1].Context)-2].Integer, int64(http.StatusOK))
			}
		})
	}
}

func TestSQLMiddleware(t *testing.T) {
	tests := []struct {
		name                string
		queryName           string
		query               string
		numLogsStartExpect  int
		numLogsEndExpect    int
		numErrLogsEndExpect int
		logLevel            zapcore.Level
		expectErr           bool
	}{
		{
			name:      "nothing is logged without an error at info level",
			logLevel:  zapcore.InfoLevel,
			queryName: "test-query-name",
			query:     "test-query",
		},
		{
			name:                "errors are logged at info level",
			logLevel:            zapcore.InfoLevel,
			queryName:           "test-query-name",
			query:               "test-query",
			numErrLogsEndExpect: 1,
			expectErr:           true,
		},
		{
			name:               "debug logs are captured",
			logLevel:           zapcore.DebugLevel,
			queryName:          "test-query-name",
			query:              "test-query",
			numLogsStartExpect: 1,
			numLogsEndExpect:   1,
		},
		{
			name:                "error logs are captured in the end middleware",
			logLevel:            zapcore.DebugLevel,
			queryName:           "test-query-name",
			query:               "test-query",
			numLogsStartExpect:  1,
			numErrLogsEndExpect: 1,
			expectErr:           true,
		},
		{
			name:               "unnamed queries are still logged in the end middleware",
			logLevel:           zapcore.DebugLevel,
			query:              "test-query",
			numLogsStartExpect: 1,
			numLogsEndExpect:   1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recordedLogs := makeLoggerObservable(t, test.logLevel)

			// Set expectations
			expectedLevel := test.logLevel
			expectedLabels := []string{"query"}
			if test.queryName != "" {
				expectedLabels = append(expectedLabels, "query_name")
			}
			expectedLabelsEnd := expectedLabels
			if test.numErrLogsEndExpect > 0 {
				expectedLevel = zapcore.ErrorLevel
				expectedLabelsEnd = append(expectedLabels, "error")
			}

			// Test the SQLMiddleware
			ctx := context.Background()
			ctx, mwEnd, err := SQLMiddleware(ctx, test.queryName, test.query)
			assert.NotNil(t, ctx)
			assert.NotNil(t, mwEnd)
			assert.NoError(t, err)

			// Verify log contents from middleware open
			currLogs := recordedLogs.All()
			assert.Len(t, currLogs, test.numLogsStartExpect)

			if test.numLogsStartExpect > 0 {
				assert.Equal(t, test.logLevel, currLogs[0].Entry.Level)
				foundLogKeysRequest := make([]string, len(currLogs[0].Context))
				for idx, field := range currLogs[0].Context {
					foundLogKeysRequest[idx] = field.Key
				}
				assert.ElementsMatch(
					t,
					expectedLabels,
					foundLogKeysRequest,
				)
			}

			var queryErr error
			if test.expectErr {
				queryErr = fmt.Errorf("query error")
			}
			ctx, err = mwEnd(ctx, test.queryName, test.query, queryErr)
			assert.NotNil(t, ctx)
			assert.NoError(t, err)

			currLogs = recordedLogs.All()
			expectedLogs := test.numLogsStartExpect + test.numLogsEndExpect + test.numErrLogsEndExpect
			assert.Len(t, currLogs, expectedLogs)
			if expectedLogs > 1 {
				logIdx := test.numLogsStartExpect
				assert.Equal(t, expectedLevel, currLogs[logIdx].Entry.Level)
				foundLogKeysRequest := make([]string, len(currLogs[logIdx].Context))
				for idx, field := range currLogs[1].Context {
					foundLogKeysRequest[idx] = field.Key
				}
				assert.ElementsMatch(
					t,
					expectedLabelsEnd,
					foundLogKeysRequest,
				)
			}
		})
	}
}
