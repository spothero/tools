// Copyright 2021 SpotHero
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

func TestIncYear(t *testing.T) {
	tests := []struct {
		name   string
		in     time.Time
		count  int
		expect time.Time
	}{
		{
			"Increment Year by 1",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			1,
			time.Date(2022, time.June, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			"Decrement Year by 1",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			-1,
			time.Date(2020, time.June, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			"Increment Year by 2",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			2,
			time.Date(2023, time.June, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			"Decrement Year by 2",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			-2,
			time.Date(2019, time.June, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IncYear(test.in, test.count)
			assert.Equal(t, test.expect, result)
		})
	}
}

func TestIncMonth(t *testing.T) {
	tests := []struct {
		name   string
		in     time.Time
		count  int
		expect time.Time
	}{
		{
			"Increment Month by 1",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			1,
			time.Date(2021, time.July, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			"Decrement Month by 1",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			-1,
			time.Date(2021, time.May, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			"Increment Month by 2",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			2,
			time.Date(2021, time.August, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			"Decrement Month by 2",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			-2,
			time.Date(2021, time.April, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IncMonth(test.in, test.count)
			assert.Equal(t, test.expect, result)
		})
	}
}

func TestIncWeek(t *testing.T) {
	tests := []struct {
		name   string
		in     time.Time
		count  int
		expect time.Time
	}{
		{
			"Increment Week by 1",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			1,
			time.Date(2021, time.June, 8, 0, 0, 0, 0, time.UTC),
		},
		{
			"Decrement Week by 1",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			-1,
			time.Date(2021, time.May, 25, 0, 0, 0, 0, time.UTC),
		},
		{
			"Increment Week by 2",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			2,
			time.Date(2021, time.June, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			"Decrement Week by 2",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			-2,
			time.Date(2021, time.May, 18, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IncWeek(test.in, test.count)
			assert.Equal(t, test.expect, result)
		})
	}
}

func TestIncDay(t *testing.T) {
	tests := []struct {
		name   string
		in     time.Time
		count  int
		expect time.Time
	}{
		{
			"Increment Day by 1",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			1,
			time.Date(2021, time.June, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			"Decrement Day by 1",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			-1,
			time.Date(2021, time.May, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			"Increment Day by 2",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			2,
			time.Date(2021, time.June, 3, 0, 0, 0, 0, time.UTC),
		},
		{
			"Decrement Day by 2",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			-2,
			time.Date(2021, time.May, 30, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IncDay(test.in, test.count)
			assert.Equal(t, test.expect, result)
		})
	}
}

func TestIncHour(t *testing.T) {
	tests := []struct {
		name   string
		in     time.Time
		count  int
		expect time.Time
	}{
		{
			"Increment Hour by 1",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			1,
			time.Date(2021, time.June, 1, 1, 0, 0, 0, time.UTC),
		},
		{
			"Decrement Hour by 1",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			-1,
			time.Date(2021, time.May, 31, 23, 0, 0, 0, time.UTC),
		},
		{
			"Increment Hour by 2",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			2,
			time.Date(2021, time.June, 1, 2, 0, 0, 0, time.UTC),
		},
		{
			"Decrement Hour by 2",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			-2,
			time.Date(2021, time.May, 31, 22, 0, 0, 0, time.UTC),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IncHour(test.in, test.count)
			assert.Equal(t, test.expect, result)
		})
	}
}

func TestIncMinute(t *testing.T) {
	tests := []struct {
		name   string
		in     time.Time
		count  int
		expect time.Time
	}{
		{
			"Increment Minute by 1",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			1,
			time.Date(2021, time.June, 1, 0, 1, 0, 0, time.UTC),
		},
		{
			"Decrement Minute by 1",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			-1,
			time.Date(2021, time.May, 31, 23, 59, 0, 0, time.UTC),
		},
		{
			"Increment Minute by 2",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			2,
			time.Date(2021, time.June, 1, 0, 2, 0, 0, time.UTC),
		},
		{
			"Decrement Minute by 2",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			-2,
			time.Date(2021, time.May, 31, 23, 58, 0, 0, time.UTC),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IncMinute(test.in, test.count)
			assert.Equal(t, test.expect, result)
		})
	}
}

func TestIncSecond(t *testing.T) {
	tests := []struct {
		name   string
		in     time.Time
		count  int
		expect time.Time
	}{
		{
			"Increment Second by 1",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			1,
			time.Date(2021, time.June, 1, 0, 0, 1, 0, time.UTC),
		},
		{
			"Decrement Second by 1",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			-1,
			time.Date(2021, time.May, 31, 23, 59, 59, 0, time.UTC),
		},
		{
			"Increment Second by 2",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			2,
			time.Date(2021, time.June, 1, 0, 0, 2, 0, time.UTC),
		},
		{
			"Decrement Second by 2",
			time.Date(2021, time.June, 1, 0, 0, 0, 0, time.UTC),
			-2,
			time.Date(2021, time.May, 31, 23, 59, 58, 0, time.UTC),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IncSecond(test.in, test.count)
			assert.Equal(t, test.expect, result)
		})
	}
}
