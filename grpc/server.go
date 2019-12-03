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
	"net"
	"os"
	"os/signal"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/spothero/tools/log"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Config contains the configuration necessary for running a GRPC Server.
type Config struct {
	Name               string                         // Name of the HTTP Server
	Address            string                         // Address on which the server will be accessible
	Port               int                            // Port on which the server will be accessible
	TLSEnabled         bool                           // Whether or not traffic should be served via HTTPS
	TLSCrtPath         string                         // Location of TLS Certificate
	TLSKeyPath         string                         // Location of TLS Key
	ServerRegistration func(*grpc.Server)             // Callback for registering GRPC API Servers
	StreamInterceptors []grpc.StreamServerInterceptor // A list of global GRPC stream interceptor functions to be called. Order is honored left to right.
	UnaryInterceptors  []grpc.UnaryServerInterceptor  // A list of global GRPC unary interceptor functions to be called. Order is honored left to right.

	CancelSignals []os.Signal // OS Signals to be used to cancel running servers. Defaults to SIGINT/`os.Interrupt`.
}

// Server contains the configured GRPC server and related components
type Server struct {
	server        *grpc.Server
	listenAddress string
	cancelSignals []os.Signal // OS Signals to be used to cancel running servers. Defaults to SIGINT/`os.Interrupt`.
}

// NewDefaultConfig returns a default GRPC server config object. The caller must still supply the
// name of the server and a serverRegistration callback. The serverRegistration callback is a
// function that registers GRPC API servers with the provided GRPC server.
func NewDefaultConfig(name string, serverRegistration func(*grpc.Server)) Config {
	return Config{
		Name:               name,
		Address:            "127.0.0.1",
		Port:               9111,
		ServerRegistration: serverRegistration,
		StreamInterceptors: []grpc.StreamServerInterceptor{},
		UnaryInterceptors:  []grpc.UnaryServerInterceptor{},
		CancelSignals:      []os.Signal{os.Interrupt},
	}
}

func (c Config) NewServer() (Server, error) {
	server := grpc.NewServer(
		grpc.StreamInterceptor(
			grpc_middleware.ChainStreamServer(
				c.StreamInterceptors...,
			),
		),
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(
				c.UnaryInterceptors...,
			),
		),
	)
	if c.ServerRegistration == nil {
		return Server{}, fmt.Errorf("no server registration function provided")
	}
	c.ServerRegistration(server)
	return Server{
		server:        server,
		listenAddress: fmt.Sprintf("%s:%d", c.Address, c.Port),
	}, nil
}

func (s Server) Run() {
	ctx := context.Background()
	listener, err := net.Listen("tcp", s.listenAddress)
	if err != nil {
		log.Get(ctx).Error("error starting grpc server listener", zap.Error(err))
	}
	go func() {
		log.Get(ctx).Info(fmt.Sprintf("grpc server started on %s", s.listenAddress))
		if err := s.server.Serve(listener); err != nil {
			log.Get(ctx).Info("grpc server shutdown", zap.Error(err))
		}
	}()

	// Capture cancellation signal and gracefully shutdown goroutines
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, s.cancelSignals...)
	<-signals
	log.Get(ctx).Info("received interrupt, shutting down")
	s.server.GracefulStop()
}
