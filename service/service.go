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
	"strings"
	"sync"

	"github.com/gorilla/mux"
	grpcauth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpcot "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpcprom "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/spf13/cobra"
	"github.com/spothero/tools/cli"
	shGRPC "github.com/spothero/tools/grpc"
	shHTTP "github.com/spothero/tools/http"
	"github.com/spothero/tools/jose"
	"github.com/spothero/tools/log"
	"github.com/spothero/tools/sentry"
	"github.com/spothero/tools/tracing"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
)

// HTTPService implementers register HTTP routes with a mux router.
type HTTPService interface {
	RegisterHandlers(router *mux.Router)
}

// GRPCService implementors register GRPC APIs with the GRPC server
type GRPCService interface {
	RegisterAPIs(*grpc.Server)
}

// ServerCmd takes functions, newHTTPService and newGRPCService, that instantiate
// the GRPCService and HTTPService by consuming the Config object after all values
// are populated from the CLI and/or environment variables so that values configured
// by this package are accessible by newService.
//
// Note that this function creates the default server configuration (grpc and http)
// for use at SpotHero. Consumers of the tools libraries are free to define their
// own server entrypoints if desired. This function is provided as a convenience
// function that should satisfy most use cases.
//
// Note that Version and GitSHA *must be specified* before calling this function.
func (c Config) ServerCmd(
	shortDescription, longDescription string,
	newHTTPService func(Config) HTTPService,
	newGRPCService func(Config) GRPCService,
) *cobra.Command {
	// HTTP Config
	httpConfig := shHTTP.NewDefaultConfig(c.Name)
	httpConfig.PreStart = c.PreStartHTTP
	httpConfig.PostShutdown = c.PostShutdownHTTP
	httpConfig.Middleware = []mux.MiddlewareFunc{
		tracing.HTTPMiddleware,
		shHTTP.NewMetrics(c.Registry, true).Middleware,
		log.HTTPMiddleware,
		sentry.NewMiddleware().HTTP,
	}

	// GRPC Config
	grpcConfig := shGRPC.Config{}
	if newGRPCService != nil {
		grpcConfig = shGRPC.NewDefaultConfig(c.Name, newGRPCService(c).RegisterAPIs)
	}
	if len(c.CancelSignals) > 0 {
		grpcConfig.CancelSignals = c.CancelSignals
		httpConfig.CancelSignals = c.CancelSignals
	}
	// Logging Config
	lc := &log.Config{
		UseDevelopmentLogger: true,
		Fields: map[string]interface{}{
			"version": c.Version,
			"git_sha": c.GitSHA[len(c.GitSHA)-6:], // Log only the last 6 digits of the Git SHA
		},
		Cores: []zapcore.Core{&sentry.Core{}},
	}
	// Sentry Config
	sc := sentry.Config{
		Environment: c.Environment,
		AppVersion:  c.Version,
	}
	// Tracing Config
	tc := tracing.Config{ServiceName: c.Name}
	// Jose Config
	jc := jose.Config{ClaimGenerators: []jose.ClaimGenerator{jose.CognitoGenerator{}}}
	cmd := &cobra.Command{
		Use:              c.Name,
		Short:            shortDescription,
		Long:             longDescription,
		Version:          fmt.Sprintf("%s (%s)", c.Version, c.GitSHA),
		PersistentPreRun: cli.CobraBindEnvironmentVariables(strings.Replace(c.Name, "-", "_", -1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.CheckFlags(); err != nil {
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

			// Ensure that GRPC Interceptors capture histograms
			grpcprom.EnableHandlingTimeHistogram()
			grpcConfig.UnaryInterceptors = []grpc.UnaryServerInterceptor{
				grpcot.UnaryServerInterceptor(),
				tracing.UnaryServerInterceptor,
				log.UnaryServerInterceptor,
				grpcprom.UnaryServerInterceptor,
				sentry.UnaryServerInterceptor,
			}
			grpcConfig.StreamInterceptors = []grpc.StreamServerInterceptor{
				grpcot.StreamServerInterceptor(),
				tracing.StreamServerInterceptor,
				log.StreamServerInterceptor,
				grpcprom.StreamServerInterceptor,
				sentry.StreamServerInterceptor,
			}

			// If the user has requested JOSE Auth, add JOSE Auth interceptors
			if jc.AuthRequired {
				jh, err := jc.NewJOSE()
				if err != nil {
					return err
				}
				joseInterceptorFunc := jose.GetContextAuth(jh, jc.AuthRequired)
				grpcConfig.UnaryInterceptors = append(
					grpcConfig.UnaryInterceptors,
					grpcauth.UnaryServerInterceptor(joseInterceptorFunc),
				)
				grpcConfig.StreamInterceptors = append(
					grpcConfig.StreamInterceptors,
					grpcauth.StreamServerInterceptor(joseInterceptorFunc),
				)
				httpConfig.Middleware = append(
					httpConfig.Middleware,
					jose.GetHTTPMiddleware(jh, jc.AuthRequired),
				)
			}
			var wg sync.WaitGroup
			if newGRPCService != nil {
				wg.Add(1)
				go func() {
					defer wg.Done()
					if err := grpcConfig.NewServer().Run(); err != nil {
						log.Get(context.Background()).Error("failed to run the grpc server", zap.Error(err))
					}
				}()
			}
			if newHTTPService != nil {
				wg.Add(1)
				go func() {
					defer wg.Done()
					httpService := newHTTPService(c)
					httpConfig.RegisterHandlers = httpService.RegisterHandlers
					httpConfig.NewServer().Run()
				}()
			}
			wg.Wait()
			return nil
		},
	}
	// Register Cobra/Viper CLI Flags
	flags := cmd.Flags()
	c.RegisterFlags(flags)
	httpConfig.RegisterFlags(flags)
	grpcConfig.RegisterFlags(flags)
	lc.RegisterFlags(flags)
	sc.RegisterFlags(flags)
	tc.RegisterFlags(flags)
	jc.RegisterFlags(flags)
	return cmd
}
