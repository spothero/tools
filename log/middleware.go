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
		requestLogger := Get(r.Context())
		logger := requestLogger.Named("http")
		method := zap.String("http_method", r.Method)
		path := zap.String("path", writer.FetchRoutePathTemplate(r))
		query := zap.String("query_string", r.URL.Query().Encode())
		logger.Info("request received", method, path, query)
		logger.Debug("request headers", zap.Reflect("Headers", r.Header))
		defer func() {
			var responseCodeField zap.Field
			if statusRecorder, ok := w.(*writer.StatusRecorder); ok {
				responseCodeField = zap.Int("response_code", statusRecorder.StatusCode)
			} else {
				responseCodeField = zap.Skip()
			}
			logger.Info("returning response", responseCodeField)
		}()
		// ensure that a logger is present for downstream handlers in the request context
		next.ServeHTTP(w, r.WithContext(NewContext(r.Context(), requestLogger)))
	})
}

// SQLMiddleware debug logs requests made against SQL databases.
func SQLMiddleware(ctx context.Context, queryName, query string, args ...interface{}) (context.Context, sqlMiddleware.MiddlewareEnd, error) {
	Get(ctx).Debug(
		"attempting sql query",
		zap.String("query_name", queryName),
		zap.String("query", query),
	)
	if queryName == "" {
		Get(ctx).Warn(
			"unnamed sql query provided",
			zap.String("query_name", queryName),
			zap.String("query", query),
		)
	}
	mwEnd := func(ctx context.Context, queryName, query string, queryErr error, args ...interface{}) (context.Context, error) {
		if queryErr != nil {
			Get(ctx).Error(
				"failed sql query",
				zap.String("query_name", queryName),
				zap.String("query", query),
				zap.Error(queryErr),
			)
		} else {
			Get(ctx).Debug(
				"completed sql query",
				zap.String("query_name", queryName),
				zap.String("query", query),
			)
		}
		return ctx, nil
	}
	return ctx, mwEnd, nil
}
