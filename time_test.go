package tools

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
