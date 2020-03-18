// Copyright 2020 SpotHero
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
	"context"

	"github.com/spf13/pflag"
	shGRPC "github.com/spothero/tools/grpc"
	shHTTP "github.com/spothero/tools/http"
	"github.com/spothero/tools/jose"
	"github.com/spothero/tools/log"
	"github.com/spothero/tools/sentry"
	"github.com/spothero/tools/tracing"
)

// options is a collection of all configurable service options
type options struct {
	httpConfig    shHTTP.Config
	grpcConfig    shGRPC.Config
	logConfig     log.Config
	sentryConfig  sentry.Config
	tracingConfig tracing.Config
	joseConfig    jose.Config
}

// configHandler defines a struct for pre and post config functions
type configHandler struct {
	preCmd  func(*pflag.FlagSet)
	postCmd func(*pflag.FlagSet)
}

// V2Config contains the new service config
type V2Config struct {
	config   Config
	options                  // service configuration options
	handlers []configHandler // configHandlers is a list of all user-provided configuration
}

// WithClaimGenerators sets JOSE and JWT Claim Generators for authentication
func (c *V2Config) WithClaimGenerators(generators ...jose.ClaimGenerator) *V2Config {
	c.handlers = append(c.handlers, configHandler{
		preCmd: func(flags *pflag.FlagSet) {
			c.options.joseConfig.ClaimGenerators = generators
		},
		postCmd: func(flags *pflag.FlagSet) {},
	})
	return c
}

// NewService constructs and returns the default Service
func (c Config) NewService(
	ctx context.Context,
	shortDescription, longDescription string,
) *V2Config {
	return &V2Config{
		config:   c,
		options:  options{},
		handlers: make([]configHandler, 0),
	}
}
