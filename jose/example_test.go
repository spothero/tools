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

package jose_test

import (
	"fmt"

	"github.com/spothero/tools/jose"
)

func Example() {
	// The JOSE Config allows users to configure the JOSE client for use with their OIDC provider
	c := jose.Config{
		// The "JWKS" endpoint for retrieving key sets for JWT verification
		JSONWebKeySetURLs: []string{"https://your-oidc-provider/.well-known/jwks.json"},
		// ValidIssuer is the URL of the JWT issuer for this environment. This must match the `iss`
		// within the JWT claim
		ValidIssuers: []string{"https://your-oidc-provider/issuerID"},
		// ClaimGenerators determine how JWT claims are to be parsed.
		ClaimGenerators: []jose.ClaimGenerator{
			jose.CognitoGenerator{},
		},
	}

	// Instantiate the JOSE provider
	client := c.NewJOSE()

	// With the instantiated client, callers may choose to add HTTP Middleware and GRPC
	// interceptors directly to their servers:
	//
	// httpMiddleware := jose.GetHTTPServerMiddleware(client),
	//
	// joseInterceptorFunc := jose.GetContextAuth(client)
	// grpcauth.UnaryServerInterceptor(joseInterceptorFunc)

	// The instantiated client may also be used directly to parse and validate JWTs
	claims := client.GetClaims()
	if err := client.ParseValidateJWT("<your-jwt>", claims...); err != nil {
		fmt.Println(fmt.Errorf("failed to parse token: %w", err))
	}
	fmt.Printf("successfully parsed token: %+v\n", claims[0])
}
