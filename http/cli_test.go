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
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestRegisterFlags(t *testing.T) {
	flags := pflag.NewFlagSet("pflags", pflag.PanicOnError)
	c := NewDefaultConfig("test")
	c.RegisterFlags(flags)
	err := flags.Parse(nil)
	assert.NoError(t, err)

	sn, err := flags.GetString("server-name")
	assert.NoError(t, err)
	assert.Equal(t, c.Name, sn)

	ad, err := flags.GetString("address")
	assert.NoError(t, err)
	assert.Equal(t, c.Address, ad)

	p, err := flags.GetInt("port")
	assert.NoError(t, err)
	assert.Equal(t, c.Port, p)

	rt, err := flags.GetInt("read-timeout")
	assert.NoError(t, err)
	assert.Equal(t, c.ReadTimeout, rt)

	wt, err := flags.GetInt("write-timeout")
	assert.NoError(t, err)
	assert.Equal(t, c.WriteTimeout, wt)

	hh, err := flags.GetBool("health-handler")
	assert.NoError(t, err)
	assert.Equal(t, c.HealthHandler, hh)

	mh, err := flags.GetBool("metrics-handler")
	assert.NoError(t, err)
	assert.Equal(t, c.MetricsHandler, mh)

	ph, err := flags.GetBool("pprof-handler")
	assert.NoError(t, err)
	assert.Equal(t, c.PprofHandler, ph)
}
