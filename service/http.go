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
	"fmt"

	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
	"github.com/spothero/tools/cli"
	shHTTP "github.com/spothero/tools/http"
	"github.com/spothero/tools/log"
	"github.com/spothero/tools/sentry"
	"github.com/spothero/tools/tracing"
	"go.uber.org/zap/zapcore"
)

type HTTPConfig struct {
	Config
	RegisterHandlers func(*mux.Router)
}

// ServerCmd creates and returns a Cobra and Viper command preconfigured to run a
// production-quality HTTP server. Note that this function returns the Default HTTP server for use
// at SpotHero. Consumers of the tools libraries are free to define their own server entrypoints if
// desired. This function is provided as a convenience function that should satisfy most use cases
// Note that Version and GitSHA *must be specified* before calling this function.
func (hc HTTPConfig) ServerCmd() *cobra.Command {
	// HTTP Config
	config := shHTTP.NewDefaultConfig(hc.Name)
	config.RegisterHandlers = hc.RegisterHandlers
	config.Middleware = shHTTP.Middleware{
		tracing.Middleware,
		shHTTP.NewMetrics(hc.Name, hc.Registry, true).Middleware,
		log.Middleware,
	}
	// Logging Config
	lc := &log.Config{
		UseDevelopmentLogger: true,
		Fields: map[string]interface{}{
			"version": hc.Version,
			"git_sha": hc.GitSHA,
		},
		Cores: []zapcore.Core{&sentry.Core{}},
	}
	// Sentry Config
	sc := sentry.Config{
		Environment: hc.Environment,
		AppVersion:  hc.Version,
	}
	cmd := &cobra.Command{
		Use:              hc.Name,
		Short:            "Starts and runs an HTTP Server",
		Long:             "Starts and runs an HTTP Server",
		Version:          fmt.Sprintf("%s (%s)", hc.Version, hc.GitSHA),
		PersistentPreRun: cli.CobraBindEnvironmentVariables(hc.Name),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := hc.CheckFlags(); err != nil {
				return err
			}
			if err := lc.InitializeLogger(); err != nil {
				return err
			}
			if err := sc.InitializeRaven(); err != nil {
				return err
			}
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
	return cmd
}
