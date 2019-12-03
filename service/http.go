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

package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
	"github.com/spothero/tools/cli"
	shHTTP "github.com/spothero/tools/http"
	"github.com/spothero/tools/log"
	"github.com/spothero/tools/sentry"
	"github.com/spothero/tools/tracing"
	"go.uber.org/zap/zapcore"
)

// HTTPConfig contains required configuration for starting an HTTP service
type HTTPConfig struct {
	Config
	PreStart     func(ctx context.Context, router *mux.Router, server *http.Server) // A function to be called before starting the web server
	PostShutdown func(ctx context.Context)                                          // A function to be called before stopping the web server
}

// HTTPService implementers register HTTP routes with a mux router.
type HTTPService interface {
	RegisterHandlers(router *mux.Router)
}

// ServerCmd creates and returns a Cobra and Viper command preconfigured to run a
// production-quality HTTP server. This method takes a function that instantiates a HTTPService interface
// that passes through the HTTPConfig object to the constructor after all values are populated from
// the CLI and/or environment variables so that values configured by this package are accessible
// downstream.
//
// Note that this function returns the Default HTTP server for use
// at SpotHero. Consumers of the tools libraries are free to define their own server entrypoints if
// desired. This function is provided as a convenience function that should satisfy most use cases
// Note that Version and GitSHA *must be specified* before calling this function.
func (hc HTTPConfig) ServerCmd(shortDescript, longDescript string, newService func(HTTPConfig) HTTPService) *cobra.Command {
	// HTTP Config
	config := shHTTP.NewDefaultConfig(hc.Name)
	config.PreStart = hc.PreStart
	config.PostShutdown = hc.PostShutdown
	config.Middleware = []mux.MiddlewareFunc{
		tracing.HTTPMiddleware,
		shHTTP.NewMetrics(hc.Registry, true).Middleware,
		log.HTTPMiddleware,
		sentry.NewMiddleware().HTTP,
	}
	// Logging Config
	lc := &log.Config{
		UseDevelopmentLogger: true,
		Fields: map[string]interface{}{
			"version": hc.Version,
			"git_sha": hc.GitSHA[len(hc.GitSHA)-6:], // Log only the last 6 digits of the Git SHA
		},
		Cores: []zapcore.Core{&sentry.Core{}},
	}
	// Sentry Config
	sc := sentry.Config{
		Environment: hc.Environment,
		AppVersion:  hc.Version,
	}
	// Tracing Config
	tc := tracing.Config{ServiceName: hc.Name}
	cmd := &cobra.Command{
		Use:              hc.Name,
		Short:            shortDescript,
		Long:             longDescript,
		Version:          fmt.Sprintf("%s (%s)", hc.Version, hc.GitSHA),
		PersistentPreRun: cli.CobraBindEnvironmentVariables(strings.Replace(hc.Name, "-", "_", -1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := hc.CheckFlags(); err != nil {
				return err
			}
			if err := lc.InitializeLogger(); err != nil {
				return err
			}
			if err := sc.InitializeSentry(); err != nil {
				return err
			}
			closer := tc.ConfigureTracer()
			defer closer.Close()
			httpService := newService(hc)
			config.RegisterHandlers = httpService.RegisterHandlers
			config.NewServer().Run()
			return nil
		},
	}
	// Register Cobra/Viper CLI Flags
	flags := cmd.Flags()
	hc.RegisterFlags(flags)
	config.RegisterFlags(flags)
	lc.RegisterFlags(flags)
	sc.RegisterFlags(flags)
	tc.RegisterFlags(flags)
	return cmd
}
