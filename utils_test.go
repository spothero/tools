// Copyright 2018 SpotHero
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

package tools

import (
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGetEnvVal(t *testing.T) {
	defaultVal := "FAKE"
	key := uuid.New().String()

	val := GetEnvVal(key, defaultVal)
	assert.Equal(t, "FAKE", val)

	os.Setenv(key, "NOT_FAKED")
	val = GetEnvVal(key, defaultVal)
	assert.Equal(t, "NOT_FAKED", val)

	os.Setenv(key, "")
	val = GetEnvVal(key, defaultVal)
	assert.Equal(t, "", val)
}