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
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/spothero/tools/log"
	"github.com/stretchr/testify/assert"
	jaeger "github.com/uber/jaeger-client-go"
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

func TestTracingMiddleware(t *testing.T) {
	tests := []struct {
		name              string
		withExistingTrace bool
		statusCode        int
	}{
		{
			"tracing middleware without an incoming trace creates a new trace",
			false,
			http.StatusOK,
		},
		{
			"tracing middleware with an incoming trace reuses the trace",
			true,
			http.StatusInternalServerError,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			sr := StatusRecorder{recorder, http.StatusOK}
			req, err := http.NewRequest("GET", "/", nil)
			assert.NoError(t, err)

			tracer, closer := jaeger.NewTracer("t", jaeger.NewConstSampler(false), jaeger.NewInMemoryReporter())
			defer closer.Close()
			opentracing.SetGlobalTracer(tracer)

			// Configure a preset span and place in request context
			var rootSpanCtx opentracing.SpanContext
			if test.withExistingTrace {
				span, spanCtx := opentracing.StartSpanFromContext(req.Context(), "test")
				rootSpanCtx = span.Context()
				req = req.WithContext(spanCtx)
			}
			deferable, r := TracingMiddleware(&sr, req)
			assert.NotNil(t, r)

			sr.StatusCode = test.statusCode
			deferable()

			// Test that the span context is returned
			requestSpan := opentracing.SpanFromContext(r.Context())
			if spanCtx, ok := requestSpan.Context().(jaeger.SpanContext); ok {
				if test.withExistingTrace {
					assert.NotNil(t, rootSpanCtx)
					if rootJaegerSpanCtx, ok := rootSpanCtx.(jaeger.SpanContext); ok {
						assert.Equal(t, rootJaegerSpanCtx.TraceID(), spanCtx.TraceID())
					} else {
						assert.FailNow(t, "unable to extract root jaeger span from span context")
					}
				}
			} else {
				assert.FailNow(t, "unable to extract jaeger span from span context")
			}
		})
	}
}
