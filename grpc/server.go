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

package grpc

import (
	"context"
	"crypto/tls"
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
	Name               string                         // Name of the GRPC Server
	Address            string                         // Address on which the server will be accessible
	Port               uint16                         // Port on which the server will be accessible
	TLSEnabled         bool                           // Whether or not traffic should be served via HTTPS
	TLSCrtPath         string                         // Location of TLS Certificate
	TLSKeyPath         string                         // Location of TLS Key
	ServerRegistration func(*grpc.Server)             // Callback for registering GRPC API Servers
	StreamInterceptors []grpc.StreamServerInterceptor // A list of global GRPC stream interceptor functions to be called. Order is honored left to right.
	UnaryInterceptors  []grpc.UnaryServerInterceptor  // A list of global GRPC unary interceptor functions to be called. Order is honored left to right.
	CancelSignals      []os.Signal                    // OS Signals to be used to cancel running servers. Defaults to SIGINT/`os.Interrupt`.
}

// Server contains the configured GRPC server and related components
type Server struct {
	server        *grpc.Server // The the GRPC Server
	listenAddress string       // The address the server should bind to
	cancelSignals []os.Signal  // OS Signals to be used to cancel running servers. Defaults to SIGINT/`os.Interrupt`.
	tlsEnabled    bool         // Whether or not traffic should be served via HTTPS
	tlsCrtPath    string       // Location of TLS Certificate
	tlsKeyPath    string       // Location of TLS Key
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

// NewServer creates and returns a configured Server object given a GRPC configuration object.
func (c *Config) NewServer() Server {
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
		panic("no server registration function provided")
	}
	c.ServerRegistration(server)
	return Server{
		server:        server,
		listenAddress: fmt.Sprintf("%s:%d", c.Address, c.Port),
		tlsEnabled:    c.TLSEnabled,
		tlsCrtPath:    c.TLSCrtPath,
		tlsKeyPath:    c.TLSKeyPath,
		cancelSignals: c.CancelSignals,
	}
}

// Run starts the GRPC server. The function returns an error if the GRPC server cannot bind to its
// listen address. This function is non-blocking and will return immediately. If no error is returned
// the server is running. The returned channel will be closed after the server shuts down.
func (s Server) Run() (chan bool, error) {
	ctx := context.Background()
	var listener net.Listener
	var err error
	if s.tlsEnabled {
		cert, err := tls.LoadX509KeyPair(s.tlsCrtPath, s.tlsKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load tls x509 key pair: %w", err)
		}
		listener, err = tls.Listen(
			"tcp",
			s.listenAddress,
			&tls.Config{Certificates: []tls.Certificate{cert}},
		)
		if err != nil {
			return nil, fmt.Errorf("error starting tls grpc server listener: %w", err)
		}
	} else {
		listener, err = net.Listen("tcp", s.listenAddress)
		if err != nil {
			return nil, fmt.Errorf("error starting grpc server listener: %w", err)
		}
	}
	go func() {
		log.Get(ctx).Info(fmt.Sprintf("grpc server started on %s", s.listenAddress))
		err := s.server.Serve(listener)
		if err != nil {
			log.Get(ctx).Error("error encountered in grpc server", zap.Error(err))
		} else {
			log.Get(ctx).Info("grpc server shutdown", zap.Error(err))
		}
	}()

	// Capture cancellation signal and gracefully shutdown goroutines
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, s.cancelSignals...)
	done := make(chan bool)
	go func() {
		<-signals
		log.Get(ctx).Info("received interrupt, shutting down grpc server")
		s.server.GracefulStop()
		close(done)
	}()
	return done, nil
}
