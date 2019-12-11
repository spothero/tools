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
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/opentracing/opentracing-go"
	"github.com/spothero/tools/log"
	"github.com/spothero/tools/service"
	"google.golang.org/grpc"
)

// These variables should be set during build with the Go link tool
// e.x.: when running go build, provide -ldflags="-X main.version=1.0.0"
var gitSHA = "not-set"
var version = "not-set"

type handler struct {
	environment string
}

// RegisterHandlers is a callback used to register HTTP endpoints to the default server
// NOTE: The HTTP server automatically registers /health and /metrics -- Have a look in your
// browser!
func (h handler) RegisterHandlers(router *mux.Router) {
	router.HandleFunc("/", h.helloWorld)
	router.HandleFunc("/best-language", bestLanguage)
}

// RegisterAPIs is a callback used to register GRPC endpoints to the default server.
// The handler is empty since we are not registering any GRPC APIs in this example.
func (h handler) RegisterAPIs(server *grpc.Server) {

}

// helloWorld simply writes "hello world" to the caller. It is ended for use as an HTTP callback.
func (h handler) helloWorld(w http.ResponseWriter, r *http.Request) {
	// NOTE: This is an example of an opentracing span
	span, _ := opentracing.StartSpanFromContext(r.Context(), "example-hello-world")
	span = span.SetTag("Key", "Value")
	defer span.Finish()
	log.Get(r.Context()).Info("hello logger")

	// NOTE: Here we write out some artisanal HTML. There are many other (better) ways to output data.
	fmt.Fprintf(w,
		`
<html>
Hello World. What's the <a href='/best-language'>best language?</a></br>
(I'm running in the %s environment)
</html>
		`,
		h.environment,
	)
}

// bestLanguage tells the caller what the best language is. It is inteded for use as an HTTP callback.
func bestLanguage(w http.ResponseWriter, r *http.Request) {
	// NOTE: This is an example of an opentracing span
	span, _ := opentracing.StartSpanFromContext(r.Context(), "example-hello-world")
	span = span.SetTag("best.language", "golang")
	span = span.SetTag("best.mascot", "gopher")
	defer span.Finish()
	log.Get(r.Context()).Info("go is the best language 😉")

	// NOTE: Here we write out some artisanal HTML. There are many other (better) ways to output data.
	fmt.Fprintf(w, "<html><a href='//golang.org/'>Go</a>, of course! \\ʕ◔ϖ◔ʔ/</br> Say <a href='/'>hello</a> again.</html>")
}

// This is the main entrypoint of the program. Here we create our root command and then execute it.
func main() {
	serverCmd := service.Config{
		Name:        "example_server",
		Version:     version,
		GitSHA:      gitSHA,
		Environment: "local",
	}
	if err := serverCmd.ServerCmd("", "", func(c service.Config) service.HTTPService {
		return handler{environment: c.Environment}
	}, func(c service.Config) service.GRPCService {
		return handler{environment: c.Environment}
	}).Execute(); err != nil {
		os.Exit(1)
	}
}
