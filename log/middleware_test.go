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

package log

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spothero/tools/http/writer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestHTTPMiddleware(t *testing.T) {
	// Override the global logger with the observable
	core, recordedLogs := observer.New(zapcore.InfoLevel)
	c := &Config{Cores: []zapcore.Core{core}}
	err := c.InitializeLogger()
	require.NoError(t, err)
	logger = zap.New(core)

	// setup a test server with logging middleware and a handler that sets the status code
	const statusCode = 666
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		// make sure the middleware placed the logger in the request context
		_, ok := r.Context().Value(logKey).(*zap.Logger)
		assert.True(t, ok)
	})
	testServer := httptest.NewServer(writer.StatusRecorderMiddleware(HTTPMiddleware(testHandler)))
	defer testServer.Close()
	res, err := http.Get(testServer.URL)
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
	assert.ElementsMatch(t, []string{"http_method", "path", "query_string"}, foundLogKeysRequest)

	// Test that response parameters are appropriately logged to our standards
	foundLogKeysResponse := make([]string, len(currLogs[1].Context))
	for idx, field := range currLogs[1].Context {
		foundLogKeysResponse[idx] = field.Key
	}
	assert.ElementsMatch(t, []string{"response_code"}, foundLogKeysResponse)
	assert.Equal(t, currLogs[1].Context[0].Integer, int64(statusCode))
}

func TestSQLMiddleware(t *testing.T) {
	tests := []struct {
		name                string
		logLevel            zapcore.Level
		queryName           string
		query               string
		numLogsStartExpect  int
		numLogsEndExpect    int
		numErrLogsEndExpect int
		expectErr           bool
	}{
		{
			"nothing is logged without an error at info level",
			zapcore.InfoLevel,
			"test-query-name",
			"test-query",
			0,
			0,
			0,
			false,
		},
		{
			"errors are logged at info level",
			zapcore.InfoLevel,
			"test-query-name",
			"test-query",
			0,
			0,
			1,
			true,
		},
		{
			"debug logs are captured",
			zapcore.DebugLevel,
			"test-query-name",
			"test-query",
			1,
			1,
			0,
			false,
		},
		{
			"error logs are captured in the end middleware",
			zapcore.DebugLevel,
			"test-query-name",
			"test-query",
			1,
			0,
			1,
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Override the global logger with the observable
			core, recordedLogs := observer.New(test.logLevel)
			c := &Config{Cores: []zapcore.Core{core}}
			err := c.InitializeLogger()
			assert.NoError(t, err)
			logger = zap.New(core)

			// Set expectations
			expectedLevel := test.logLevel
			expectedLabels := []string{"query_name", "query"}
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
