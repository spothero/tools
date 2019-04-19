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

package tools

import (
	"net/http"
	"net/http/httptest"
	"testing"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/spothero/tools/http/writer"
	"github.com/stretchr/testify/assert"
	jaeger "github.com/uber/jaeger-client-go"
)

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
