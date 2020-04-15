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
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/spothero/tools/log"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

// Config contains configuration for the JOSE package
type Config struct {
	JSONWebKeySetURLs []string // JSON Web Key Set (JWKS) URLs for JSON Web Token (JWT) Verification
	ValidIssuers      []string // URL of the JWT Issuer for this environment
	// List of one or more claims to be captured from JWTs. If using http middleware,
	// these generators will determine which claims appear on the context.
	ClaimGenerators []ClaimGenerator
	AuthRequired    bool // If true, missing/invalid `Authorization` will result in an Unauthenticated error
}

// ClaimGenerator defines an interface which creates a JWT Claim
type ClaimGenerator interface {
	// New creates and returns a new Claim of the given underlying type
	New() Claim
}

// Claim defines an interface for common JWT claim functionality, such as registering claims to
// contexts.
type Claim interface {
	// NewContext accepts an input context and embeds the claim within the context, returning it
	// for further use
	NewContext(c context.Context) context.Context
}

// JOSEHandler defines an interface for interfacing with JOSE and JWT functionality
type JOSEHandler interface {
	// GetClaims returns an array containing empty claims
	GetClaims() []Claim
	// ParseValidateJWT accepts an input JWT string and populates any provided claims with
	// available claim data from the token.
	ParseValidateJWT(input string, claims ...Claim) error
}

// JOSE contains configuration for handling JWTs, JWKS, and other JOSE specifications
type JOSE struct {
	claimGenerators []ClaimGenerator
	validIssuers    []string
	jwks            []*jose.JSONWebKeySet
	authRequired    bool
}

// JWTHeaderCtxKey is the type used to uniquely place the JWT Header in the context
type JWTHeaderCtxKey int

// JWTClaimKey is the value used to uniquely place the JWT Header within the context
const JWTClaimKey JWTHeaderCtxKey = iota

// NewJOSE creates and returns a JOSE client for use.
func (c Config) NewJOSE() (JOSE, error) {
	logger := log.Get(context.Background())
	if len(c.JSONWebKeySetURLs) == 0 {
		logger.Warn("no jwks urls specified - no authentication will be performed")
		return JOSE{}, nil
	}

	// Fetch JSON Web Key Sets from the specified URL
	allJWKS := make([]*jose.JSONWebKeySet, 0)
	for _, jwks_url := range c.JSONWebKeySetURLs {
		resp, err := http.Get(jwks_url)
		if err != nil {
			return JOSE{}, fmt.Errorf("failed to retrieve jwks from url: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return JOSE{}, fmt.Errorf("received non-200 response from jwks url `%v`", resp.Status)
		}

		// Decode the response body into a JSONWebKeySet
		jwks := &jose.JSONWebKeySet{}
		err = json.NewDecoder(resp.Body).Decode(jwks)
		if err != nil {
			return JOSE{}, fmt.Errorf("failed to decoded jwks json: %w", err)
		}
		allJWKS = append(allJWKS, jwks)
	}
	return JOSE{
		jwks:            allJWKS,
		validIssuers:    c.ValidIssuers,
		claimGenerators: c.ClaimGenerators,
		authRequired:    c.AuthRequired,
	}, nil
}

// GetClaims returns a set of empty and initialized Claims registered to the JOSE struct
func (j JOSE) GetClaims() []Claim {
	claims := make([]Claim, len(j.claimGenerators))
	for i, generator := range j.claimGenerators {
		claims[i] = generator.New()
	}
	return claims
}

// ParseValidateJWT accepts a string containing a JWT token and attempts to parse and validate the
// token. If you wish to inspect other components of the payload, you may supply one or more
// claims structs which will be populated if the JWT is valid. Claims must be structs with json
// fields that match the keys in the payload field, or a map[string]interface{}. Use of
// map[string]interface{} is strongly discouraged.
func (j JOSE) ParseValidateJWT(input string, claims ...Claim) error {
	tok, err := jwt.ParseSigned(input)
	if err != nil {
		return fmt.Errorf("failed to parse jwt token: %w", err)
	}

	// Extract Token Claims from the payload and ensure that the signing signature is valid
	publicClaims := &jwt.Claims{}
	allClaims := []interface{}{publicClaims}
	for i := 0; i < len(claims); i++ {
		allClaims = append(allClaims, claims[i])
	}

	for _, jwks := range j.jwks {
		err = tok.Claims(jwks, allClaims...)
		if err == nil {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("failed to extract claims from jwt: %w", err)
	}

	// Validate that the claims were issued by a trusted source and are not expired
	for _, issuer := range j.validIssuers {
		if err = publicClaims.Validate(jwt.Expected{Issuer: issuer, Time: time.Now()}); err == nil {
			break
		}
	}
	return err
}
