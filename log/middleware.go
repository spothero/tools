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
	"net/http"

	"github.com/spothero/tools/http/utils"
	"go.uber.org/zap"
)

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
func LoggingMiddleware(sr *utils.StatusRecorder, r *http.Request) (func(), *http.Request) {
	remoteAddress := zap.String("remote_address", r.RemoteAddr)
	method := zap.String("http_method", r.Method)
	path := zap.String("path", utils.FetchRoutePathTemplate(r))
	query := zap.String("query_string", r.URL.Query().Encode())
	hostname := zap.String("hostname", r.URL.Hostname())
	port := zap.String("port", r.URL.Port())
	Get(r.Context()).Info("Request Received", remoteAddress, method, path, query, hostname, port)
	Get(r.Context()).Debug("Request Headers", zap.Reflect("Headers", r.Header))
	return func() {
		Get(r.Context()).Info(
			"Returning Response",
			remoteAddress, hostname, port, zap.Int("response_code", sr.StatusCode))
	}, r
}