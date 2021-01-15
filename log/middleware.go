// Copyright 2021 SpotHero
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

package log

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/cep21/circuit/v3"
	"github.com/spothero/tools/http/writer"
	sqlMiddleware "github.com/spothero/tools/sql/middleware"
	"go.uber.org/zap"
)

// getFields returns appropriate zap logger fields given the HTTP Request
func getFields(r *http.Request) []zap.Field {
	fields := []zap.Field{
		zap.String("http.method", r.Method),
		zap.String("http.url", r.URL.String()),
		zap.String("http.path", writer.FetchRoutePathTemplate(r)),
		zap.String("http.user_agent", r.UserAgent()),
	}
	if contentLengthStr := r.Header.Get("Content-Length"); len(contentLengthStr) > 0 {
		if contentLength, err := strconv.Atoi(contentLengthStr); err == nil {
			fields = append(fields, zap.Int("http.content_length", contentLength))
		}
	}
	return fields
}

// HTTPServerMiddleware logs a series of standard attributes for every HTTP request and attaches
// a logger onto the request context.
//
//  On inbound request received these attributes include:
// * The remote address of the client
// * The HTTP Method utilized
//
// On outbound response return these attributes include all of the above as well as:
// * HTTP response code
// Note that this middleware must be attached after writer.StatusRecorderMiddleware
// for HTTP response code logging to function and after tracing.HTTPServerMiddleware for trace ids
// to show up in logs.
func HTTPServerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		logger := Get(r.Context()).Named("http").With(getFields(r)...)
		logger.Debug("http request received")
		defer func() {
			var responseCodeField zap.Field
			if statusRecorder, ok := w.(*writer.StatusRecorder); ok {
				responseCodeField = zap.Int("http.status_code", statusRecorder.StatusCode)
			} else {
				responseCodeField = zap.Skip()
			}
			logger = logger.With(responseCodeField, zap.Duration("http.duration", time.Since(startTime)))
			logger.Info("http response returned")
			r = r.WithContext(NewContext(r.Context(), logger))
		}()
		// ensure that a logger is present for downstream handlers in the request context
		next.ServeHTTP(w, r.WithContext(NewContext(r.Context(), logger)))
	})
}

// RoundTripper provides a proxied HTTP RoundTripper which logs client HTTP request details
type RoundTripper struct {
	RoundTripper http.RoundTripper
}

// RoundTrip completes HTTP roundtrips while logging HTTP request details
func (rt RoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	// Ensure the inner RoundTripper was set on the RoundTripper
	if rt.RoundTripper == nil {
		panic("no roundtripper provided to log round tripper")
	}

	startTime := time.Now()
	logger := Get(r.Context()).Named("http").With(getFields(r)...)
	logger.Debug("http request started")
	resp, err := rt.RoundTripper.RoundTrip(r)
	if err != nil {
		var circuitError circuit.Error
		if errors.As(err, &circuitError) {
			logger.Warn(
				"circuit breaker error on http request",
				zap.String("host", r.URL.Host),
				zap.Bool("circuit_opened", circuitError.CircuitOpen()),
				zap.Bool("concurrency_limit_reached", circuitError.ConcurrencyLimitReached()),
				zap.String("reason", circuitError.Error()),
				zap.Error(err),
			)
		}
		return nil, fmt.Errorf("http client request failed: %w", err)
	}

	logger = logger.With(zap.Int("http.status_code", resp.StatusCode), zap.Duration("http.duration", time.Since(startTime)))
	logger.Info("http request completed")
	return resp, err
}

// SQLMiddleware debug logs requests made against SQL databases.
func SQLMiddleware(ctx context.Context, queryName, query string, args ...interface{}) (context.Context, sqlMiddleware.MiddlewareEnd, error) {
	logger = Get(ctx).With(zap.String("query", query))
	if queryName != "" {
		logger = logger.With(zap.String("query_name", queryName))
	}
	logger.Debug("attempting sql query")
	mwEnd := func(ctx context.Context, queryName, query string, queryErr error, args ...interface{}) (context.Context, error) {
		if queryErr != nil {
			logger.With(zap.Error(queryErr)).Error("failed sql query")
		} else {
			Get(ctx).Debug("completed sql query")
		}
		return ctx, nil
	}
	return ctx, mwEnd, nil
}
