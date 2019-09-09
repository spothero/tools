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
	"encoding/json"
	"fmt"
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
}

// JOSE contains configuration for handling JWTs, JWKS, and other JOSE specifications
type JOSE struct {
	validIssuer string
	jwks        *jose.JSONWebKeySet
}

// CognitoClaim defines a JWT Claim for tokens issued by the AWS Cognito Service
type CognitoClaim struct {
	TokenUse string `json:"token_use"`
	Scope    string `json:"scope"`
	ClientID string `json:"client_id"`
	Version  int    `json:"version"`
}

// NewJOSE creates and returns a JOSE client for use.
func (c Config) NewJOSE() (JOSE, error) {
	if len(c.JSONWebKeySetURL) == 0 {
		return JOSE{}, fmt.Errorf("no jwks url specified and jwt signature verification enabled")
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
		jwks:        jwks,
		validIssuer: c.ValidIssuer,
	}, nil
}

// ParseJWT accepts a string containing a JWT token and attempts to parse and validate the token.
// If you wish to inspect other components of the payload, you may supply one or more claims
// structs which will be populated if the JWT is valid. Claims must be structs with json fields
// that match the keys in the payload field, or a map[string]interface{}. Use of
// map[string]interface{} is strongly discouraged.
func (j JOSE) ParseJWT(input string, claims ...interface{}) error {
	tok, err := jwt.ParseSigned(input)
	if err != nil {
		return xerrors.Errorf("failed to parse jwt token: %w", err)
	}

	// Extract Token Claims from the payload and ensure that the signing signature is valid
	publicClaims := jwt.Claims{}
	if err = tok.Claims(j.jwks, append(claims, &publicClaims)...); err != nil {
		return xerrors.Errorf("failed to extract claims from jwt: %w", err)
	}

	// Validate that the claims were issued by a trusted source and are not expired
	return publicClaims.Validate(jwt.Expected{
		Issuer: j.validIssuer,
		Time:   time.Now(),
	})
}
