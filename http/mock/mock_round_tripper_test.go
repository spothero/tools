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

package mock

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name        string
		statusCodes []int
		expectErr   bool
	}{
		{
			"the set status code is returned with no error",
			[]int{http.StatusOK},
			false,
		},
		{
			"if an error is requested, an error is returned",
			[]int{http.StatusOK},
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mrt := RoundTripper{
				ResponseStatusCodes: test.statusCodes,
				CreateErr:           test.expectErr,
			}
			resp, err := mrt.RoundTrip(nil)
			if test.expectErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			}
		})
	}
}
