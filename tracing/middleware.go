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
	"errors"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"strconv"

	"github.com/cep21/circuit/v3"
	"github.com/spothero/tools/http/writer"
	sql "github.com/spothero/tools/sql/middleware"
)

// setSpanTags sets default HTTP span tags
func setSpanTags(r *http.Request, span trace.Span) trace.Span {
	var attrs []attribute.KeyValue
	attrs = append(attrs, attribute.String("http.method", r.Method))
	attrs = append(attrs, attribute.String("http.url", r.URL.String()))
	attrs = append(attrs, attribute.String("http.path", writer.FetchRoutePathTemplate(r)))
	attrs = append(attrs, attribute.String("http.user_agent", r.UserAgent()))
	if contentLengthStr := r.Header.Get("Content-Length"); len(contentLengthStr) > 0 {
		if contentLength, err := strconv.Atoi(contentLengthStr); err == nil {
			attrs = append(attrs, attribute.Int("http.content_length", contentLength))
		}
	}
	span.SetAttributes(attrs...)
	return span
}

// HTTPServerMiddleware extracts the OpenTracing context on all incoming HTTP requests, if present. if
// no trace ID is present in the headers, a trace is initiated.
//
// The following tags are placed on all incoming HTTP requests:
// * http.method
// * http.url
//
// Outbound responses will be tagged with the following tags, if applicable:
// * http.status_code
// * error (if the status code is >= 500)
//
// The returned HTTP Request includes the wrapped OpenTracing Span Context.
// Note that this middleware must be attached after writer.StatusRecorderMiddleware
// for HTTP response span tagging to function.
func HTTPServerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		//wireContext := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		//trace.SpanFromContext(r.Context()).SpanContext()
		//span, spanCtx := opentracing.StartSpanFromContext(r.Context(), writer.FetchRoutePathTemplate(r), ext.RPCServerOption(wireContext))
		links := []trace.Link{{SpanContext: trace.SpanFromContext(r.Context()).SpanContext()}}
		trace.WithLinks(links...)
		span, spanCtx := StartSpanFromContext(r.Context(), writer.FetchRoutePathTemplate(r), trace.WithLinks(links...))
		span = setSpanTags(r, span)
		defer func() {
			if statusRecorder, ok := w.(*writer.StatusRecorder); ok {
				span.SetAttributes(attribute.String("http.status_code", strconv.Itoa(statusRecorder.StatusCode)))
				// 5XX Errors are our fault -- note that this span belongs to an errored request
				if statusRecorder.StatusCode >= http.StatusInternalServerError {
					span.SetAttributes(attribute.Bool("error", true))
				}
			}
			span.End()
		}()
		next.ServeHTTP(w, r.WithContext(EmbedCorrelationID(spanCtx)))
	})
}

// RoundTripper provides a proxied HTTP RoundTripper which traces client HTTP request details
type RoundTripper struct {
	RoundTripper http.RoundTripper
}

// RoundTrip completes HTTP roundtrips while tracing HTTP request details
func (rt RoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	// Ensure the inner RoundTripper was set on the RoundTripper
	if rt.RoundTripper == nil {
		panic("no roundtripper provided to tracing round tripper")
	}

	operationName := fmt.Sprintf("%s %s", r.Method, r.URL.String())
	span, spanCtx := StartSpanFromContext(r.Context(), operationName)
	span = setSpanTags(r, span)

	resp, err := rt.RoundTripper.RoundTrip(r.WithContext(EmbedCorrelationID(spanCtx)))
	if err != nil {
		var circuitError circuit.Error
		if errors.As(err, &circuitError) {
			span.SetAttributes(attribute.String("circuit-breaker", circuitError.Error()))
		}
		span.SetAttributes(attribute.Bool("error", true))
		span.End()
		return nil, fmt.Errorf("http client request failed: %w", err)
	}

	span.SetAttributes(attribute.String("http.status_code", resp.Status))
	if resp.StatusCode >= http.StatusBadRequest {
		span.SetAttributes(attribute.Bool("error", true))
	}
	span.End()
	return resp, err
}

// GetCorrelationID returns the correlation ID associated with the given
// Context. This function only produces meaningful results for Contexts
// associated with gRPC or HTTP Requests which have passed through
// their associated tracing middleware.
func GetCorrelationID(ctx context.Context) string {
	if maybeCorrelationID, ok := ctx.Value(CorrelationIDCtxKey).(string); ok {
		return maybeCorrelationID
	} else {
		return ""
	}
}

// SQLMiddleware traces requests made against SQL databases.
//
// Span names always start with "db". If a queryName is provided (highly recommended), the span
// name will include the queryname in the format "db_<queryName>"
//
// The following tags are placed on all SQL traces:
// * component - Always set to "tracing"
// * db.type - Always set to "sql"
// * db.statement - Always set to the query statement
// * error - Set to true only if an error was encountered with the query
func SQLMiddleware(ctx context.Context, queryName, query string, args ...interface{}) (context.Context, sql.MiddlewareEnd, error) {
	spanName := "db"
	if queryName != "" {
		spanName = fmt.Sprintf("%s_%s", spanName, queryName)
	}
	span, spanCtx := StartSpanFromContext(ctx, spanName)
	var attrs []attribute.KeyValue
	attrs = append(attrs, attribute.String("component", "tracing"))
	attrs = append(attrs, attribute.String("db.type", "sql"))
	attrs = append(attrs, attribute.String("db.statement", query))
	//attrs = append(attrs, attribute.StringSlice("db.statement.arguments", args))
	span.SetAttributes(attrs...)
	mwEnd := func(ctx context.Context, queryName, query string, queryErr error, args ...interface{}) (context.Context, error) {
		defer span.End()
		if queryErr != nil {
			span.SetAttributes(attribute.Bool("error", true))
		}
		return ctx, nil
	}
	return EmbedCorrelationID(spanCtx), mwEnd, nil
}
