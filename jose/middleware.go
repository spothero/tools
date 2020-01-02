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
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/spothero/tools/log"
)

// authHeader defines the name of the header containing the JWT authorization data
const authHeader = "Authorization"

// bearerPrefix defines the standard expected form for OIDC JWT Tokens in Authorization headers.
// Eg `Authorization: Bearer <JWT>`
const bearerPrefix = "Bearer "

const (
	authHeaderNotFound   = "no authorization header found"
	bearerPrefixNotFound = "authorization header did not include bearer prefix"
	invalidBearerToken   = "bearer token is invalid"
	noBearerToken        = "no authorization bearer token found"
)

// GetHTTPMiddleware returns an HTTP middleware function which extracts the Authorization header,
// if present, on all incoming HTTP requests. If an Authorization header is found, this middleware
// attempts to parse and validate that value as a JWT with the configured Credential types for
// the given JOSE provider.
func GetHTTPMiddleware(jh JOSEHandler, authRequired bool) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := log.Get(r.Context())
			authHeader := r.Header.Get(authHeader)
			var parseErrMsg string
			if len(authHeader) == 0 {
				logger.Debug(authHeaderNotFound)
				parseErrMsg = authHeaderNotFound
			}

			if len(parseErrMsg) == 0 && !strings.HasPrefix(authHeader, bearerPrefix) {
				logger.Debug(bearerPrefixNotFound)
				parseErrMsg = bearerPrefixNotFound
			}

			var claims []Claim
			bearerToken := ""
			if len(parseErrMsg) == 0 {
				claims = jh.GetClaims()
				bearerToken = strings.TrimPrefix(authHeader, bearerPrefix)
				err := jh.ParseValidateJWT(bearerToken, claims...)
				if err != nil {
					logger.Debug(err.Error())
					parseErrMsg = invalidBearerToken
				}
			}
			if len(parseErrMsg) == 0 {
				// Populate each claim on the context, if any
				for _, claim := range claims {
					r = r.WithContext(claim.NewContext(r.Context()))
				}
				// Set the bearer token on the context so it can be passed to any downstream services
				r = r.WithContext(context.WithValue(r.Context(), JWTClaimKey, bearerToken))
			}

			if len(parseErrMsg) != 0 {
				logger.Debug(parseErrMsg)
				if authRequired {
					httpStatus := http.StatusForbidden
					if parseErrMsg == bearerPrefixNotFound || parseErrMsg == authHeaderNotFound {
						w.Header().Set("WWW-Authenticate", "Bearer")
						httpStatus = http.StatusUnauthorized
					}
					http.Error(w, parseErrMsg, httpStatus)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// HTTPClientMiddleware is middleware for use in HTTP Clients which propagates the Authorization
// headers
func HTTPClientMiddleware(r *http.Request) (*http.Request, func(*http.Response) error, error) {
	if jwtData, ok := r.Context().Value(JWTClaimKey).(string); ok {
		r.Header.Set(authHeader, fmt.Sprintf("%s%s", bearerPrefix, jwtData))
	}
	return r, func(resp *http.Response) error { return nil }, nil
}
