// Copyright 2022 SpotHero
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
	"github.com/spothero/tools/http/mock"
	"github.com/spothero/tools/http/writer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetSpanTags(t *testing.T) {
	span, _ := StartSpanFromContext(context.Background(), "test")

	mockReq := httptest.NewRequest("POST", "/path", nil)
	mockReq.Header.Set("Content-Length", "1")

	// There's not much we can test here since we can't access the underlying tags
	span = setSpanTags(mockReq, span)
	assert.NotNil(t, span)
}

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
			shutdown, _ := GetTracerProvider()
			ctx := context.Background()
			defer func() {
				if err := shutdown(ctx); err != nil {
					assert.Error(t, err)
				}
			}()

			// Configure a preset span and place in request context
			var rootSpanCtx trace.SpanContext
			existingSpanMiddleware := func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if test.withExistingTrace {
						span, spanCtx := StartSpanFromContext(r.Context(), "test")
						rootSpanCtx = span.SpanContext()
						r = r.WithContext(spanCtx)
					}
					next.ServeHTTP(w, r)
				})
			}

			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestSpan := trace.SpanFromContext(r.Context())
				spanCtx := requestSpan.SpanContext()
				if test.withExistingTrace {
					assert.NotNil(t, rootSpanCtx)
					rootJaegerSpanCtx := rootSpanCtx
					assert.Equal(t, rootJaegerSpanCtx.TraceID(), spanCtx.TraceID())

					correlationId, ok := r.Context().Value(CorrelationIDCtxKey).(string)
					assert.Equal(t, true, ok)
					assert.NotNil(t, correlationId)
					assert.NotEqual(t, "", correlationId)
				}
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

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name         string
		roundTripper http.RoundTripper
		expectErr    bool
		expectPanic  bool
	}{
		{
			"no round tripper results in a panic",
			nil,
			false,
			true,
		},
		{
			"roundtripper errors are returned to the caller",
			&mock.RoundTripper{ResponseStatusCodes: []int{http.StatusOK}, CreateErr: true},
			true,
			false,
		},
		{
			"successful requests are traced appropriately in client calls",
			&mock.RoundTripper{ResponseStatusCodes: []int{http.StatusOK}, CreateErr: false},
			false,
			false,
		},
		{
			"failed requests are traced appropriately in client calls",
			&mock.RoundTripper{ResponseStatusCodes: []int{http.StatusInternalServerError}, CreateErr: false},
			false,
			false,
		},
		{
			"circuit-breaking errors are logged",
			&mock.RoundTripper{
				ResponseStatusCodes: []int{http.StatusOK},
				CreateErr:           true,
				DesiredErr:          mock.CircuitError{CircuitOpened: true},
			},
			true,
			false,
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
				shutdown, _ := GetTracerProvider()
				ctx := context.Background()
				defer func() {
					if err := shutdown(ctx); err != nil {
						assert.Error(t, err)
					}
				}()

				mockReq := httptest.NewRequest("GET", "/path", nil)
				resp, err := rt.RoundTrip(mockReq)
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			}
		})
	}
}

func TestGetCorrelationID(t *testing.T) {
	// first, assert a request through the HTTPServerMiddleware contains a context
	// which produces a meaningful result for GetCorrelationID()
	shutdown, _ := GetTracerProvider()
	ctx := context.Background()
	defer func() {
		if err := shutdown(ctx); err != nil {
			assert.Error(t, err)
		}
	}()

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
		args      []interface{}
	}{
		{
			"non-errored no queryname requests are successfully traced",
			"",
			"SELECT * FROM tests",
			false,
			nil,
		},
		{
			"non-errored with queryname requests are successfully traced",
			"getAllTests",
			"SELECT * FROM tests",
			false,
			nil,
		},
		{
			"non-errored with args requests are successfully traced",
			"getAllTests",
			"SELECT * FROM tests",
			false,
			[]interface{}{1, "test"},
		},
		{
			"errored requests are successfully traced and marked as errored",
			"",
			"SELECT * FROM tests",
			true,
			nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			shutdown, _ := GetTracerProvider()
			ctx := context.Background()
			defer func() {
				if err := shutdown(ctx); err != nil {
					assert.Error(t, err)
				}
			}()

			// Create a span and span context
			span, spanCtx := StartSpanFromContext(context.Background(), "test")
			jaegerSpanCtxStart := span.SpanContext()
			expectedTraceID := jaegerSpanCtxStart.TraceID()

			// Invoke the middleware
			spanCtx, mwEnd, err := SQLMiddleware(spanCtx, test.queryName, test.query, test.args)
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
			span = trace.SpanFromContext(spanCtx)
			jaegerSpanCtxEnd := span.SpanContext()
			endTraceID := jaegerSpanCtxEnd.TraceID()
			assert.Equal(t, expectedTraceID, endTraceID)
		})
	}
}
