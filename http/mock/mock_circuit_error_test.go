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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestError(t *testing.T) {
	assert.Equal(t, "message", CircuitError{Message: "message"}.Error())
}

func TestConcurrencyLimitReached(t *testing.T) {
	assert.True(t, CircuitError{ConcurrencyExceeded: true}.ConcurrencyLimitReached())
}

func TestCircuitOpen(t *testing.T) {
	assert.True(t, CircuitError{CircuitOpened: true}.CircuitOpen())
}
