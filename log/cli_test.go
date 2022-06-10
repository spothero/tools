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

package log

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

	udl, err := flags.GetBool("use-development-logger")
	assert.NoError(t, err)
	assert.True(t, udl)

	lop, err := flags.GetStringArray("log-output-paths")
	assert.NoError(t, err)
	assert.Len(t, lop, 0)

	leop, err := flags.GetStringArray("log-error-output-paths")
	assert.NoError(t, err)
	assert.Len(t, leop, 0)

	ll, err := flags.GetString("log-level")
	assert.NoError(t, err)
	assert.Equal(t, "info", ll)

	lsi, err := flags.GetInt("log-sampling-initial")
	assert.NoError(t, err)
	assert.Equal(t, 100, lsi)

	lst, err := flags.GetInt("log-sampling-thereafter")
	assert.NoError(t, err)
	assert.Equal(t, 100, lst)

	e, err := flags.GetString("log-encoding")
	assert.NoError(t, err)
	assert.Equal(t, "json", e)
}
