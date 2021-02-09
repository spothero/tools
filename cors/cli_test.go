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

package cors

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

	enableMiddleware, err := flags.GetBool("cors-enable-middleware")
	assert.NoError(t, err)
	assert.Equal(t, false, enableMiddleware)

	origins, err := flags.GetString("cors-allowed-origins")
	assert.NoError(t, err)
	assert.Equal(t, "", origins)

	methods, err := flags.GetString("cors-allowed-methods")
	assert.NoError(t, err)
	assert.Equal(t, "", methods)

	headers, err := flags.GetString("cors-allowed-headers")
	assert.NoError(t, err)
	assert.Equal(t, "", headers)
}
