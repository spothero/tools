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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spothero/tools/http/utils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoggingMiddleware(t *testing.T) {
	recorder := httptest.NewRecorder()
	sr := utils.StatusRecorder{ResponseWriter: recorder, StatusCode: http.StatusOK}
	req, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)

	// Override the global logger with the observable
	core, recordedLogs := observer.New(zapcore.InfoLevel)
	c := &Config{Cores: []zapcore.Core{core}}
	c.InitializeLogger()
	logger = zap.New(core)

	deferable, r := LoggingMiddleware(&sr, req)

	// Test that request parameters are appropriately logged to our standards
	assert.NotNil(t, r)
	currLogs := recordedLogs.All()
	assert.Len(t, currLogs, 1)
	foundLogKeysRequest := make([]string, len(currLogs[0].Context))
	for idx, field := range currLogs[0].Context {
		foundLogKeysRequest[idx] = field.Key
	}
	assert.ElementsMatch(
		t,
		[]string{"remote_address", "http_method", "path", "query_string", "hostname", "port"},
		foundLogKeysRequest,
	)

	// Test that response parameters are appropriately logged to our standards
	deferable()
	currLogs = recordedLogs.All()
	assert.Len(t, currLogs, 2)
	foundLogKeysResponse := make([]string, len(currLogs[1].Context))
	for idx, field := range currLogs[1].Context {
		foundLogKeysResponse[idx] = field.Key
	}
	assert.ElementsMatch(
		t,
		[]string{"remote_address", "hostname", "port", "response_code"},
		foundLogKeysResponse,
	)
}
