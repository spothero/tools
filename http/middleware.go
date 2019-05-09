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

	"github.com/spothero/tools/http/writer"
)

// MiddlewareFunc defines a middleware function used in processing HTTP Requests. Request
// preprocessing may be specified in the body of the middleware function call. If post-processing
// is required, please use the returned deferable func() to encapsulate that logic.
type MiddlewareFunc func(*writer.StatusRecorder, *http.Request) (func(), *http.Request)

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
func (m Middleware) handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Default to http.StatusOK which is the golang default if the status is not set.
		wrappedWriter := &writer.StatusRecorder{ResponseWriter: w, StatusCode: http.StatusOK}
		for _, mw := range m {
			var deferable func()
			deferable, r = mw(wrappedWriter, r)
			defer deferable()
		}
		next.ServeHTTP(wrappedWriter, r)
	})
}
