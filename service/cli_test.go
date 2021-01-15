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

package service

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestRegisterFlags(t *testing.T) {
	flags := pflag.NewFlagSet("pflags", pflag.PanicOnError)
	c := Config{
		Name:        "test",
		Version:     "0.1.0",
		GitSHA:      "abc123",
		Environment: "test",
	}
	c.RegisterFlags(flags)
	assert.NoError(t, flags.Parse(nil))

	n, err := flags.GetString("name")
	assert.NoError(t, err)
	assert.Equal(t, c.Name, n)

	e, err := flags.GetString("environment")
	assert.NoError(t, err)
	assert.Equal(t, c.Environment, e)
}

func TestCheckFlags(t *testing.T) {
	tests := []struct {
		name        string
		c           Config
		expectError bool
	}{
		{
			"a blank config leads to an error",
			Config{},
			true,
		},
		{
			"a populated config does not lead to an error",
			Config{
				Name:        "test",
				Version:     "0.1.0",
				GitSHA:      "abc123",
				Environment: "test",
			},
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.expectError {
				assert.Error(t, test.c.CheckFlags())
			} else {
				assert.NoError(t, test.c.CheckFlags())
			}
		})
	}
}
