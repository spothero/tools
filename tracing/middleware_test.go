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
	jaeger "github.com/uber/jaeger-client-go"
)

func TestHTTPMiddleware(t *testing.T) {
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
			sr := writer.StatusRecorder{ResponseWriter: recorder, StatusCode: http.StatusOK}
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
			deferable, r := HTTPMiddleware(&sr, req)
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
