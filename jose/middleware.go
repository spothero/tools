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

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spothero/tools/http/writer"
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
				logger.Info("failed to parse and validate claims", zap.Error(err))
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

// authMetrics is a bundle of prometheus HTTP metrics recorders
type authMetrics struct {
	authSuccessCounter *prometheus.CounterVec
	authFailureCounter *prometheus.CounterVec
}

// newAuthMetrics creates and returns a metrics bundle. The user may optionally
// specify an existing Prometheus Registry. If no Registry is provided, the global Prometheus
// Registry is used. Finally, if mustRegister is true, and a registration error is encountered,
// the application will panic.
func newAuthMetrics(registry prometheus.Registerer) authMetrics {
	labels := []string{"path", "method"}
	authSuccessCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_authentication_success_total",
			Help: "Total number of HTTP requests in which authentication succeeded",
		},
		labels,
	)
	authFailureCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_authentication_failure_total",
			Help: "Total number of HTTP requests in which authentication failed",
		},
		labels,
	)
	// If the user hasnt provided a Prometheus Registry, use the global Registry
	if registry == nil {
		registry = prometheus.DefaultRegisterer
	}

	toRegister := []prometheus.Collector{
		authSuccessCounter,
		authFailureCounter,
	}
	for _, collector := range toRegister {
		// intentionally ignore error
		_ = registry.Register(collector)
	}
	return authMetrics{
		authSuccessCounter: authSuccessCounter,
		authFailureCounter: authFailureCounter,
	}
}

// AuthenticationMiddleware enforces authentication for all routes associated
// with a subrouter.
func AuthenticationMiddleware(next http.Handler) http.Handler {
	return EnforceAuthentication(next.ServeHTTP)
}

// EnforceAuthentication enforces authentication for a single HTTP handler.
func EnforceAuthentication(next http.HandlerFunc) http.HandlerFunc {
	var defaultRegistry prometheus.Registerer = nil
	metrics := newAuthMetrics(defaultRegistry)

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := log.Get(ctx)
		isAuthenticated := ctx.Value(JWTClaimKey) != nil
		labels := prometheus.Labels{
			"path":   writer.FetchRoutePathTemplate(r),
			"method": r.Method,
		}

		if isAuthenticated {
			logger.Debug("authentication successfully enforced on request")
			metrics.authSuccessCounter.With(labels).Inc()
			next(w, r)
		} else {
			logger.Debug("authentication enforcement failed on request")
			metrics.authFailureCounter.With(labels).Inc()
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, bearerPrefixNotFound, http.StatusUnauthorized)
		}
	}
}
