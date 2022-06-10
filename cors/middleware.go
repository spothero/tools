// Copyright 2022 SpotHero
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

package cors

import (
	"net/http"

	"github.com/gorilla/mux"
)

// GetHTTPServerMiddleware returns middleware that adds cors header options to
// the response and short-circuits further request processing if the request
// method is OPTIONS.
func (c Config) GetHTTPServerMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", c.AllowedOrigins)
			w.Header().Set("Access-Control-Allow-Methods", c.AllowedMethods)
			w.Header().Set("Access-Control-Allow-Headers", c.AllowedHeaders)
			if r.Method == http.MethodOptions {
				// Downstream handlers do not (yet) have any specific logic for
				// the OPTIONS method so we can return early.
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
