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
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestNewDefaultConfig(t *testing.T) {
	config := NewDefaultConfig("test")
	// Just check that middleware is set
	assert.NotNil(t, config.Middleware)
	// Set middleware nil because it contains singletons -- so we dont want to recreate them for
	// the below equality assertion.
	config.Middleware = nil

	// Ensure that the remaining fields are correctly configured
	assert.Equal(t, Config{
		Name:           "test",
		Address:        "0.0.0.0",
		Port:           8080,
		ReadTimeout:    5,
		WriteTimeout:   30,
		HealthHandler:  true,
		MetricsHandler: true,
		PprofHandler:   true,
		CancelSignals:  []os.Signal{os.Interrupt},
	}, config)
}

func TestNewServer(t *testing.T) {
	registrationCalled := false
	mockRegistration := func(r *mux.Router) {
		registrationCalled = true
	}
	mockPreStart := func(ctx context.Context, router *mux.Router, server *http.Server) {}
	mockPostShutdown := func(ctx context.Context) {}

	config := Config{
		Address:          "127.0.0.1",
		Port:             9090,
		HealthHandler:    true,
		MetricsHandler:   true,
		PprofHandler:     true,
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
		"/debug/pprof":         true,
		"/debug/pprof/cmdline": true,
		"/debug/pprof/profile": true,
		"/debug/pprof/symbol":  true,
		"/debug/pprof/trace":   true,
		"/metrics":             true,
	}
	server.router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		routeName, err := route.GetPathTemplate()
		assert.NoError(t, err)
		if _, ok := expectedRoutes[routeName]; !ok {
			assert.Fail(t, "Missing Route", routeName)
		}
		delete(expectedRoutes, routeName)
		return nil
	})
	assert.Lenf(t, expectedRoutes, 0, "some expected routes were not registered: %v", expectedRoutes)
}

//
// TODO: WIP NOT COMPLETED YET
//
func TestRun(t *testing.T) {
	mockPreStart := func(ctx context.Context, router *mux.Router, server *http.Server) {}
	mockPostShutdown := func(ctx context.Context) {}
	router := mux.NewRouter()
	server := httptest.NewServer(router)
	s := Server{
		httpServer:    server,
		router:        router,
		preStart:      mockPreStart,
		postShutdown:  mockPostShutdown,
		cancelSignals: []os.Signal{TODO},
	}
	// TODO:
	// TODO: Mock signal?
}
