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

import "github.com/spf13/pflag"

// RegisterFlags registers JOSE flags with pflags
func (c *Config) RegisterFlags(flags *pflag.FlagSet) {
	flags.StringVar(&c.JSONWebKeySetURL, "jose-jwks-url", c.JSONWebKeySetURL, "JSON Web Key Set (JWKS) URL for JSON Web Token (JWT) Verification")
	flags.StringVar(&c.ValidIssuer, "jose-valid-issuer", c.ValidIssuer, "Valid issuer (iss) of JWT tokens in this environment")
	flags.BoolVar(&c.AuthRequired, "jose-auth-required", true, "HTTP Middleware Only: If true (default), return a 4XX error if the `Authorization` header is missing or invalid. If false, ignores missing `Authorization`.")
}
