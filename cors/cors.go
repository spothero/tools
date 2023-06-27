// Copyright 2023 SpotHero
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

// Config contains configuration for the cors package
type Config struct {
	// AllowedOrigins indicates which origins allow responses to be populated
	// with headers to allow cross origin requests (e.g. "*" or
	// "https://example.com")
	AllowedOrigins string
	// AllowedMethods indicates which methods allow responses to be populated
	// with headers to allow cross origin requests (e.g. "POST, GET, OPTIONS,
	// PUT, DELETE")
	AllowedMethods string
	// AllowedHeaders indicates which headers are allowed for cross origin
	// requests (e.g. "*" or "Accept, Content-Type, Content-Length")
	AllowedHeaders string
	// EnableMiddleware indicates whether or not CORS middleware is enabled to
	// enforce policies on cross origin requests
	EnableMiddleware bool
}
