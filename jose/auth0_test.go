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
		UserID:   "abc123",
		ClientID: "abc123",
		Email:    "email",
	}

	expected := context.WithValue(context.Background(), Auth0ClaimKey, &cc)
	assert.Equal(t, expected, cc.NewContext(ctx))
}
