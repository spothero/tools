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
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/xerrors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestGetContextAuth(t *testing.T) {
	tests := []struct {
		name                string
		authMetadataPresent bool
		authMetadata        string
		jwt                 string
		parseJWTError       bool
		expectClaim         bool
		authRequired        bool
		expectErr           bool
		errorCode           codes.Code
	}{
		{
			"no auth metadata results in no claim, auth not required, no error returned",
			false,
			"",
			"",
			false,
			false,
			false,
			false,
			codes.OK,
		}, {
			"failed jwt parsings are rejected, auth not required, no error returned",
			true,
			"Bearer fake.jwt.header",
			"fake.jwt.header",
			true,
			false,
			false,
			false,
			codes.OK,
		}, {
			"with auth: no auth metadata results in no claim and a 401",
			false,
			"",
			"",
			false,
			false,
			true,
			true,
			codes.Unauthenticated,
		}, {
			"failed jwt parsings are rejected, auth required",
			true,
			"Bearer fake.jwt.header",
			"fake.jwt.header",
			true,
			false,
			true,
			true,
			codes.Unauthenticated,
		}, {
			"failed jwt parsings, auth not required, no error returned",
			true,
			"Bearer fake.jwt.header",
			"fake.jwt.header",
			true,
			false,
			false,
			false,
			codes.OK,
		}, {
			"jwt tokens are parsed and placed in context when present",
			true,
			"Bearer fake.jwt.header",
			"fake.jwt.header",
			false,
			true,
			true,
			false,
			codes.OK,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := &MockHandler{
				claimGenerators: []ClaimGenerator{MockGenerator{}},
			}

			var parseErr error
			if test.parseJWTError {
				parseErr = xerrors.Errorf("a jwt parsing error occurred in this test")
			}
			handler.On(
				"ParseValidateJWT",
				test.jwt,
				handler.GetClaims(),
			).Return(parseErr)

			joseInterceptor := GetContextAuth(handler, test.authRequired)

			ctx := context.Background()
			if test.authMetadataPresent {
				md := metadata.New(map[string]string{"authorization": test.authMetadata})
				ctx = metadata.NewIncomingContext(ctx, md)
			}

			ctx, err := joseInterceptor(ctx)
			assert.NotNil(t, ctx)
			if test.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, test.errorCode, status.Code(err))
		})
	}
}
