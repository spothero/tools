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

	"github.com/stretchr/testify/mock"
)

// mockCtxKey is the type used to uniquely place the mock claim in the context
type mockCtxKey int

// MockClaimKey is the value used to uniquely place the mock claim within the context
const MockClaimKey mockCtxKey = iota

// MockGenerator satisfies the ClaimGenerator interface
type MockGenerator struct{}

// MockClaim defines a JWT Claim for tokens
type MockClaim struct {
	contents string `json:"contents"`
}

// MockHandler defines an interface for mocking JOSE and JWT functionality
type MockHandler struct {
	mock.Mock
	claimGenerators []ClaimGenerator
}

// New satisfies the ClaimGenerator interface, returning an empty claim for use with JOSE parsing
// and validation.
func (mg MockGenerator) New() Claim {
	return &MockClaim{}
}

// WithContext registers a claim to a given context
func (mc MockClaim) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, MockClaimKey, &mc)
}

// GetClaims mocks retrieval of claim instances
func (mh MockHandler) GetClaims() []Claim {
	claims := make([]Claim, len(mh.claimGenerators))
	for i, generator := range mh.claimGenerators {
		claims[i] = generator.New()
	}
	return claims
}

// ParseValidateJWT mocks the ParseValidateJWT function
func (mh MockHandler) ParseValidateJWT(input string, claims ...interface{}) error {
	return mh.Called(input, claims).Error(0)
}
