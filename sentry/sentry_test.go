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

package sentry

import (
	"testing"

	"github.com/getsentry/sentry-go"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	flags := pflag.NewFlagSet("pflags", pflag.PanicOnError)
	c := Config{}
	c.RegisterFlags(flags)
	err := flags.Parse(nil)
	assert.NoError(t, err)

	dsn, err := flags.GetString("sentry-dsn")
	assert.NoError(t, err)
	assert.Equal(t, "", dsn)
}

func TestInitializeSentry(t *testing.T) {
	tests := []struct {
		name                string
		config              Config
		expectClientCreated bool
	}{
		{
			"successfully initialize sentry",
			Config{Enabled: true},
			true,
		},
		{
			"no client created when sentry is disabled",
			Config{
				Enabled: false,
			},
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.NoError(t, test.config.InitializeSentry())

			sentryClient := sentry.CurrentHub().Client()
			if test.expectClientCreated {
				assert.NotNil(t, sentryClient)
			} else {
				assert.Nil(t, sentryClient)
			}

			// Reset client
			sentry.CurrentHub().BindClient(nil)
		})
	}
}
