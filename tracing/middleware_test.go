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

package tracing

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/spothero/tools/http/writer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	jaeger "github.com/uber/jaeger-client-go"
)

func TestHTTPServerMiddleware(t *testing.T) {
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
			tracer, closer := jaeger.NewTracer("t", jaeger.NewConstSampler(false), jaeger.NewInMemoryReporter())
			defer closer.Close()
			opentracing.SetGlobalTracer(tracer)

			// Configure a preset span and place in request context
			var rootSpanCtx opentracing.SpanContext
			existingSpanMiddleware := func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if test.withExistingTrace {
						span, spanCtx := opentracing.StartSpanFromContext(r.Context(), "test")
						rootSpanCtx = span.Context()
						r = r.WithContext(spanCtx)
					}
					next.ServeHTTP(w, r)
				})
			}

			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

				correlationId, ok := r.Context().Value(CorrelationIDCtxKey).(string)
				assert.Equal(t, true, ok)
				assert.NotNil(t, correlationId)
				assert.NotEqual(t, "", correlationId)
			})

			testServer := httptest.NewServer(
				writer.StatusRecorderMiddleware(existingSpanMiddleware(HTTPServerMiddleware(testHandler))))
			defer testServer.Close()
			res, err := http.Get(testServer.URL)
			require.NoError(t, err)
			require.NotNil(t, res)
			defer res.Body.Close()
		})
	}
}

func TestHTTPClientMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{
			"tracing client middleware correctly records 2XX responses",
			http.StatusOK,
		},
		{
			"tracing client middleware correctly records 5XX responses",
			http.StatusInternalServerError,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tracer, closer := jaeger.NewTracer("t", jaeger.NewConstSampler(false), jaeger.NewInMemoryReporter())
			defer closer.Close()
			opentracing.SetGlobalTracer(tracer)

			mockReq := httptest.NewRequest("GET", "/path", nil)
			mockReq, respHandler, err := HTTPClientMiddleware(mockReq)
			assert.NoError(t, err)
			assert.NotNil(t, respHandler)

			correlationId, ok := mockReq.Context().Value(CorrelationIDCtxKey).(string)
			assert.Equal(t, true, ok)
			assert.NotNil(t, correlationId)
			assert.NotEqual(t, "", correlationId)

			assert.NoError(t, respHandler(&http.Response{StatusCode: test.statusCode}))
		})
	}
}

func TestGetCorrelationID(t *testing.T) {
	// first, assert a request through the HTTPServerMiddleware contains a context
	// which produces a meaningful result for GetCorrelationID()
	tracer, closer := jaeger.NewTracer("t", jaeger.NewConstSampler(false), jaeger.NewInMemoryReporter())
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationId, ok := r.Context().Value(CorrelationIDCtxKey).(string)
		assert.Equal(t, true, ok)
		assert.NotNil(t, correlationId)
		assert.NotEqual(t, "", correlationId)

		ctx := r.Context()
		_correlationId := GetCorrelationID(ctx)
		assert.NotNil(t, _correlationId)
		assert.NotEqual(t, "", _correlationId)
		assert.Equal(t, correlationId, _correlationId)
	})

	testServer := httptest.NewServer(writer.StatusRecorderMiddleware(HTTPServerMiddleware(testHandler)))
	defer testServer.Close()
	res, err := http.Get(testServer.URL)
	require.NoError(t, err)
	require.NotNil(t, res)
	defer res.Body.Close()

	// last, ensure a non-nil trivial string is returned when
	// GetCorrelationID() is passed a Context which does not contain a
	// correlation ID
	emptyCtx := context.Background()
	trivialString := GetCorrelationID(emptyCtx)
	assert.Equal(t, "", trivialString)
}

func TestSQLMiddleware(t *testing.T) {
	tests := []struct {
		name      string
		queryName string
		query     string
		expectErr bool
	}{
		{
			"non-errored no queryname requests are successfully traced",
			"",
			"SELECT * FROM tests",
			false,
		},
		{
			"non-errored with queryname requests are successfully traced",
			"getAllTests",
			"SELECT * FROM tests",
			false,
		},
		{
			"errored requests are successfully traced and marked as errored",
			"",
			"SELECT * FROM tests",
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tracer, closer := jaeger.NewTracer("t", jaeger.NewConstSampler(false), jaeger.NewInMemoryReporter())
			defer closer.Close()
			opentracing.SetGlobalTracer(tracer)

			// Create a span and span context
			span, spanCtx := opentracing.StartSpanFromContext(context.Background(), "test")
			jaegerSpanCtxStart, ok := span.Context().(jaeger.SpanContext)
			if !ok {
				assert.FailNow(t, "unable to convert opentracing to jaeger span")
			}
			expectedTraceID := jaegerSpanCtxStart.TraceID()

			// Invoke the middleware
			spanCtx, mwEnd, err := SQLMiddleware(spanCtx, test.queryName, test.query)
			assert.NotNil(t, spanCtx)
			assert.NotNil(t, mwEnd)
			assert.Nil(t, err)

			// Invoke the middleware end
			var queryErr error
			if test.expectErr {
				queryErr = fmt.Errorf("query error")
			}
			spanCtx, err = mwEnd(spanCtx, test.queryName, test.query, queryErr)
			assert.NotNil(t, spanCtx)
			assert.Nil(t, err)

			// Test that the span context is returned
			span = opentracing.SpanFromContext(spanCtx)
			if jaegerSpanCtxEnd, ok := span.Context().(jaeger.SpanContext); ok {

				assert.Equal(t, expectedTraceID, jaegerSpanCtxEnd.TraceID())
			} else {
				assert.FailNow(t, "unable to extract jaeger span from span context")
			}
		})
	}
}
