// Copyright 2020 SpotHero
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
	"go.uber.org/zap"
)

// authHeader defines the name of the header containing the JWT authorization data
const authHeader = "Authorization"

// bearerPrefix defines the standard expected form for OIDC JWT Tokens in Authorization headers.
// Eg `Authorization: Bearer <JWT>`
const bearerPrefix = "Bearer "

const (
	bearerPrefixNotFound = "authorization header did not include bearer prefix"
	invalidBearerToken   = "bearer token is invalid"
)

// GetHTTPServerMiddleware returns an HTTP middleware function which extracts the Authorization
// header, if present, on all incoming HTTP requests. If an Authorization header is found, this
// middleware attempts to parse and validate that value as a JWT with the configured Credential
// types for the given JOSE provider.
func GetHTTPServerMiddleware(jh JOSEHandler) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := log.Get(r.Context())
			authHeader := r.Header.Get(authHeader)
			if len(authHeader) == 0 {
				logger.Debug("no bearer token found")
				next.ServeHTTP(w, r)
				return
			}

			if !strings.HasPrefix(authHeader, bearerPrefix) {
				logger.Debug(bearerPrefixNotFound)
				w.Header().Set("WWW-Authenticate", "Bearer")
				http.Error(w, bearerPrefixNotFound, http.StatusUnauthorized)
				return
			}

			claims := jh.GetClaims()
			bearerToken := strings.TrimPrefix(authHeader, bearerPrefix)
			err := jh.ParseValidateJWT(bearerToken, claims...)
			if err != nil {
				logger.Debug("invalid bearer token", zap.Error(err))
				http.Error(w, invalidBearerToken, http.StatusForbidden)
				return
			}

			// Populate each claim on the context, if any
			for _, claim := range claims {
				r = r.WithContext(claim.NewContext(r.Context()))
			}
			// Set the bearer token on the context so it can be passed to any downstream services
			r = r.WithContext(context.WithValue(r.Context(), JWTClaimKey, bearerToken))

			next.ServeHTTP(w, r)
		})
	}
}

// RoundTripper implements an http.RoundTripper which passes along any auth headers automatically
type RoundTripper struct {
	RoundTripper http.RoundTripper
}

// RoundTrip is intended for use in HTTP Clients and it propagates the Authorization
// headers on outgoing HTTP calls automatically
func (rt RoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	// Ensure the RoundTripper was set on the MiddlewareRoundTripper
	if rt.RoundTripper == nil {
		panic("no roundtripper provided to auth round tripper")
	}

	if jwtData, ok := r.Context().Value(JWTClaimKey).(string); ok {
		r.Header.Set(authHeader, fmt.Sprintf("%s%s", bearerPrefix, jwtData))
	}
	return rt.RoundTripper.RoundTrip(r)
}
