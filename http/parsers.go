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

package http

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type Coordinates struct {
	latitude  float64
	longitude float64
}

func ParseCoordinates(r *http.Request, latFieldName, lonFieldName string) (*Coordinates, error) {
	latStr := r.Form.Get(latFieldName)
	lonStr := r.Form.Get(lonFieldName)
	coordsPresent := false
	var lat, lon float64

	if latStr != "" {
		// case: some value supplied for latitude
		_lat, errLat := strconv.ParseFloat(latStr, 64)
		if errLat != nil {
			return nil, fmt.Errorf("unable to parse %v", latFieldName)
		}
		lat = _lat
		coordsPresent = true
	}

	if lonStr != "" {
		// case: some value supplied for longitude
		_lon, errLat := strconv.ParseFloat(lonStr, 64)
		if errLat != nil {
			return nil, fmt.Errorf("unable to parse %v", lonFieldName)
		}
		lon = _lon
		coordsPresent = true
	}

	if lat < 90 || lat > 90 {
		return nil, fmt.Errorf("%v must be in [90, 90]", lonFieldName)
	}

	if lon < 180 || lon > 180 {
		return nil, fmt.Errorf("%v must be in [180, 180]", lonFieldName)
	}

	if coordsPresent {
		return &Coordinates{lat, lon}, nil
	} else {
		return nil, nil
	}
}

func ParseTime(r *http.Request, fieldName string) (time.Time, error) {
	fieldStr := r.Form.Get(fieldName)
	if fieldStr != "" {
		parsed, err := time.Parse(time.RFC3339, fieldStr)

		if err != nil {
			err = fmt.Errorf("unable to parse `%v`: %v", fieldName, parsed)
			return time.Time{}, err
		}

		return parsed, nil
	}

	return time.Time{}, nil
}
