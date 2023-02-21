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

package strings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringInSlice(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		inputSlice      []string
		expectedOutcome bool
	}{
		{
			"string found in slice returns true",
			"in-slice",
			[]string{"test", "test2", "in-slice", "test3"},
			true,
		},
		{
			"string not matching case returns false",
			"In-slice",
			[]string{"test", "test2", "in-slice", "test3"},
			false,
		},
		{
			"string not in slice returns false",
			"notInSlice",
			[]string{"test", "test2", "test3"},
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectedOutcome, StringInSlice(test.input, test.inputSlice))
		})
	}
}
