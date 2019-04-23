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

package http

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spothero/tools/log"
	"go.uber.org/zap"
)

// Config contains the configuration necessary for running an HTTP/HTTPS Server.
type Config struct {
	Name             string                                                             // Name of the HTTP Server
	Address          string                                                             // Address on which the server will be accessible
	Port             int                                                                // Port on which the server will be accessible
	TLSEnabled       bool                                                               // Whether or not traffic should be served via HTTPS
	TLSCrtPath       string                                                             // Location of TLS Certificate
	TLSKeyPath       string                                                             // Location of TLS Key
	ReadTimeout      int                                                                // The Read Timeout for Server Requests
	WriteTimeout     int                                                                // The Write Timeout for Server Requests
	HealthHandler    bool                                                               // If true, register a healthcheck endpoint at /health
	MetricsHandler   bool                                                               // If true, register a Prometheus metrics endpoint at /metrics
	PprofHandler     bool                                                               // If true, register pprof endpoints under /debug/pprof
	PreStart         func(ctx context.Context, router *mux.Router, server *http.Server) // A function to be called before starting the web server
	PostShutdown     func(ctx context.Context)                                          // A function to be called before stopping the web server
	RegisterHandlers func(*mux.Router)                                                  // Handler registration callback function. Register your routes in this function.
	Middleware       Middleware                                                         // A list of global middleware functions to be called. Order is honored.
	CancelSignals    []os.Signal                                                        // OS Signals to be used to cancel running servers. Defaults to SIGINT/`os.Interrupt`.
}

// Server contains unexported fields and is used to start and manage the Server.
type Server struct {
	httpServer    *http.Server
	router        *mux.Router
	preStart      func(ctx context.Context, router *mux.Router, server *http.Server)
	postShutdown  func(ctx context.Context)
	cancelSignals []os.Signal
}

// NewDefaultConfig returns a standard configuration given a server name. It is recommended to
// invoke this function for a Config before providing further customization.
func NewDefaultConfig(name string) Config {
	return Config{
		Name:           name,
		Address:        "0.0.0.0",
		Port:           8080,
		ReadTimeout:    5,
		WriteTimeout:   30,
		HealthHandler:  true,
		MetricsHandler: true,
		PprofHandler:   true,
		CancelSignals:  []os.Signal{os.Interrupt},
	}
}

// NewServer uses the given http Config to create and return a server ready to be run.
func (c Config) NewServer() Server {
	router := mux.NewRouter()
	if c.HealthHandler {
		router.HandleFunc("/health", healthHandler)
	}
	if c.PprofHandler {
		router.HandleFunc("/debug/pprof", pprof.Index)
		router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		router.HandleFunc("/debug/pprof/profile", pprof.Profile)
		router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		router.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}
	if c.MetricsHandler {
		router.Handle("/metrics", promhttp.Handler())
	}
	if c.RegisterHandlers != nil {
		c.RegisterHandlers(router)
	}
	return Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", c.Address, c.Port),
			Handler:      c.Middleware.handler(router),
			ReadTimeout:  time.Duration(c.ReadTimeout) * time.Second,
			WriteTimeout: time.Duration(c.WriteTimeout) * time.Second,
		},
		router:        router,
		preStart:      c.PreStart,
		postShutdown:  c.PostShutdown,
		cancelSignals: c.CancelSignals,
	}
}

// Run starts the web server, calling any provided preStart hooks and registering the provided
// muxes. The server runs until a cancellation signal is sent to exit. At that point, the server is
// stopped and any postShutdown hooks are called.
//
// Note that cancelSignals defines the os.Signals that should cause the server to exit and shut
// down. If no cancelSignals are provided, this defaults to os.Interrupt. Note that if you override
// this value and still wish to handle os.Interrupt you _must_ additionally include that value.
func (s Server) Run() {
	// Setup a context to send cancellation signals to goroutines
	ctx, cancel := context.WithCancel(context.Background())

	// Call any existing pre-start callback
	if s.preStart != nil {
		s.preStart(ctx, s.router, s.httpServer)
	}

	// Start all configured servers
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		go func() {
			log.Get(ctx).Info(fmt.Sprintf("HTTP server started on %s", s.httpServer.Addr))
			if err := s.httpServer.ListenAndServe(); err != nil {
				log.Get(ctx).Info("HTTP server shutdown", zap.Error(err))
			}
		}()
		<-ctx.Done()
	}()

	// Capture cancellation signal and deliver to running goroutines
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, s.cancelSignals...)
	<-signals
	log.Get(ctx).Info("Received interrupt, shutting down")
	cancel()

	// Wait for servers to finish exiting and initiate shutdown
	wg.Wait()
	shutdown, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := s.httpServer.Shutdown(shutdown); err != nil {
		log.Get(shutdown).Error("error shutting down http server", zap.Error(err))
	}

	// Call any existing post-shutdown callback
	if s.postShutdown != nil {
		s.postShutdown(shutdown)
	}
}
