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

package log

import (
	"context"
	"net/http"
	"time"

	"github.com/spothero/tools/http/writer"
	sqlMiddleware "github.com/spothero/tools/sql/middleware"
	"go.uber.org/zap"
)

// HTTPMiddleware logs a series of standard attributes for every HTTP request and attaches
// a logger onto the request context.
//
//  On inbound request received these attributes include:
// * The remote address of the client
// * The HTTP Method utilized
//
// On outbound response return these attributes include all of the above as well as:
// * HTTP response code
// Note that this middleware must be attached after writer.StatusRecorderMiddleware
// for HTTP response code logging to function and after tracing.HTTPMiddleware for trace ids
// to show up in logs.
func HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		requestLogger := Get(r.Context())
		logger := requestLogger.Named("http")
		method := zap.String("http.method", r.Method)
		url := zap.String("http.url", r.URL.String())
		path := zap.String("http.path", writer.FetchRoutePathTemplate(r))
		userAgent := zap.String("http.user_agent", r.UserAgent())
		logger.Debug("http request received", method, url, path, userAgent)
		defer func() {
			var responseCodeField zap.Field
			if statusRecorder, ok := w.(*writer.StatusRecorder); ok {
				responseCodeField = zap.Int("http.status_code", statusRecorder.StatusCode)
			} else {
				responseCodeField = zap.Skip()
			}
			logger.Info(
				"http response returned",
				responseCodeField,
				method,
				url,
				path,
				userAgent,
				zap.Duration("http.duration", time.Since(startTime)),
			)
		}()
		// ensure that a logger is present for downstream handlers in the request context
		next.ServeHTTP(w, r.WithContext(NewContext(r.Context(), requestLogger)))
	})
}

// HTTPClientMiddleware is middleware for use in HTTP Clients which logs outbound requests
func HTTPClientMiddleware(r *http.Request) (*http.Request, func(*http.Response) error, error) {
	startTime := time.Now()
	requestLogger := Get(r.Context())
	logger := requestLogger.Named("http")
	method := zap.String("http.method", r.Method)
	url := zap.String("http.url", r.URL.String())
	path := zap.String("http.path", writer.FetchRoutePathTemplate(r))
	userAgent := zap.String("http.user_agent", r.UserAgent())
	logger.Debug("http request started", method, url, path, userAgent)
	return r, func(resp *http.Response) error {
		logger.Info(
			"http request completed",
			zap.Int("http.status_code", resp.StatusCode),
			method,
			url,
			path,
			userAgent,
			zap.Duration("http.duration", time.Since(startTime)),
		)
		return nil
	}, nil
}

// SQLMiddleware debug logs requests made against SQL databases.
func SQLMiddleware(ctx context.Context, queryName, query string, args ...interface{}) (context.Context, sqlMiddleware.MiddlewareEnd, error) {
	var fields []zap.Field
	if queryName != "" {
		fields = []zap.Field{zap.String("query", query), zap.String("query_name", queryName)}
	} else {
		fields = []zap.Field{zap.String("query", query)}
	}
	Get(ctx).Debug("attempting sql query", fields...)
	mwEnd := func(ctx context.Context, queryName, query string, queryErr error, args ...interface{}) (context.Context, error) {
		if queryErr != nil {
			fields = append(fields, zap.Error(queryErr))
			Get(ctx).Error("failed sql query", fields...)
		} else {
			Get(ctx).Debug("completed sql query", fields...)
		}
		return ctx, nil
	}
	return ctx, mwEnd, nil
}
