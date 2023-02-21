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

package mock

import (
	"fmt"
	"net/http"
)

// This roundtripper is a noop and is intended for use only within tests.
type RoundTripper struct {
	ResponseStatusCodes []int
	CreateErr           bool
	CallNumber          int
	// Optional, if specified raise this error type, otherwise a standard error is returned
	DesiredErr error
}

// RoundTrip Performs a "noop" round trip. It is intended for use only within tests.
func (mockRT *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	currCallNumber := mockRT.CallNumber
	mockRT.CallNumber++
	if mockRT.CreateErr {
		if mockRT.DesiredErr != nil {
			return nil, mockRT.DesiredErr
		}
		return nil, fmt.Errorf("error in roundtripper")
	}
	return &http.Response{StatusCode: mockRT.ResponseStatusCodes[currCallNumber]}, nil
}
