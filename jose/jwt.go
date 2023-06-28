// Copyright 2023 SpotHero
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

// Handler defines an interface for interfacing with JOSE and JWT functionality
type Handler interface {
	// GetClaims returns an array containing empty claims
	GetClaims() []Claim
	// ParseValidateJWT accepts an input JWT string and populates any provided claims with
	// available claim data from the token.
	ParseValidateJWT(input string, claims ...Claim) error
}

// JOSE contains configuration for handling JWTs, JWKS, and other JOSE specifications
type JOSE struct {
	// Map of JSON Web Key Set (JWKS) URLs and their retrieved public keys
	jwks            map[string]*jose.JSONWebKeySet
	claimGenerators []ClaimGenerator
	validIssuers    []string
}

// JWTHeaderCtxKey is the type used to uniquely place the JWT Header in the context
type JWTHeaderCtxKey int

// JWTClaimKey is the value used to uniquely place the JWT Header within the context
const JWTClaimKey JWTHeaderCtxKey = iota

// NewJOSE creates and returns a JOSE client for use.
func (c Config) NewJOSE() JOSE {
	logger := log.Get(context.Background())
	if len(c.JSONWebKeySetURLs) == 0 {
		logger.Warn("no jwks urls specified - no authentication will be performed")
		return JOSE{}
	}

	// Fetch JSON Web Key Sets from the specified URL
	allJWKS := make(map[string]*jose.JSONWebKeySet)
	for _, jwksURL := range c.JSONWebKeySetURLs {
		jwks, err := getKeysForURL(jwksURL)
		if err != nil {
			// Log error and continue. Any jwks URL that failed to return a
			// valid set of keys will be retried as incoming tokens are decoded.
			logger.Error(err.Error())
		}

		allJWKS[jwksURL] = jwks
	}
	return JOSE{
		jwks:            allJWKS,
		validIssuers:    c.ValidIssuers,
		claimGenerators: c.ClaimGenerators,
	}
}

// getKeysForURL retrieves the set of public keys hosted at the specified URL
func getKeysForURL(jwksURL string) (*jose.JSONWebKeySet, error) {
	resp, err := http.Get(jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve jwks from url: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response from jwks url `%v`", resp.Status)
	}

	// Decode the response body into a JSONWebKeySet
	jwks := &jose.JSONWebKeySet{}
	err = json.NewDecoder(resp.Body).Decode(jwks)
	if err != nil {
		return nil, fmt.Errorf("failed to decode jwks json: %w", err)
	}

	return jwks, nil
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

	for jwksURL, keys := range j.jwks {
		// Lazily load any public keys that have not yet been successfully
		// retrieved and stored
		if keys == nil {
			keys, err = getKeysForURL(jwksURL)
			if err != nil {
				// Log error and continue. Another attempt will be made when the
				// next incoming token is decoded.
				log.Get(context.Background()).Error(err.Error())
				continue
			}
			// Save keys in map for next time
			j.jwks[jwksURL] = keys
		}

		err = tok.Claims(keys, allClaims...)
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
