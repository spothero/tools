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

import "github.com/spf13/pflag"

// RegisterFlags registers HTTP flags with pflags
func (c *Config) RegisterFlags(flags *pflag.FlagSet) {
	flags.StringVar(&c.Name, "server-name", c.Name, "Server Name")
	flags.StringVarP(&c.Address, "address", "a", c.Address, "Address for server")
	flags.Uint16VarP(&c.Port, "port", "p", c.Port, "Port for server")
	flags.IntVar(&c.ReadTimeout, "read-timeout", c.ReadTimeout, "HTTP Server Read Timeout")
	flags.IntVar(&c.WriteTimeout, "write-timeout", c.WriteTimeout, "HTTP Server Write Timeout")
	flags.BoolVar(&c.HealthHandler, "health-handler", c.HealthHandler, "Enable /health endpoint")
	flags.BoolVar(&c.MetricsHandler, "metrics-handler", c.MetricsHandler, "Enable /metrics endpoints")
	flags.BoolVar(&c.PprofHandler, "pprof-handler", c.PprofHandler, "Enable /pprof/debug/* endpoints")
}
