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
	"os"
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

}

func TestRun(t *testing.T) {

}
