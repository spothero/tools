package currency

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDollarsToPennies(t *testing.T) {
	tests := []struct {
		name     string
		dollars  float64
		roundUp  bool
		expected uint
	}{
		{
			name:     "even dollar amounts are correctly converted",
			dollars:  13.00,
			roundUp:  true,
			expected: 1300,
		},
		{
			name:     "uneven dollar amounts are correctly converted",
			dollars:  13.13,
			roundUp:  true,
			expected: 1313,
		},
		{
			name:     "pennies are added to fractional amounts when requested",
			dollars:  13.130000000001,
			roundUp:  true,
			expected: 1314,
		},
		{
			name:     "pennies are not added to fractional amounts when not requested",
			dollars:  13.130000000001,
			roundUp:  false,
			expected: 1313,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, DollarsToPennies(test.dollars, test.roundUp))
		})
	}
}
