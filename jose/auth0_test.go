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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuth0GeneratorNew(t *testing.T) {
	assert.Equal(t, &Auth0Claim{}, Auth0Generator{}.New())
}

func TestAuth0ClaimNewContext(t *testing.T) {
	ctx := context.Background()
	cc := Auth0Claim{
		ID:        "abc123",
		Email:     "email",
		GrantType: "client-credentials",
	}

	expected := context.WithValue(context.Background(), Auth0ClaimKey, &cc)
	assert.Equal(t, expected, cc.NewContext(ctx))
}

func TestFromContext(t *testing.T) {
	// GIVEN an auth0 claim is present in a context
	ctx := context.Background()
	cc := Auth0Claim{
		ID:        "abc123",
		Email:     "email",
		GrantType: "client-credentials",
	}
	ctx = cc.NewContext(ctx)

	// WHEN we attempt to extract an auth0 claim
	maybeClaim, err := FromContext(ctx)

	// THEN an auth0 claim can be extracted from that context
	assert.NoError(t, err)
	assert.NotNil(t, maybeClaim)
	assert.Equal(t, cc, *maybeClaim)

	// GIVEN an empty context
	ctx2 := context.Background()

	// WHEN we attempt to extract an auth0 claim
	maybeClaim, err = FromContext(ctx2)

	// THEN an error occurs
	assert.Error(t, err)
	assert.Nil(t, maybeClaim)
}

func TestGetClientID(t *testing.T) {
	tests := []struct {
		name     string
		input    Auth0Claim
		expected string
	}{
		{
			name:     "client id present",
			input:    Auth0Claim{"id", "email", "client-credentials"},
			expected: "id",
		},
		{
			name:     "user id present",
			input:    Auth0Claim{"id", "email", "password"},
			expected: "",
		},
		{
			name:     "unknown grant",
			input:    Auth0Claim{"id", "email", "BoGuS"},
			expected: "",
		},
		{
			name:     "remove @clients suffix",
			input:    Auth0Claim{"leeroy-jenkins@clients", "email", "client-credentials"},
			expected: "leeroy-jenkins",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.input.GetClientID()
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestGetUserID(t *testing.T) {
	tests := []struct {
		name     string
		input    Auth0Claim
		expected string
	}{
		{
			name:     "client id present",
			input:    Auth0Claim{"client-id", "email", "client-credentials"},
			expected: "",
		},
		{
			name:     "user id presented as password",
			input:    Auth0Claim{"user-id", "email", "password"},
			expected: "user-id",
		},
		{
			name:     "user id presented as authorization_code",
			input:    Auth0Claim{"user-id", "email", "password"},
			expected: "user-id",
		},
		{
			name:     "unknown grant",
			input:    Auth0Claim{"id", "email", "BoGuS"},
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.input.GetUserID()
			assert.Equal(t, test.expected, actual)
		})
	}
}
