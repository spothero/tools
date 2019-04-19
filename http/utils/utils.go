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

package utils

import (
	"net/http"

	"github.com/gorilla/mux"
)

// StatusRecorder wraps the http ResponseWriter, allowing additional instrumentation and metrics
// capture before the response is returned to the client.
type StatusRecorder struct {
	http.ResponseWriter
	StatusCode int
}

// WriteHeader implements the http ResponseWriter WriteHeader interface. This function acts as a
// middleware which captures the StatusCode on the StatusRecorder and then delegates the actual
// work of writing the header to the underlying http ResponseWriter.
func (sr *StatusRecorder) WriteHeader(code int) {
	sr.StatusCode = code
	sr.ResponseWriter.WriteHeader(code)
}

// FetchRoutePath extracts the path template from a given request, or emptry string if none could
// be found.
func FetchRoutePathTemplate(r *http.Request) string {
	routePath := ""
	if route := mux.CurrentRoute(r); route != nil {
		routePath, _ = route.GetPathTemplate()
	}
	return routePath
}
