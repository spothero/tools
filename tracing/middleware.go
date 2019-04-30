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
	"net/http"
	"strconv"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/spothero/tools/http/writer"
	"github.com/spothero/tools/log"
	"github.com/uber/jaeger-client-go"
	"go.uber.org/zap"
)

// HTTPMiddleware extracts the OpenTracing context on all incoming HTTP requests, if present. if
// no trace ID is present in the headers, a trace is initiated.
//
// The following tags are placed on all incoming HTTP requests:
// * http.method
// * http.hostname
// * http.port
// * http.remote_address
//
// Outbound responses will be tagged with the following tags, if applicable:
// * http.status_code
// * error (if the status code is >= 500)
//
// The returned HTTP Request includes the wrapped OpenTracing Span Context.
func HTTPMiddleware(sr *writer.StatusRecorder, r *http.Request) (func(), *http.Request) {
	wireContext, err := opentracing.GlobalTracer().Extract(
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(r.Header))
	if err != nil {
		log.Get(r.Context()).Debug("failed to extract opentracing context on an incoming http request")
	}
	span, spanCtx := opentracing.StartSpanFromContext(r.Context(), writer.FetchRoutePathTemplate(r), ext.RPCServerOption(wireContext))
	span = span.SetTag("http.method", r.Method)
	span = span.SetTag("http.url", r.URL.String())

	// While this removes the veneer of OpenTracing abstraction, the current specification does not
	// provide a method of accessing Trace ID directly. Until OpenTracing 2.0 is released with
	// support for abstract access for Trace ID we will coerce the type to the underlying tracer.
	// See: https://github.com/opentracing/specification/issues/123
	if sc, ok := span.Context().(jaeger.SpanContext); ok {
		// Embed the Trace ID in the logging context for all future requests
		spanCtx = log.NewContext(spanCtx, zap.String("trace_id", sc.TraceID().String()))
	}
	return func() {
		span.SetTag("http.status_code", strconv.Itoa(sr.StatusCode))
		// 5XX Errors are our fault -- note that this span belongs to an errored request
		if sr.StatusCode >= http.StatusInternalServerError {
			span.SetTag("error", true)
		}
		span.Finish()
	}, r.WithContext(spanCtx)
}
