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

package service_test

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/spothero/tools/service"
	"google.golang.org/grpc"
)

type handler struct {
	environment string
}

// RegisterHandlers is a callback used to register HTTP endpoints to the default server
// NOTE: The HTTP server automatically registers /health, /debug, and /metrics -- Have
// a look in your browser!
func (h handler) RegisterHandlers(router *mux.Router) {
	router.HandleFunc("/", h.helloWorld)
}

// RegisterAPIs is a callback used to register GRPC endpoints to the default server.
// The handler is empty since we are not registering any GRPC APIs in this example.
func (h handler) RegisterAPIs(server *grpc.Server) {
	// Here you would register any GRPC APIs with the GRPC server. In this example we do not have
	// any GRPC endpoints to register.
}

// helloWorld simply writes "hello world" to the caller. It is intended for use as an HTTP callback.
func (h handler) helloWorld(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World")
}

// This is the main entrypoint of the program. Here we create our root command and then execute it.
func Example() {
	// Configure the service with default settings. These settings may be overridden via CLI flags
	// or environment variables from the cobra command
	config := service.Config{
		Name:        "<your-application-name>",
		Version:     "<semantic-version>",
		GitSHA:      "<git-sha>",
		Environment: "<environment>",
	}

	// Using the config, create the cobra command by passing in an HTTP and GRPC registration
	// callback function
	_ = config.ServerCmd(context.Background(), "", "", func(c service.Config) service.HTTPService {
		return handler{environment: c.Environment}
	}, func(c service.Config) service.GRPCService {
		return handler{environment: c.Environment}
	})

	// In the previous function call, we discard the cobra command that is returned. If we had
	// called it `cobraCmd`, we could then run the server by simply calling `Execute()`
	// if err := cobraCmd.Execute(); err != nil {
	//	os.Exit(1)
	// }
}
