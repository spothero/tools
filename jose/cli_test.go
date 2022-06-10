// Copyright 2022 SpotHero
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

package jose

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestRegisterFlags(t *testing.T) {
	flags := pflag.NewFlagSet("pflags", pflag.PanicOnError)
	c := Config{}
	c.RegisterFlags(flags)
	err := flags.Parse(nil)
	assert.NoError(t, err)

	ju, err := flags.GetStringSlice("jose-jwks-urls")
	assert.NoError(t, err)
	assert.Equal(t, []string{}, ju)

	vi, err := flags.GetStringSlice("jose-valid-issuers")
	assert.NoError(t, err)
	assert.Equal(t, []string{}, vi)
}
