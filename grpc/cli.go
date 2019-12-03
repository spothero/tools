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

package grpc

import "github.com/spf13/pflag"

// RegisterFlags registers GRPC flags with pflags
func (c *Config) RegisterFlags(flags *pflag.FlagSet) {
	flags.StringVar(&c.Name, "grpc-server-name", c.Name, "GRPC Server Name")
	flags.StringVarP(&c.Address, "grpc-address", "a", c.Address, "GRPC Address for server")
	flags.IntVar(&c.Port, "grpc-port", c.Port, "GRPC Port for server")
}
