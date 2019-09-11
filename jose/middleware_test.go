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
	"net/http/httptest"
	"testing"

	"github.com/spothero/tools/http/writer"
	"github.com/stretchr/testify/assert"
	"golang.org/x/xerrors"
)

func TestGetHTTPMiddleware(t *testing.T) {
	tests := []struct {
		name              string
		authHeaderPresent bool
		authHeader        string
		jwt               string
		parseJWTError     bool
		expectClaim       bool
	}{
		{
			"no auth header results in no claim",
			false,
			"",
			"",
			false,
			false,
		}, {
			"malformed auth headers are rejected",
			true,
			"bearer fake.jwt.header",
			"",
			false,
			false,
		}, {
			"failed jwt parsings are rejected",
			true,
			"Bearer fake.jwt.header",
			"fake.jwt.header",
			true,
			false,
		}, {
			"jwt tokens are parsed and placed in context when present",
			true,
			"Bearer fake.jwt.header",
			"fake.jwt.header",
			false,
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			sr := writer.StatusRecorder{ResponseWriter: recorder, StatusCode: http.StatusOK}
			req, err := http.NewRequest("GET", "/", nil)
			if test.authHeaderPresent {
				req.Header.Add("Authorization", test.authHeader)
			}
			assert.NoError(t, err)

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
				[]interface{}{handler.GetClaims()},
			).Return(parseErr)
			deferable, r := GetHTTPMiddleware(handler)(&sr, req)
			defer deferable()
			assert.NotNil(t, r)

			value, ok := r.Context().Value(MockClaimKey).(*MockClaim)
			if test.expectClaim {
				assert.True(t, ok)
				assert.Equal(t, &MockClaim{}, value)
			} else {
				assert.False(t, ok)
				assert.Nil(t, value)
			}
		})
	}
}
