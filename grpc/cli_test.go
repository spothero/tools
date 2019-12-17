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

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestRegisterFlags(t *testing.T) {
	flags := pflag.NewFlagSet("pflags", pflag.PanicOnError)
	c := NewDefaultConfig("test", func(*grpc.Server) {})
	c.RegisterFlags(flags)
	err := flags.Parse(nil)
	assert.NoError(t, err)

	sn, err := flags.GetString("grpc-server-name")
	assert.NoError(t, err)
	assert.Equal(t, c.Name, sn)

	ad, err := flags.GetString("grpc-address")
	assert.NoError(t, err)
	assert.Equal(t, c.Address, ad)

	p, err := flags.GetUint16("grpc-port")
	assert.NoError(t, err)
	assert.Equal(t, c.Port, p)
}

func TestClientRegisterFlags(t *testing.T) {
	tests := []struct {
		name        string
		serverName  string
		expectPanic bool
	}{
		{
			"no server name results in a panic",
			"",
			true,
		},
		{
			"the server name is rendered into the flags",
			"server",
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			flags := pflag.NewFlagSet("pflags", pflag.PanicOnError)
			cc := NewDefaultClientConfig(context.Background())

			if test.expectPanic {
				assert.Panics(t, func() {
					cc.RegisterFlags(flags, test.serverName)
				})
			} else {
				cc.RegisterFlags(flags, test.serverName)
				err := flags.Parse(nil)
				assert.NoError(t, err)

				ad, err := flags.GetString(fmt.Sprintf("%s-grpc-server-address", strings.ToLower(test.serverName)))
				assert.NoError(t, err)
				assert.Equal(t, cc.Address, ad)

				p, err := flags.GetUint16(fmt.Sprintf("%s-grpc-server-port", strings.ToLower(test.serverName)))
				assert.NoError(t, err)
				assert.Equal(t, cc.Port, p)

				ph, err := flags.GetBool(fmt.Sprintf("%s-grpc-auth-propagate-headers", strings.ToLower(test.serverName)))
				assert.NoError(t, err)
				assert.Equal(t, cc.PropagateAuthHeaders, ph)

				rse, err := flags.GetBool(fmt.Sprintf("%s-grpc-retry-server-errors", strings.ToLower(test.serverName)))
				assert.NoError(t, err)
				assert.Equal(t, cc.RetryServerErrors, rse)
			}
		})
	}
}
