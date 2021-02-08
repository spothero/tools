// Copyright 2021 SpotHero
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

import "github.com/spf13/pflag"

// RegisterFlags registers JOSE flags with pflags
func (c *Config) RegisterFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&c.EnableMiddleware, "cors-enable-middleware", c.EnableMiddleware, "Specify whether or not CORS middleware is enabled to  enforce policies on cross origin requests")
	flags.StringVar(&c.AllowedOrigins, "cors-allowed-origins", c.AllowedOrigins, "Specify which origin(s) allow responses to be populated with headers to allow cross origin requests (e.g. \"*\") or \"https://example.com\")")
	flags.StringVar(&c.AllowedMethods, "cors-allowed-methods", c.AllowedMethods, "Specify which method(s) allow responses to be populated with headers to allow cross origin requests (e.g. \"POST, GET, OPTIONS, PUT, DELETE\")")
	flags.StringVar(&c.AllowedMethods, "cors-allowed-headers", c.AllowedHeaders, "Specify which header(s) which headers are allowed for cross origin requests (e.g. \"*\" or \"Accept, Content-Type, Content-Length\")")
}
