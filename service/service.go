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
	"fmt"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	grpcauth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
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
	ctx context.Context,
	shortDescription, longDescription string,
	newHTTPService func(Config) HTTPService,
	newGRPCService func(Config) GRPCService,
) *cobra.Command {
	// HTTP Config
	httpConfig := shHTTP.NewDefaultConfig(c.Name)
	httpConfig.Middleware = []mux.MiddlewareFunc{
		tracing.HTTPServerMiddleware,
		shHTTP.NewMetrics(c.Registry, true).Middleware,
		log.HTTPServerMiddleware,
		sentry.NewMiddleware().HTTP,
	}

	// GRPC Config
	// XXX: passing `nil` as newGRPCService is a hack to delay the calling of
	// that closure until control reaches the `RunE` function.
	// see reference:f9d302c2-df3f-4110-9529-94b0515c4a17 in this file.
	// Follow-up: https://spothero.atlassian.net/browse/PMP-402
	grpcConfig := shGRPC.NewDefaultConfig(c.Name, nil)

	if len(c.CancelSignals) > 0 {
		grpcConfig.CancelSignals = c.CancelSignals
		httpConfig.CancelSignals = c.CancelSignals
	}
	// Logging Config
	lc := &log.Config{
		UseDevelopmentLogger: true,
		Fields: map[string]interface{}{
			"version": c.Version,
			"git_sha": c.GitSHA[:6], // Log only the first 6 digits of the Git SHA
		},
		Cores: []zapcore.Core{&sentry.Core{LevelEnabler: zap.InfoLevel}},
	}
	// Sentry Config
	sc := sentry.Config{AppVersion: c.Version}
	// Tracing Config
	tc := tracing.Config{ServiceName: c.Name}
	// Jose Config
	jc := jose.Config{
		ClaimGenerators: []jose.ClaimGenerator{
			jose.CognitoGenerator{},
			jose.Auth0Generator{},
		},
	}
	cmd := &cobra.Command{
		Use:              c.Name,
		Short:            shortDescription,
		Long:             longDescription,
		Version:          fmt.Sprintf("%s (%s)", c.Version, c.GitSHA),
		PersistentPreRun: cli.CobraBindEnvironmentVariables(strings.Replace(c.Name, "-", "_", -1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			sc.Environment = c.Environment

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

			// Ensure that gRPC Interceptors capture histograms
			grpcprom.EnableHandlingTimeHistogram()
			grpcConfig.UnaryInterceptors = []grpc.UnaryServerInterceptor{
				grpcot.UnaryServerInterceptor(),
				tracing.UnaryServerInterceptor,
				log.UnaryServerInterceptor,
				grpcprom.UnaryServerInterceptor,
			}
			grpcConfig.StreamInterceptors = []grpc.StreamServerInterceptor{
				grpcot.StreamServerInterceptor(),
				tracing.StreamServerInterceptor,
				log.StreamServerInterceptor,
				grpcprom.StreamServerInterceptor,
			}

			// Add JOSE Auth interceptors
			jh, err := jc.NewJOSE()
			if err != nil {
				return err
			}
			joseInterceptorFunc := jose.GetContextAuth(jh)
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
				jose.GetHTTPServerMiddleware(jh),
			)

			// Add panic handlers to the middleware. Panic handlers should always come last,
			// because they can help recover error state such that it is correctly handled by
			// upstream interceptors.
			grpcConfig.UnaryInterceptors = append(
				grpcConfig.UnaryInterceptors,
				grpcrecovery.UnaryServerInterceptor(),
				sentry.UnaryServerInterceptor,
			)
			grpcConfig.StreamInterceptors = append(
				grpcConfig.StreamInterceptors,
				grpcrecovery.StreamServerInterceptor(),
				sentry.StreamServerInterceptor,
			)

			if c.PreStart != nil {
				var err error
				ctx, err = c.PreStart(ctx)
				if err != nil {
					return err
				}
			}

			var wg sync.WaitGroup
			if newGRPCService != nil {
				// XXX: here we mutate grpc.Config, which is hitherto nil; this
				// is done in order to defer calling newGRPCService until
				// control reaches this point and to avoid calling this closure
				// from within the scope of the ServerCmd func.
				// reference:f9d302c2-df3f-4110-9529-94b0515c4a17
				// Follow-up: https://spothero.atlassian.net/browse/PMP-402
				grpcConfig.ServerRegistration = newGRPCService(c).RegisterAPIs
				grpcDone, err := grpcConfig.NewServer().Run()
				if err != nil {
					return err
				}
				wg.Add(1)
				go func() {
					<-grpcDone
					wg.Done()
				}()
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				if newHTTPService != nil {
					httpService := newHTTPService(c)
					httpConfig.RegisterHandlers = httpService.RegisterHandlers
				}
				httpConfig.NewServer().Run()
			}()

			wg.Wait()
			if c.PostShutdown != nil {
				if err := c.PostShutdown(ctx); err != nil {
					return err
				}
			}
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
