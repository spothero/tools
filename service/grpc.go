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
	"strings"

	grpcauth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpcot "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpcprom "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/spf13/cobra"
	"github.com/spothero/tools/cli"
	shGRPC "github.com/spothero/tools/grpc"
	"github.com/spothero/tools/jose"
	"github.com/spothero/tools/log"
	"github.com/spothero/tools/sentry"
	"github.com/spothero/tools/tracing"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
)

// GRPCConfig contains required configuration for starting a GRPC service
type GRPCConfig struct {
	Config
}

// GRPCService implementors register GRPC APIs with the GRPC server
type GRPCService interface {
	ServerRegistration(*grpc.Server)
}

// This method takes a function, newService, that instantiates the GRPCService by consuming
// the GRPCConfig object after all values are populated from the CLI and/or environment
// variables so that values configured by this package are accessible by newService.
//
// Note that this function returns the Default GRPC server for use at SpotHero. Consumers of the
// tools libraries are free to define their own server entrypoints if desired. This function is
// provided as a convenience function that should satisfy most use cases.
//
// Note that Version and GitSHA *must be specified* before calling this function.
func (gc GRPCConfig) ServerCmd(
	shortDescription, longDescription string,
	newService func(GRPCConfig) GRPCService,
) *cobra.Command {
	// GRPC Config
	config := shGRPC.NewDefaultConfig(gc.Name, newService(gc).ServerRegistration)
	if len(gc.CancelSignals) > 0 {
		config.CancelSignals = gc.CancelSignals
	}
	// Logging Config
	lc := &log.Config{
		UseDevelopmentLogger: true,
		Fields: map[string]interface{}{
			"version": gc.Version,
			"git_sha": gc.GitSHA[len(gc.GitSHA)-6:], // Log only the last 6 digits of the Git SHA
		},
		Cores: []zapcore.Core{&sentry.Core{}},
	}
	// Sentry Config
	sc := sentry.Config{
		Environment: gc.Environment,
		AppVersion:  gc.Version,
	}
	// Tracing Config
	tc := tracing.Config{ServiceName: gc.Name}
	// Jose Config
	jc := jose.Config{ClaimGenerators: []jose.ClaimGenerator{jose.CognitoGenerator{}}}
	cmd := &cobra.Command{
		Use:              gc.Name,
		Short:            shortDescription,
		Long:             longDescription,
		Version:          fmt.Sprintf("%s (%s)", gc.Version, gc.GitSHA),
		PersistentPreRun: cli.CobraBindEnvironmentVariables(strings.Replace(gc.Name, "-", "_", -1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := gc.CheckFlags(); err != nil {
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
			config.UnaryInterceptors = []grpc.UnaryServerInterceptor{
				grpcot.UnaryServerInterceptor(),
				tracing.UnaryServerInterceptor,
				log.UnaryServerInterceptor,
				grpcprom.UnaryServerInterceptor,
			}
			config.StreamInterceptors = []grpc.StreamServerInterceptor{
				grpcot.StreamServerInterceptor(),
				tracing.StreamServerInterceptor,
				log.StreamServerInterceptor,
				grpcprom.StreamServerInterceptor,
			}

			// If the user has requested JOSE Auth, add JOSE Auth interceptors
			if jc.AuthRequired {
				jh, err := jc.NewJOSE()
				if err != nil {
					return err
				}
				joseInterceptorFunc := jose.GetContextAuth(jh, jc.AuthRequired)
				config.UnaryInterceptors = append(
					config.UnaryInterceptors,
					grpcauth.UnaryServerInterceptor(joseInterceptorFunc),
				)
				config.StreamInterceptors = append(
					config.StreamInterceptors,
					grpcauth.StreamServerInterceptor(joseInterceptorFunc),
				)
			}
			if err := config.NewServer().Run(); err != nil {
				return fmt.Errorf("failed to run the grpc server: %x", err)
			}
			return nil
		},
	}
	// Register Cobra/Viper CLI Flags
	flags := cmd.Flags()
	gc.RegisterFlags(flags)
	config.RegisterFlags(flags)
	lc.RegisterFlags(flags)
	sc.RegisterFlags(flags)
	tc.RegisterFlags(flags)
	jc.RegisterFlags(flags)
	return cmd
}
