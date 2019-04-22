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

	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
	"github.com/spothero/tools"
	shHTTP "github.com/spothero/tools/http"
	"github.com/spothero/tools/log"
	"go.uber.org/zap/zapcore"
)

type HTTPConfig struct {
	Name             string                                                             // Name of the application server
	Version          string                                                             // Semantic Version of this Application
	GitSHA           string                                                             // Git SHA of the compiled Application
	Address          string                                                             // Address where the server will be acccessible. Default 0.0.0.0
	Port             int                                                                // Port where the server will be accessible. Default 8080
	RegisterHandlers func(*mux.Router)                                                  // Router registration callback
	PreStart         func(ctx context.Context, router *mux.Router, server *http.Server) // Server pre-start callback
	PostShutdown     func(ctx context.Context)                                          // Server post-shutdown callback
}

// DefaultServer creates and returns a Cobra and Viper command preconfigured to run a
// production-quality HTTP server.
func (hc HTTPConfig) DefaultServer() *cobra.Command {
	// HTTP Config
	config := shHTTP.NewDefaultConfig(hc.Name)
	config.Address = hc.Address
	if config.Address == "" {
		config.Address = "0.0.0.0"
	}
	config.Port = hc.Port
	if config.Port == 0 {
		config.Port = 8080
	}
	config.PreStart = hc.PreStart
	config.PostShutdown = hc.PostShutdown
	config.RegisterHandlers = hc.RegisterHandlers
	config.Middleware = shHTTP.Middleware{
		tools.TracingMiddleware,
		shHTTP.NewMetrics(hc.Name, nil, true).Middleware,
		log.LoggingMiddleware,
	}
	// Logging Config
	lc := &log.Config{
		UseDevelopmentLogger: true,
		Fields: map[string]interface{}{
			"version": hc.Version,
			"git_sha": hc.GitSHA,
		},
		Cores: []zapcore.Core{&tools.SentryCore{}},
	}
	cmd := &cobra.Command{
		Use:              hc.Name,
		Short:            "Starts and runs an HTTP Server",
		Long:             `Starts and runs an HTTP Server`,
		Version:          fmt.Sprintf("%s (%s)", hc.Version, hc.GitSHA),
		PersistentPreRun: tools.CobraBindEnvironmentVariables(hc.Name),
		RunE: func(cmd *cobra.Command, args []string) error {
			lc.InitializeLogger()
			config.NewServer().Run()
			return nil
		},
	}
	// Register Cobra/Viper CLI Flags
	flags := cmd.Flags()
	config.RegisterFlags(flags)
	lc.RegisterFlags(flags)
	return cmd
}
