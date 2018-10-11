package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spothero/core"
	"github.com/spothero/<%=appName%>/pkg/<%=appName%>"
)

// These variables should be set during build with the Go link tool
// e.x.: when running go build, provide -ldflags="-X main.version=1.0.0"
var gitSHA = "not-set"
var version = "not-set"
var appPackage = "github.com/spothero/<%=appName%>"

// newRootCmd is the entrypoint to your go program. Coupled with the `main` function at the bottom
// of this file, this is how you create and expose your CLI and Environment variables. We're
// building on the excellent open-source tool Cobra and Viper from spf13
func newRootCmd(args []string) *cobra.Command {
	config := &core.HTTPServerConfig{
		Name:       "server",
		Version:    version,
		AppPackage: appPackage,
		GitSHA:     gitSHA,
	}
	cmd := &cobra.Command{
		Use:              "server",
		Short:            "SpotHero Golang Microservice",
		Long:             `SpotHero Golang Microservice built on SpotHero Core.`,
		Version:          fmt.Sprintf("%s (%s)", version, gitSHA),
		PersistentPreRun: core.CobraBindEnvironmentVariables("server"),
		RunE: func(cmd *cobra.Command, args []string) error {
			config.RunHTTPServer(nil, nil, <%=appName%>.RegisterMuxes)
			return nil
		},
	}
	// Register default http server flags
	flags := cmd.Flags()
	config.CLIRegistration(flags, 8080)
	return cmd
}

// This is the main entrypoint of the program. Here we create our root command and then execute it.
func main() {
	cmd := newRootCmd(os.Args[1:])
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
