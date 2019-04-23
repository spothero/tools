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
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/pflag"
)

// Config defines service level configuration for HTTP servers
type Config struct {
	Name        string               // Name of the application
	Version     string               // Semantic Version of the application
	GitSHA      string               // GitSHA of the application when compiled
	Environment string               // Environment where the server is running
	Package     string               // The name of this go package, eg `github.com/spothero/myservice`
	Registry    *prometheus.Registry // The Prometheus Registry to use. If nil, the global registry is used by default.
}

// RegisterFlags registers Service flags with pflags
func (c *Config) RegisterFlags(flags *pflag.FlagSet) {
	flags.StringVarP(&c.Name, "name", "n", c.Name, "Name of the application")
	flags.StringVarP(&c.Version, "version", "v", c.Version, "Semantic Version of the application")
	flags.StringVarP(&c.GitSHA, "git-sha", "g", c.GitSHA, "Git SHA of the application")
	flags.StringVarP(&c.Environment, "environment", "e", c.Environment, "Environment where the application is running")
	flags.StringVarP(&c.Package, "package", "k", c.Package, "Go package, eg `github.com/spothero/<myapp>`")
}

// CheckFlags ensures that the Service Config contains all necessary configuration for use at
// runtime. An error is returned describing any missing fields.
func (c Config) CheckFlags() error {
	errors := make([]string, 0)
	if c.Name == "" {
		errors = append(errors, "no server name provided")
	}
	if c.Version == "" {
		errors = append(errors, "no version provided")
	}
	if c.GitSHA == "" {
		errors = append(errors, "not git sha provided")
	}
	if c.Environment == "" {
		errors = append(errors, "no environment specified")
	}
	if c.Package == "" {
		errors = append(errors, "no package specified")
	}
	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, ", "))
	}
	return nil
}
