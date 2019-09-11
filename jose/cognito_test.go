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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCognitoGeneratorNew(t *testing.T) {
	assert.Equal(t, &CognitoClaim{}, CognitoGenerator{}.New())
}

func TestCognitoClaimNewContext(t *testing.T) {
	ctx := context.Background()
	cc := CognitoClaim{
		ClientID: "abc123",
		Scope:    "all",
		TokenUse: "use",
		Version:  1,
	}

	expected := context.WithValue(context.Background(), CognitoClaimKey, &cc)
	assert.Equal(t, expected, cc.NewContext(ctx))
}
