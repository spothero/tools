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

package jose

import (
	"net/http"
	"strings"

	"github.com/spothero/tools/http/writer"
	"github.com/spothero/tools/log"
	"go.uber.org/zap"
)

// authHeader defines the name of the header containing the JWT authorization data
const authHeader = "Authorization"

// bearerPrefix defines the standard expected form for OIDC JWT Tokens in Authorization headers.
// Eg `Authorization: Bearer <JWT>`
const bearerPrefix string = "Bearer "

// GetHTTPMiddleware returns an HTTP middleware function which extracts the Authorization header,
// if present, on all incoming HTTP requests. If an Authorization header is found, this middleware
// attempts to parse and validate that value as a JWT with  the configured Credential types for
// the given JOSE provider.
func GetHTTPMiddleware(jh JOSEHandler) func(*writer.StatusRecorder, *http.Request) (func(), *http.Request) {
	return func(sr *writer.StatusRecorder, r *http.Request) (func(), *http.Request) {
		logger := log.Get(r.Context())
		authHeader := r.Header.Get(authHeader)
		if len(authHeader) == 0 {
			logger.Debug("no authorization header found")
			return func() {}, r
		}

		if !strings.HasPrefix(authHeader, bearerPrefix) {
			logger.Debug("authorization header did not include bearer prefix")
			return func() {}, r
		}

		claims := jh.GetClaims()
		bearerToken := strings.TrimPrefix(authHeader, bearerPrefix)
		err := jh.ParseValidateJWT(bearerToken, claims)
		if err != nil {
			logger.Debug(
				"failed to parse and validate incoming jwt",
				zap.Error(err),
			)
			return func() {}, r
		}

		// Populate each claim on the context, if any
		for _, claim := range claims {
			r = r.WithContext(claim.WithContext(r.Context()))
		}
		return func() {}, r
	}
}
