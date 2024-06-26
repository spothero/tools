// Copyright 2023 SpotHero
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
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestNewDefaultConfig(t *testing.T) {
	config := NewDefaultConfig("test")
	// Ensure that the remaining fields are correctly configured
	assert.Equal(t, Config{
		Name:            "test",
		Address:         "127.0.0.1",
		Port:            8080,
		ReadTimeout:     5,
		WriteTimeout:    60,
		HealthHandler:   true,
		MetricsHandler:  true,
		PprofHandler:    false,
		DynamicLogLevel: true,
		Middleware:      nil,
		CancelSignals:   []os.Signal{os.Interrupt},
	}, config)
}

func TestNewServer(t *testing.T) {
	registrationCalled := false
	mockRegistration := func(_ *mux.Router) {
		registrationCalled = true
	}
	mockPreStart := func(_ context.Context, _ *mux.Router, _ *http.Server) {}
	mockPostShutdown := func(_ context.Context) {}

	config := Config{
		Address:          "127.0.0.1",
		Port:             9090,
		HealthHandler:    true,
		MetricsHandler:   true,
		PprofHandler:     true,
		DynamicLogLevel:  true,
		RegisterHandlers: mockRegistration,
		PreStart:         mockPreStart,
		PostShutdown:     mockPostShutdown,
	}
	server := config.NewServer()
	assert.Equal(t, "127.0.0.1:9090", server.httpServer.Addr)
	assert.True(t, registrationCalled)
	assert.NotNil(t, server.preStart)
	assert.NotNil(t, server.postShutdown)

	// walk routes to ensure default routes are registered
	expectedRoutes := map[string]bool{
		"/health":              true,
		"/debug/":              true,
		"/debug/pprof/":        true,
		"/debug/pprof/cmdline": true,
		"/debug/pprof/profile": true,
		"/debug/pprof/symbol":  true,
		"/debug/pprof/trace":   true,
		"/metrics":             true,
		"/loglevel":            true,
	}
	err := server.router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		routeName, err := route.GetPathTemplate()
		assert.NoError(t, err)
		if _, ok := expectedRoutes[routeName]; !ok {
			assert.Fail(t, "Missing Route", routeName)
		}
		delete(expectedRoutes, routeName)
		return nil
	})
	assert.NoError(t, err)
	assert.Lenf(t, expectedRoutes, 0, "some expected routes were not registered: %v", expectedRoutes)
}

func TestRun(t *testing.T) {
	preStartCalled := false
	mockPreStart := func(_ context.Context, _ *mux.Router, _ *http.Server) {
		preStartCalled = true
	}
	postShutdownCalled := false
	mockPostShutdown := func(_ context.Context) {
		postShutdownCalled = true
	}
	router := mux.NewRouter()
	tests := []struct {
		name   string
		server Server
	}{
		{
			"an invalid tcp binding results in an error",
			Server{
				httpServer: &http.Server{
					Addr:    "127.0.0.1:-1",
					Handler: router,
				},
				router:        router,
				preStart:      mockPreStart,
				postShutdown:  mockPostShutdown,
				cancelSignals: []os.Signal{syscall.SIGUSR1},
			},
		},
		{
			"http servers bind with valid settings",
			Server{
				httpServer: &http.Server{
					Addr:    "127.0.0.1:60987",
					Handler: router,
				},
				router:        router,
				preStart:      mockPreStart,
				postShutdown:  mockPostShutdown,
				cancelSignals: []os.Signal{syscall.SIGUSR1},
			},
		},
		{
			"https/tls servers bind with valid settings",
			Server{
				httpServer: &http.Server{
					Addr:    "127.0.0.1:60987",
					Handler: router,
				},
				router:        router,
				preStart:      mockPreStart,
				postShutdown:  mockPostShutdown,
				cancelSignals: []os.Signal{syscall.SIGUSR1},
				tlsEnabled:    true,
				tlsCrtPath:    "../testdata/fake-crt.pem",
				tlsKeyPath:    "../testdata/fake-key.pem",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			timer := time.NewTimer(20 * time.Millisecond)
			go func() {
				<-timer.C
				assert.NoError(t, syscall.Kill(syscall.Getpid(), syscall.SIGUSR1))
			}()
			test.server.Run()
			assert.True(t, preStartCalled)
			assert.True(t, postShutdownCalled)
		})
	}
}
