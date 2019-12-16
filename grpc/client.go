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
	"fmt"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	grpcot "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpcprom "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"
)

// ClientConfig contains the configuration necessary for connecting to a GRPC Server.
type ClientConfig struct {
	Address string            // Address on which the server is accessible
	Port    uint16            // Port on which the server is accessible
	Options []grpc.DialOption // Additional server options
}

// NewDefaultClientConfig returns the default SpotHero GRPC Client Configuration
func NewDefaultClientConfig() ClientConfig {
	grpcprom.EnableClientHandlingTimeHistogram()
	grpcprom.EnableClientStreamReceiveTimeHistogram()
	grpcprom.EnableClientStreamSendTimeHistogram()
	grpcprom.EnableHandlingTimeHistogram()
	return ClientConfig{
		Address: "localhost",
		Port:    9111,
		Options: []grpc.DialOption{
			grpc.WithUnaryInterceptor(
				grpc_middleware.ChainUnaryClient(
					grpcot.UnaryClientInterceptor(),
					// TODO: Custom OT Tracer to add correlation_id
					// TODO: Custom Logger
					grpcprom.UnaryClientInterceptor,
					// TODO: Sentry
					// TODO: Auth header passer?
				),
			),
			grpc.WithStreamInterceptor(
				grpc_middleware.ChainStreamClient(
					grpcot.StreamClientInterceptor(),
					// TODO: Custom OT Tracer to add correlation_id
					// TODO: Custom Logger
					grpcprom.StreamClientInterceptor,
					// TODO: Sentry
					// TODO: Auth header passer?
				),
			),
			grpc.WithInsecure(),
		},
	}
}

// GetConn dials and returns a GRPC connection. It is the responsibility of the caller to make sure
// they call `conn.Close()` through a defer statement or otherwise.
func (cc ClientConfig) GetConn() (*grpc.ClientConn, error) {
	return grpc.Dial(
		fmt.Sprintf("%v:%v", cc.Address, cc.Port),
		cc.Options...,
	)
}
