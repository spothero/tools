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
func (sr StatusRecorder) WriteHeader(code int) {
	sr.StatusCode = code
	sr.ResponseWriter.WriteHeader(code)
}

// MiddlewareFunc defines a middleware function used in processing HTTP Requests. Request
// preprocessing may be specified in the body of the middleware function call. If post-processing
// is required, please use the returned deferable func() to encapsulate that logic.
type MiddlewareFunc func(*StatusRecorder, *http.Request) (func(), *http.Request)

// middleware defines a collection of middleware functions. All functions must return a deferable
// call, whether or not the callback performs any work in addition to the http.Request.
type middleware struct {
	functions []MiddlewareFunc
}

// handler is meant to be used as middleware for every request. Common usages of
// middleware functions:
//
// * Start an opentracing span, place it in http.Request context, and
//   close the span when the request completes
// * Capture any unhandled errors and send them to Sentry
// * Capture metrics to Prometheus for the duration of the HTTP request
//
// Middleware is an effective way to add functionality to every request traversing the server --
// both before and after processing is completed.
func (m middleware) handler(next http.Handler, serverName string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Default to http.StatusOK which is the golang default if the status is not set.
		wrappedWriter := &StatusRecorder{w, http.StatusOK}
		for _, mw := range m.functions {
			deferable, r := mw(wrappedWriter, r)
			defer deferable()
		}
	})
}
