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
	"fmt"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestNewDefaultConfig(t *testing.T) {
	config := NewDefaultConfig("test", nil)
	assert.Equal(t, Config{
		Name:               "test",
		Address:            "127.0.0.1",
		Port:               9111,
		TLSEnabled:         false,
		TLSCrtPath:         "",
		TLSKeyPath:         "",
		ServerRegistration: nil,
		StreamInterceptors: []grpc.StreamServerInterceptor{},
		UnaryInterceptors:  []grpc.UnaryServerInterceptor{},
		CancelSignals:      []os.Signal{os.Interrupt},
	}, config)
}

func TestNewServer(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectPanic bool
	}{
		{
			"no registration function results in a panic",
			Config{},
			true,
		},
		{
			"the server object is properly configured when a registration function is provided",
			Config{
				Name:               "test",
				Address:            "127.0.0.1",
				Port:               9111,
				ServerRegistration: func(*grpc.Server) {},
			},
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.expectPanic {
				assert.Panics(t, func() {
					server := test.config.NewServer()
					assert.NotNil(t, server.server)
				})
			} else {
				server := test.config.NewServer()
				assert.NotNil(t, server.server)
				assert.Equal(
					t,
					fmt.Sprintf("%s:%d", test.config.Address, test.config.Port),
					server.listenAddress,
				)
			}
		})
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		name      string
		server    Server
		expectErr bool
	}{
		{
			"an invalid tcp binding results in an error",
			Server{
				server:        grpc.NewServer(),
				listenAddress: "127.0.0.1:-1",
				cancelSignals: []os.Signal{syscall.SIGUSR1},
			},
			true,
		},
		{
			"a valid tcp binding does not result in an error",
			Server{
				server:        grpc.NewServer(),
				listenAddress: "127.0.0.1:60123",
				cancelSignals: []os.Signal{syscall.SIGUSR1},
			},
			false,
		},
		{
			"missing or in valid tls certificates result in an error",
			Server{
				server:        grpc.NewServer(),
				listenAddress: "127.0.0.1:60123",
				cancelSignals: []os.Signal{syscall.SIGUSR1},
				tlsEnabled:    true,
				tlsCrtPath:    "testdata/does-not-exist-crt.pem",
				tlsKeyPath:    "testdata/fake-key.pem",
			},
			true,
		},
		{
			"an invalid tcp binding with tls server results in an error",
			Server{
				server:        grpc.NewServer(),
				listenAddress: "127.0.0.1:-1",
				cancelSignals: []os.Signal{syscall.SIGUSR1},
				tlsEnabled:    true,
				tlsCrtPath:    "testdata/fake-crt.pem",
				tlsKeyPath:    "testdata/fake-key.pem",
			},
			true,
		},
		{
			"valid tls certificates are loaded",
			Server{
				server:        grpc.NewServer(),
				listenAddress: "127.0.0.1:60123",
				cancelSignals: []os.Signal{syscall.SIGUSR1},
				tlsEnabled:    true,
				tlsCrtPath:    "testdata/fake-crt.pem",
				tlsKeyPath:    "testdata/fake-key.pem",
			},
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			done, err := test.server.Run()
			if test.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NoError(t, syscall.Kill(syscall.Getpid(), syscall.SIGUSR1))
			<-done
		})
	}
}
