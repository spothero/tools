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

	"github.com/spothero/tools/http/writer"
	"github.com/spothero/tools/log"
)

// HTTPMiddleware extracts the Authorization header, if present, on all incoming HTTP requests.
// If an Authorization header is found, it is attempted to be parsed as a JWT with the configured
// Credential types for the given JOSE provider.
func HTTPMiddleware(sr *writer.StatusRecorder, r *http.Request) (func(), *http.Request) {
	logger := log.Get(r.Context())
	// TODO: Update these return values!
	return func() {
		logger.Info("jose middleware unimplemented")
	}, r
}
