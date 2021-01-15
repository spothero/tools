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

package http

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Coordinates is an encapsulation of geospatial coordinates in decimal
// degrees.
type Coordinates struct {
	Latitude  float64
	Longitude float64
}

// ParseCoordinates reads and parses from the query parameters to the supplied
// request latitude and logitude corresponding to latFieldName and
// lonFieldName, respectively. An error is returned if only one of the named
// fields is present, if either value cannot be parsed as a float, or if either
// value is out of range for decimal latitude and longitude. The returned
// struct reference is nil if none of latFieldName or lonFieldName are present
// in the query parameters to the given request.
func ParseCoordinates(r *http.Request, latFieldName, lonFieldName string) (*Coordinates, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	latStr := r.Form.Get(latFieldName)
	lonStr := r.Form.Get(lonFieldName)
	latPresent, lonPresent := false, false
	var lat, lon float64

	if latStr != "" {
		// case: some value supplied for latitude
		_lat, errLat := strconv.ParseFloat(latStr, 64)
		if errLat != nil {
			return nil, fmt.Errorf("unable to parse %v", latFieldName)
		}
		lat = _lat
		latPresent = true
	}

	if lonStr != "" {
		// case: some value supplied for longitude
		_lon, errLat := strconv.ParseFloat(lonStr, 64)
		if errLat != nil {
			return nil, fmt.Errorf("unable to parse %v", lonFieldName)
		}
		lon = _lon
		lonPresent = true
	}

	if lat < -90 || lat > 90 {
		return nil, fmt.Errorf("%v must be in [-90, 90]", latFieldName)
	}

	if lon < -180 || lon > 180 {
		return nil, fmt.Errorf("%v must be in [-180, 180]", lonFieldName)
	}

	if latPresent != lonPresent {
		return nil, fmt.Errorf("both %v and %v must be provided", latFieldName, lonFieldName)
	} else if !latPresent && !lonPresent {
		return nil, nil
	}
	return &Coordinates{lat, lon}, nil
}

// ParseTime reads and parses from the query parameters to the supplied request
// a Time value corresponding to fieldName. Attempts are made to parse the Time
// value using each specified layout format in the order they are provided. An
// error is returned if a Time could not be parsed from the given field using
// any of the specified layouts. The zero-valued Time is returned if the given
// field is not present in the query parameters to the supplied request.
func ParseTime(r *http.Request, fieldName string, layouts []string) (time.Time, error) {
	if err := r.ParseForm(); err != nil {
		return time.Time{}, err
	}

	fieldStr := r.Form.Get(fieldName)
	if fieldStr == "" {
		return time.Time{}, nil
	}

	for _, layout := range layouts {
		parsed, err := time.Parse(layout, fieldStr)
		if err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse `%v`", fieldName)
}
