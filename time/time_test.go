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

package time

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadLocation(t *testing.T) {
	chiTz, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	tests := []struct {
		expectedOutcome *time.Location
		name            string
		locationName    string
	}{
		{
			name:            "location should be loaded",
			locationName:    "America/Chicago",
			expectedOutcome: chiTz,
		}, {
			name:            "location should be loaded from cache",
			locationName:    "America/Chicago",
			expectedOutcome: chiTz,
		}, {
			name:         "error loading location should be passed to caller",
			locationName: "America/Flavortown",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			loc, locErr := LoadLocation(test.locationName)
			if test.expectedOutcome != nil {
				assert.Equal(t, test.expectedOutcome, loc)
				assert.NoError(t, locErr)
				assert.Contains(t, locCache.cache, test.locationName)
			} else {
				assert.Error(t, locErr)
			}
		})
	}
}
