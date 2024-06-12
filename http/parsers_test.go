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

package http

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseCoordinates(t *testing.T) {
	assertTest := assert.New(t)

	tests := []struct {
		name         string
		url          string
		latFieldName string
		lonFieldName string
		coords       *Coordinates
		errStr       string
	}{
		{
			"Typical success",
			"https://example.com/?lat=10&lon=10",
			"lat",
			"lon",
			&Coordinates{10.0, 10.0},
			"",
		},
		{
			"Missing latitude",
			"https://example.com/?lon=10",
			"lat",
			"lon",
			nil,
			"both lat and lon must be provided",
		},
		{
			"Missing longitude",
			"https://example.com/?lat=10",
			"lat",
			"lon",
			nil,
			"both lat and lon must be provided",
		},
		{
			"Unparceable latitude",
			"https://example.com/?lat=ten&lon=10",
			"lat",
			"lon",
			nil,
			"unable to parse lat",
		},
		{
			"Unparceable longitude",
			"https://example.com/?lat=10&lon=ten",
			"lat",
			"lon",
			nil,
			"unable to parse lon",
		},
		{
			"Latitude at the margin",
			"https://example.com/?lat=-90&lon=10",
			"lat",
			"lon",
			&Coordinates{-90, 10},
			"",
		},
		{
			"Latitude at the margin",
			"https://example.com/?lat=90&lon=10",
			"lat",
			"lon",
			&Coordinates{90, 10},
			"",
		},
		{
			"Latitude out of range",
			"https://example.com/?lat=-90.1&lon=10",
			"lat",
			"lon",
			nil,
			"lat must be in [-90, 90]",
		},
		{
			"Latitude out of range",
			"https://example.com/?lat=90.1&lon=10",
			"lat",
			"lon",
			nil,
			"lat must be in [-90, 90]",
		},
		{
			"Longitude at the margin",
			"https://example.com/?lat=-10&lon=-180",
			"lat",
			"lon",
			&Coordinates{-10, -180},
			"",
		},
		{
			"Longitude at the margin",
			"https://example.com/?lat=-10&lon=180",
			"lat",
			"lon",
			&Coordinates{-10, 180},
			"",
		},
		{
			"Longitude out of range",
			"https://example.com/?lat=10&lon=-180.1",
			"lat",
			"lon",
			nil,
			"lon must be in [-180, 180]",
		},
		{
			"Longitude out of range",
			"https://example.com/?lat=10&lon=180.1",
			"lat",
			"lon",
			nil,
			"lon must be in [-180, 180]",
		},
		{
			"Differently named latitude param",
			"https://example.com/?foo=10&lon=10",
			"foo",
			"lon",
			&Coordinates{10, 10},
			"",
		},
		{
			"Differently named longitude param",
			"https://example.com/?lat=10&bar=10",
			"lat",
			"bar",
			&Coordinates{10, 10},
			"",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(_ *testing.T) {
			r, _ := http.NewRequest(http.MethodGet, test.url, strings.NewReader(""))
			maybeCoords, err := ParseCoordinates(r, test.latFieldName, test.lonFieldName)
			assertTest.True((err == nil && test.errStr == "") || err != nil && err.Error() == test.errStr)
			assertTest.Equal(test.coords, maybeCoords)
		})
	}
}

func TestParseTime(t *testing.T) {
	assertTest := assert.New(t)

	buildTime := func(isoStr string) time.Time {
		parsedTime, err := time.Parse(time.RFC3339, isoStr)
		if err != nil {
			panic(fmt.Sprintf("Could not parse as RFC3339: %v", isoStr))
		}

		return parsedTime
	}

	tests := []struct {
		result    time.Time
		name      string
		url       string
		fieldName string
		layouts   []string
		err       bool
	}{
		{
			name:      "Basic success",
			url:       "https://example.com/?time=2019-10-07T15:07:39-05:00",
			fieldName: "time",
			layouts:   []string{time.RFC3339},
			result:    buildTime("2019-10-07T15:07:39-05:00"),
		},
		{
			name:      "Basic success with multiple layouts",
			url:       "https://example.com/?time=2019-10-07T15:07:39-05:00",
			fieldName: "time",
			layouts:   []string{"2006-01-02T15:04:05", "invalid", time.RFC3339},
			result:    buildTime("2019-10-07T15:07:39-05:00"),
		},
		{
			name:      "Differently named field",
			url:       "https://example.com/?foobar=2019-10-07T15:07:39-05:00",
			fieldName: "foobar",
			layouts:   []string{time.RFC3339},
			result:    buildTime("2019-10-07T15:07:39-05:00"),
		},
		{
			name:      "Unparseable time",
			url:       "https://example.com/?time=bogus",
			fieldName: "time",
			layouts:   []string{time.RFC3339},
			err:       true,
		},
		{
			name:      "Field missing from query",
			url:       "https://example.com/",
			fieldName: "time",
			layouts:   []string{time.RFC3339},
		},
		{
			name:      "Mismatched layout",
			url:       "https://example.com/?time=2019-10-07T15:07:39-05:00",
			fieldName: "time",
			layouts:   []string{"2006-01-02T15:04:05"},
			err:       true,
		},
		{
			name:      "Invalid layout",
			url:       "https://example.com/?time=2019-10-07T15:07:39-05:00",
			fieldName: "time",
			layouts:   []string{"invalid"},
			err:       true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(_ *testing.T) {
			r, _ := http.NewRequest(http.MethodGet, test.url, strings.NewReader(""))
			maybeTime, err := ParseTime(r, test.fieldName, test.layouts)
			assertTest.True((err == nil) != test.err)
			assertTest.Equal(test.result, maybeTime)
		})
	}
}
