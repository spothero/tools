// Copyright 2020 SpotHero
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

// Auth0CtxKey is the type used to uniquely place the cognito claim in the context
type Auth0CtxKey int

// Auth0ClaimKey is the value used to uniquely place the cognito claim within the context
const Auth0ClaimKey Auth0CtxKey = iota

// Auth0Generator satisfies the ClaimGenerator interface, allowing middleware to create
// intermediate Claim objects without specific knowledge of the underlying implementing types.
type Auth0Generator struct{}

// Auth0Claim defines a JWT Claim for tokens issued by the AWS Auth0 Service
type Auth0Claim struct {
	UserID   string `json:"sub"`
	ClientID string `json:"aud"`
	Email    string `json:"email"`
}

// New satisfies the ClaimGenerator interface, returning an empty claim for use with JOSE parsing
// and validation.
func (cg Auth0Generator) New() Claim {
	return &Auth0Claim{}
}

// NewContext registers a claim to a given context and returns that new context
func (cc Auth0Claim) NewContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, Auth0ClaimKey, &cc)
}
