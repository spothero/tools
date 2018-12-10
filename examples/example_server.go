// Copyright 2018 SpotHero
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

package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/opentracing/opentracing-go"
	"github.com/spf13/cobra"
	"github.com/spothero/tools"
)

// These variables should be set during build with the Go link tool
// e.x.: when running go build, provide -ldflags="-X main.version=1.0.0"
var gitSHA = "not-set"
var version = "not-set"
var appPackage = "github.com/spothero/core"

// newRootCmd is the entrypoint to your go program. Coupled with the `main` function at the bottom
// of this file, this is how you create and expose your CLI and Environment variables. We're
// building on the excellent open-source tool Cobra and Viper from spf13
func newRootCmd(args []string) *cobra.Command {
	config := &tools.HTTPServerConfig{}
	cmd := &cobra.Command{
		Use:              "example_server",
		Short:            "SpotHero Example Golang Microservice",
		Long:             `The SpotHero Example Golang Microservice shows off the capabilities of our Core library`,
		Version:          fmt.Sprintf("%s (%s)", version, gitSHA),
		PersistentPreRun: tools.CobraBindEnvironmentVariables("example_server"),
		RunE: func(cmd *cobra.Command, args []string) error {
			config.RunHTTPServer(nil, nil, registerMuxes)
			return nil
		},
	}
	// Register default http server flags
	flags := cmd.Flags()
	config.RegisterViperFlags(flags, 8080, "example_server", version, appPackage, gitSHA)
	return cmd
}

// RegisterMuxes is a callback used to register HTTP endpoints to the default server
// NOTE: The HTTP server automatically registers /health and /metrics -- Have a look in your
// browser!
func registerMuxes(mux *http.ServeMux) {
	mux.HandleFunc("/", helloWorld)
	mux.HandleFunc("/best-language", bestLanguage)
}

// helloWorld simply writes "hello world" to the caller. It is ended for use as an HTTP callback.
func helloWorld(w http.ResponseWriter, r *http.Request) {
	// NOTE: This is an example of an opentracing span
	span, _ := opentracing.StartSpanFromContext(r.Context(), "example-hello-world")
	span = span.SetTag("Key", "Value")
	defer span.Finish()

	// NOTE: Here we write out some artisanal HTML. There are many other (better) ways to output data.
	fmt.Fprintf(w, "<html>Hello World. What's the <a href='/best-language'>best language?</a></html>")
}

// bestLanguage tells the caller what the best language is. It is inteded for use as an HTTP callback.
func bestLanguage(w http.ResponseWriter, r *http.Request) {
	// NOTE: This is an example of an opentracing span
	span, _ := opentracing.StartSpanFromContext(r.Context(), "example-hello-world")
	span = span.SetTag("best.language", "golang")
	span = span.SetTag("best.mascot", "gopher")
	defer span.Finish()

	// NOTE: Here we write out some artisanal HTML. There are many other (better) ways to output data.
	fmt.Fprintf(w, "<html><a href='//golang.org/'>Golang</a>, of course! \\ʕ◔ϖ◔ʔ/</br> Say <a href='/'>hello</a> again.</html>")
}

// This is the main entrypoint of the program. Here we create our root command and then execute it.
func main() {
	cmd := newRootCmd(os.Args[1:])
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
