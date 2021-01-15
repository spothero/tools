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

package tracing

import (
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestRegisterFlags(t *testing.T) {
	flags := pflag.NewFlagSet("pflags", pflag.PanicOnError)
	c := Config{}
	c.RegisterFlags(flags)
	err := flags.Parse(nil)
	assert.NoError(t, err)

	te, err := flags.GetBool("tracer-enabled")
	assert.NoError(t, err)
	assert.True(t, te)

	tst, err := flags.GetString("tracer-sampler-type")
	assert.NoError(t, err)
	assert.Equal(t, "", tst)

	tsp, err := flags.GetFloat64("tracer-sampler-param")
	assert.NoError(t, err)
	assert.Equal(t, 1.0, tsp)

	trls, err := flags.GetBool("tracer-reporter-log-spans")
	assert.NoError(t, err)
	assert.False(t, trls)

	trmqs, err := flags.GetInt("tracer-reporter-max-queue-size")
	assert.NoError(t, err)
	assert.Equal(t, 100, trmqs)

	trfi, err := flags.GetDuration("tracer-reporter-flush-interval")
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(1000000000), trfi)

	tah, err := flags.GetString("tracer-agent-host")
	assert.NoError(t, err)
	assert.Equal(t, "localhost", tah)

	tap, err := flags.GetInt("tracer-agent-port")
	assert.NoError(t, err)
	assert.Equal(t, 5775, tap)

	tsn, err := flags.GetString("tracer-service-name")
	assert.NoError(t, err)
	assert.Equal(t, "", tsn)
}
