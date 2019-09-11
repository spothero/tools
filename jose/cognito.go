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
)

// CognitoCtxKey is the type used to uniquely place the cognito claim in the context
type CognitoCtxKey int

// CognitoClaimKey is the value used to uniquely place the cognito claim within the context
const CognitoClaimKey CognitoCtxKey = iota

// CognitoGenerator satisfies the ClaimGenerator interface, allowing middleware to create
// intermediate Claim objects without specific knowledge of the underlying implementing types.
type CognitoGenerator struct{}

// CognitoClaim defines a JWT Claim for tokens issued by the AWS Cognito Service
type CognitoClaim struct {
	TokenUse string `json:"token_use"`
	Scope    string `json:"scope"`
	ClientID string `json:"client_id"`
	Version  int    `json:"version"`
}

// New satisfies the ClaimGenerator interface, returning an empty claim for use with JOSE parsing
// and validation.
func (cg CognitoGenerator) New() Claim {
	return &CognitoClaim{}
}

// NewContext registers a claim to a given context and returns that new context
func (cc CognitoClaim) NewContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, CognitoClaimKey, &cc)
}
