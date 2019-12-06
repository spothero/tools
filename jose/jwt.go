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
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/xerrors"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

// Config contains configuration for the JOSE package
type Config struct {
	JSONWebKeySetURL string // JSON Web Key Set (JWKS) URL for JSON Web Token (JWT) Verification
	ValidIssuer      string // URL of the JWT Issuer for this environment
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
	validIssuer     string
	jwks            *jose.JSONWebKeySet
	authRequired    bool
}

// NewJOSE creates and returns a JOSE client for use.
func (c Config) NewJOSE() (JOSE, error) {
	if len(c.JSONWebKeySetURL) == 0 {
		return JOSE{}, xerrors.Errorf("no jwks url specified")
	}

	// Fetch JSON Web Key Sets from the specified URL
	resp, err := http.Get(c.JSONWebKeySetURL)
	if err != nil {
		return JOSE{}, xerrors.Errorf("failed to retrieve jwks from url: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return JOSE{}, xerrors.Errorf("received non-200 response from jwks url `%v`", resp.Status)
	}

	// Decode the response body into a JSONWebKeySet
	jwks := &jose.JSONWebKeySet{}
	err = json.NewDecoder(resp.Body).Decode(jwks)
	if err != nil {
		return JOSE{}, xerrors.Errorf("failed to decoded jwks json: %w", err)
	}

	return JOSE{
		jwks:            jwks,
		validIssuer:     c.ValidIssuer,
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
		return xerrors.Errorf("failed to parse jwt token: %w", err)
	}

	// Extract Token Claims from the payload and ensure that the signing signature is valid
	publicClaims := &jwt.Claims{}
	allClaims := []interface{}{publicClaims}
	for i := 0; i < len(claims); i++ {
		allClaims = append(allClaims, claims[i])
	}

	if err = tok.Claims(j.jwks, allClaims...); err != nil {
		return xerrors.Errorf("failed to extract claims from jwt: %w", err)
	}

	// Validate that the claims were issued by a trusted source and are not expired
	return publicClaims.Validate(jwt.Expected{
		Issuer: j.validIssuer,
		Time:   time.Now(),
	})
}
