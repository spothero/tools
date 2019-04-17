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

package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/spothero/tools/log"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestWriteHeader(t *testing.T) {
	recorder := httptest.NewRecorder()
	sr := StatusRecorder{recorder, http.StatusNotImplemented}
	sr.WriteHeader(http.StatusOK)
	assert.Equal(t, sr.StatusCode, http.StatusOK)
	assert.Equal(t, recorder.Result().StatusCode, http.StatusOK)
}

func TestHandler(t *testing.T) {
	middlewareCalled := false
	deferableCalled := false
	mw := Middleware{
		func(sr *StatusRecorder, r *http.Request) (func(), *http.Request) {
			middlewareCalled = true
			return func() { deferableCalled = true }, r
		},
	}
	httpRec := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)
	http.HandlerFunc(mw.handler(mux.NewRouter())).ServeHTTP(httpRec, req)
	assert.True(t, middlewareCalled)
	assert.True(t, deferableCalled)
}

func TestLoggingMiddleware(t *testing.T) {
	recorder := httptest.NewRecorder()
	sr := StatusRecorder{recorder, http.StatusOK}
	req, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)

	// Override the global logger with the observable
	core, recordedLogs := observer.New(zapcore.InfoLevel)
	lc := &log.LoggingConfig{Cores: []zapcore.Core{core}}
	lc.InitializeLogger()
	logger := log.Get(context.Background())
	*logger = *zap.New(core)

	deferable, r := LoggingMiddleware(&sr, req)
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

func TestTracingMiddleware(t *testing.T) {

}
