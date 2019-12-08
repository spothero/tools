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

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/spothero/tools/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// bearerTokenType defines the standard expected form for OIDC JWT Tokens in
// Authorization metadata over GRPC.
const bearerTokenType = "bearer"

// GetContextAuth is a function which parameterizes and returns a function which extracts the
// Authorization data (if present) from incoming GRPC requests. If  Authorization data is
// found, this function attempts to parse and validate that value as a JWT  with the
// configured Credential types for the given JOSE provider.
func GetContextAuth(jh JOSEHandler, authRequired bool) func(context.Context) (context.Context, error) {
	return func(ctx context.Context) (context.Context, error) {
		var parseErrMsg string
		bearerToken, err := grpc_auth.AuthFromMD(ctx, bearerTokenType)
		if err != nil {
			parseErrMsg = fmt.Sprintf("no authorization bearer token found: %w", err)
		}

		var claims []Claim
		if len(parseErrMsg) == 0 {
			claims = jh.GetClaims()
			err := jh.ParseValidateJWT(bearerToken, claims...)
			if err != nil {
				log.Get(ctx).Debug(err.Error())
				parseErrMsg = invalidBearerToken
			}
		}

		if len(parseErrMsg) == 0 {
			// Populate each claim on the context, if any
			for _, claim := range claims {
				ctx = claim.NewContext(ctx)
			}
		}

		var finalErr error
		if authRequired && len(parseErrMsg) != 0 {
			finalErr = status.Errorf(codes.Unauthenticated, parseErrMsg)
		}
		return ctx, finalErr
	}
}
