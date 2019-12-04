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
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

type mockGRPCService struct{}

func (ms mockGRPCService) ServerRegistration(*grpc.Server) {}

func TestDefaultGRPCServer(t *testing.T) {
	c := GRPCConfig{
		Config: Config{
			Name:     "test",
			Registry: prometheus.NewRegistry(),
			Version:  "0.1.0",
			GitSHA:   "abc123",
		},
	}
	cmd := c.ServerCmd("short", "long", func(GRPCConfig) GRPCService { return mockGRPCService{} })
	assert.NotNil(t, cmd)
	assert.NotZero(t, cmd.Use)
	assert.NotZero(t, cmd.Short)
	assert.NotZero(t, cmd.Long)
	assert.True(t, strings.Contains(cmd.Version, c.Version))
	assert.True(t, strings.Contains(cmd.Version, c.GitSHA))
	assert.NotNil(t, cmd.PersistentPreRun)
	assert.NotNil(t, cmd.RunE)
	assert.True(t, cmd.Flags().HasFlags())
}
