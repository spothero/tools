// Copyright 2022 SpotHero
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

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
)

// RegisterFlags registers GRPC flags with pflags
func (c *Config) RegisterFlags(flags *pflag.FlagSet) {
	flags.StringVar(&c.Name, "grpc-server-name", c.Name, "The name of the GRPC Server. This will be emitted in components such as logs and tracing.")
	flags.StringVar(&c.Address, "grpc-address", c.Address, "GRPC Address for server")
	flags.Uint16Var(&c.Port, "grpc-port", c.Port, "GRPC Port for server")
}

// RegisterFlags registers gRPC client configuration flags with pflags. Callers must specify a
// server name when calling this function. For example, if your service interacts with grpc service
// "foo" and "bar", you might register flags for both of these servers in your application. As such
// you must provide the names "foo" and "bar" and call this function twice. If you do, you will end
// up with a set of flags for each server in the format `--foo-grpc-server-address` and
// `--bar-grpc-server-address`.
//
// Failure to provide a server name will result in a panic.
func (cc *ClientConfig) RegisterFlags(flags *pflag.FlagSet, serverName string) {
	if len(serverName) == 0 {
		panic("no server name was specified when registering grpc client configuration flags")
	}
	lowerServerName := strings.ToLower(serverName)
	flags.StringVar(
		&cc.Address,
		fmt.Sprintf("%s-grpc-server-address", lowerServerName),
		cc.Address,
		fmt.Sprintf("gRPC Address for remote server `%s`", serverName),
	)
	flags.Uint16Var(
		&cc.Port,
		fmt.Sprintf("%s-grpc-server-port", lowerServerName),
		cc.Port,
		fmt.Sprintf("gRPC Port for remote server `%s`", serverName),
	)
	flags.BoolVar(
		&cc.PropagateAuthHeaders,
		fmt.Sprintf("%s-grpc-auth-propagate-headers", serverName),
		cc.PropagateAuthHeaders,
		fmt.Sprintf("If true, propagate headers to the gRPC remote server `%s`", serverName),
	)
	flags.BoolVar(
		&cc.RetryServerErrors,
		fmt.Sprintf("%s-grpc-retry-server-errors", serverName),
		cc.RetryServerErrors,
		fmt.Sprintf("If true, automatically retry on server errors from the gRPC remote server `%s`", serverName),
	)
}
