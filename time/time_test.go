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

package time

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestLoadLocation(t *testing.T) {
	chiTz, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	tests := []struct {
		name            string
		locationName    string
		expectedOutcome *time.Location
	}{
		{
			"location should be loaded",
			"America/Chicago",
			chiTz,
		}, {
			"location should be loaded from cache",
			"America/Chicago",
			chiTz,
		}, {
			"error loading location should be passed to caller",
			"America/Flavortown",
			nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			loc, err := LoadLocation(test.locationName)
			if test.expectedOutcome != nil {
				assert.Equal(t, test.expectedOutcome, loc)
				assert.NoError(t, err)
				assert.Contains(t, locCache.cache, test.locationName)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
