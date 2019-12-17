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

package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	grpczap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpcretry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	grpcot "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpcprom "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/spothero/tools/jose"
	"github.com/spothero/tools/log"
	"github.com/spothero/tools/tracing"
	"google.golang.org/grpc"
)

// ClientConfig contains the configuration necessary for connecting to a gRPC Server.
type ClientConfig struct {
	Address              string                         // Address on which the server is accessible
	Port                 uint16                         // Port on which the server is accessible
	PropagateAuthHeaders bool                           // If true propagate any authorization header to the server
	RetryServerErrors    bool                           // If true, the client will automatically retry on server errors
	UnaryInterceptors    []grpc.UnaryClientInterceptor  // Client unary interceptors to apply
	StreamInterceptors   []grpc.StreamClientInterceptor // Client stream interceptors to apply
	Options              []grpc.DialOption              // Additional server options
}

// NewDefaultClientConfig returns the default SpotHero gRPC Client Configuration
func NewDefaultClientConfig(ctx context.Context) ClientConfig {
	grpcprom.EnableClientHandlingTimeHistogram()
	grpcprom.EnableClientStreamReceiveTimeHistogram()
	grpcprom.EnableClientStreamSendTimeHistogram()
	grpcprom.EnableHandlingTimeHistogram()
	return ClientConfig{
		Address:              "localhost",
		Port:                 9111,
		PropagateAuthHeaders: false,
		RetryServerErrors:    false,
		UnaryInterceptors: []grpc.UnaryClientInterceptor{
			grpcot.UnaryClientInterceptor(),
			tracing.UnaryClientInterceptor,
			grpczap.UnaryClientInterceptor(log.Get(ctx)),
			grpcprom.UnaryClientInterceptor,
		},
		StreamInterceptors: []grpc.StreamClientInterceptor{
			grpcot.StreamClientInterceptor(),
			tracing.StreamClientInterceptor,
			grpczap.StreamClientInterceptor(log.Get(ctx)),
			grpcprom.StreamClientInterceptor,
		},
		Options: []grpc.DialOption{grpc.WithInsecure()},
	}
}

// GetConn dials and returns a gRPC connection. It is the responsibility of the caller to make sure
// they call `conn.Close()` through a defer statement or otherwise.
func (cc ClientConfig) GetConn() (*grpc.ClientConn, error) {
	if cc.PropagateAuthHeaders {
		cc.UnaryInterceptors = append(cc.UnaryInterceptors, jose.UnaryClientInterceptor)
		cc.StreamInterceptors = append(cc.StreamInterceptors, jose.StreamClientInterceptor)
	}
	if cc.RetryServerErrors {
		opts := []grpcretry.CallOption{
			// Try with exponential backoff, starting with 100ms and a 10% jitter offset
			grpcretry.WithBackoff(grpcretry.BackoffExponentialWithJitter(100*time.Millisecond, .10)),
			grpcretry.WithCodes(grpcretry.DefaultRetriableCodes...),
			grpcretry.WithMax(5), // Try with exponential backoff 5 times by default
		}
		cc.UnaryInterceptors = append(cc.UnaryInterceptors, grpcretry.UnaryClientInterceptor(opts...))
		cc.StreamInterceptors = append(cc.StreamInterceptors, grpcretry.StreamClientInterceptor(opts...))
	}
	return grpc.Dial(
		fmt.Sprintf("%v:%v", cc.Address, cc.Port),
		append(
			cc.Options,
			grpc.WithUnaryInterceptor(
				grpc_middleware.ChainUnaryClient(
					cc.UnaryInterceptors...,
				),
			),
			grpc.WithStreamInterceptor(
				grpc_middleware.ChainStreamClient(
					cc.StreamInterceptors...,
				),
			),
		)...,
	)
}
