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
	"github.com/getsentry/sentry-go"
	"github.com/spf13/pflag"
)

// Config defines the necessary configuration for instantiating a Sentry Reporter
type Config struct {
	DSN         string
	Environment string
	AppVersion  string
	Enabled     bool
}

// RegisterFlags registers Sentry flags with pflags
func (c *Config) RegisterFlags(flags *pflag.FlagSet) {
	flags.StringVar(&c.DSN, "sentry-dsn", "", "Sentry DSN")
	flags.BoolVarP(&c.Enabled, "sentry-logger-enabled", "t", true, "Enable Sentry")
}

// InitializeSentry Initializes the Sentry client. This function should be called as soon as
// possible after the application configuration is loaded so that sentry
// is setup.
func (c Config) InitializeSentry() error {
	if !c.Enabled {
		return nil
	}

	opts := sentry.ClientOptions{
		Dsn:         c.DSN,
		Environment: c.Environment,
		Release:     c.AppVersion,
	}
	if err := sentry.Init(opts); err != nil {
		return err
	}
	return nil
}
