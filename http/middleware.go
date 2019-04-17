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
	"net/http"
	"strconv"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/spothero/tools/log"
	jaeger "github.com/uber/jaeger-client-go"
	"go.uber.org/zap"
)

// StatusRecorder wraps the http ResponseWriter, allowing additional instrumentation and metrics
// capture before the response is returned to the client.
type StatusRecorder struct {
	http.ResponseWriter
	StatusCode int
}

// WriteHeader implements the http ResponseWriter WriteHeader interface. This function acts as a
// middleware which captures the StatusCode on the StatusRecorder and then delegates the actual
// work of writing the header to the underlying http ResponseWriter.
func (sr *StatusRecorder) WriteHeader(code int) {
	sr.StatusCode = code
	sr.ResponseWriter.WriteHeader(code)
}

// MiddlewareFunc defines a middleware function used in processing HTTP Requests. Request
// preprocessing may be specified in the body of the middleware function call. If post-processing
// is required, please use the returned deferable func() to encapsulate that logic.
type MiddlewareFunc func(*StatusRecorder, *http.Request) (func(), *http.Request)

// Middleware defines a collection of middleware functions.
type Middleware []MiddlewareFunc

// handler is meant to be used as middleware for every request on a given handler. Common usages of
// middleware functions:
//
// * Start an opentracing span, place it in http.Request context, and
//   close the span when the request completes
// * Capture any unhandled errors and send them to Sentry
// * Capture metrics to Prometheus for the duration of the HTTP request
//
// Middleware is an effective way to add functionality to every request traversing the server --
// both before and after processing is completed.
//
// Its worth noting that this handler can be used to wrap individual and grouped routes. This is an
// effective strategy for the following kinds of example strategies:
//
// * Auth enforcement
// * Checking for specific headers: version, content-type, etc
// * Rate limiting
func (m Middleware) handler(next http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Default to http.StatusOK which is the golang default if the status is not set.
		wrappedWriter := &StatusRecorder{w, http.StatusOK}
		for _, mw := range m {
			var deferable func()
			deferable, r = mw(wrappedWriter, r)
			defer deferable()
		}
		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs a series of standard attributes for every HTTP request.
//
//  On inbound request received these attributes include:
// * The remote address of the client
// * The HTTP Method utilized
// * The hostname specified on this request
// * The port specified on this request
//
// On outbound response return these attributes include all of the above as well as:
// * HTTP response code
func LoggingMiddleware(sr *StatusRecorder, r *http.Request) (func(), *http.Request) {
	remoteAddress := zap.String("remote_address", r.RemoteAddr)
	method := zap.String("method", r.Method)
	hostname := zap.String("hostname", r.URL.Hostname())
	port := zap.String("port", r.URL.Port())
	log.Get(r.Context()).Info("Request Received", remoteAddress, method, hostname, port)
	log.Get(r.Context()).Debug("Request Headers", zap.Reflect("Headers", r.Header))
	return func() {
		log.Get(r.Context()).Info(
			"Returning Response",
			remoteAddress, method, hostname, port, zap.Int("response_code", sr.StatusCode))
	}, r
}

// TracingMiddleware extracts the OpenTracing context on all incoming HTTP requests, if present. if
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
func TracingMiddleware(sr *StatusRecorder, r *http.Request) (func(), *http.Request) {
	wireContext, err := opentracing.GlobalTracer().Extract(
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(r.Header))
	if err != nil {
		log.Get(r.Context()).Debug("Failed to extract opentracing context on an incoming HTTP request.")
	}
	span, spanCtx := opentracing.StartSpanFromContext(r.Context(), r.URL.Path, ext.RPCServerOption(wireContext))
	span = span.SetTag("http.method", r.Method)
	span = span.SetTag("http.hostname", r.URL.Hostname())
	span = span.SetTag("http.port", r.URL.Port())
	span = span.SetTag("http.remote_address", r.RemoteAddr)

	// While this removes the veneer of OpenTracing abstraction, the current specification does not
	// provide a method of accessing Trace ID directly. Until OpenTracing 2.0 is released with
	// support for abstract access for Trace ID we will coerce the type to the  underlying tracer.
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
