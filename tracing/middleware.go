// Copyright 2020 SpotHero
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
	"strconv"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/spothero/tools/http/writer"
	"github.com/spothero/tools/log"
	sql "github.com/spothero/tools/sql/middleware"
)

// setSpanTags sets default HTTP span tags
func setSpanTags(r *http.Request, span opentracing.Span) opentracing.Span {
	span = span.SetTag("http.method", r.Method)
	span = span.SetTag("http.url", r.URL.String())
	span = span.SetTag("http.path", writer.FetchRoutePathTemplate(r))
	span = span.SetTag("http.user_agent", r.UserAgent())
	if contentLengthStr := r.Header.Get("Content-Length"); len(contentLengthStr) > 0 {
		if contentLength, err := strconv.Atoi(contentLengthStr); err == nil {
			span = span.SetTag("http.content_length", contentLength)
		}
	}
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
		logger := log.Get(r.Context())
		wireContext, err := opentracing.GlobalTracer().Extract(
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(r.Header))
		if err != nil {
			logger.Debug("failed to extract opentracing context on an incoming http request")
		}
		span, spanCtx := opentracing.StartSpanFromContext(r.Context(), writer.FetchRoutePathTemplate(r), ext.RPCServerOption(wireContext))
		span = setSpanTags(r, span)
		defer func() {
			if statusRecorder, ok := w.(*writer.StatusRecorder); ok {
				span = span.SetTag("http.status_code", strconv.Itoa(statusRecorder.StatusCode))
				// 5XX Errors are our fault -- note that this span belongs to an errored request
				if statusRecorder.StatusCode >= http.StatusInternalServerError {
					span = span.SetTag("error", true)
				}
			}
			span.Finish()
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
	span, spanCtx := opentracing.StartSpanFromContext(r.Context(), operationName)
	span = setSpanTags(r, span)

	resp, err := rt.RoundTripper.RoundTrip(r.WithContext(EmbedCorrelationID(spanCtx)))
	if err != nil {
		span.Finish()
		return nil, fmt.Errorf("http client request failed: %w", err)
	}

	span = span.SetTag("http.status_code", resp.Status)
	if resp.StatusCode >= http.StatusBadRequest {
		span = span.SetTag("error", true)
	}
	span.Finish()
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
	span, spanCtx := opentracing.StartSpanFromContext(ctx, spanName)
	span = span.
		SetTag("component", "tracing").
		SetTag("db.type", "sql").
		SetTag("db.statement", query).
		SetTag("db.statement.arguments", args)
	mwEnd := func(ctx context.Context, queryName, query string, queryErr error, args ...interface{}) (context.Context, error) {
		defer span.Finish()
		if queryErr != nil {
			span = span.SetTag("error", true)
		}
		return ctx, nil
	}
	return EmbedCorrelationID(spanCtx), mwEnd, nil
}
