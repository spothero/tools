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

// HTTPMiddleware logs a series of standard attributes for every HTTP request.
//
//  On inbound request received these attributes include:
// * The remote address of the client
// * The HTTP Method utilized
//
// On outbound response return these attributes include all of the above as well as:
// * HTTP response code
func HTTPMiddleware(sr *writer.StatusRecorder, r *http.Request) (func(), *http.Request) {
	method := zap.String("http_method", r.Method)
	path := zap.String("path", writer.FetchRoutePathTemplate(r))
	query := zap.String("query_string", r.URL.Query().Encode())
	Get(r.Context()).Info("request received", method, path, query)
	Get(r.Context()).Debug("request headers", zap.Reflect("Headers", r.Header))
	return func() {
		Get(r.Context()).Info(
			"returning response",
			zap.Int("response_code", sr.StatusCode))
	}, r
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
