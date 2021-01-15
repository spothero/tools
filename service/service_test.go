// Copyright 2021 SpotHero
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
	"context"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

type mockHTTPService struct{}

func (ms mockHTTPService) RegisterHandlers(_ *mux.Router) {}

type mockGRPCService struct{}

func (ms mockGRPCService) RegisterAPIs(*grpc.Server) {}

type prePost struct {
	mock.Mock
}

func (p *prePost) preStart(ctx context.Context) (context.Context, error) {
	returns := p.Called(ctx)
	return returns.Get(0).(context.Context), returns.Error(1)
}

func (p *prePost) postShutdown(ctx context.Context) error {
	return p.Called(ctx).Error(0)
}

func TestDefaultServer(t *testing.T) {
	mockPrePost := prePost{}
	c := Config{
		Name:          "test",
		Environment:   "test",
		Registry:      prometheus.NewRegistry(),
		Version:       "0.1.0",
		GitSHA:        "abc123",
		CancelSignals: []os.Signal{syscall.SIGUSR1},
		PreStart:      mockPrePost.preStart,
		PostShutdown:  mockPrePost.postShutdown,
	}
	ctx := context.Background()
	cmd := c.ServerCmd(
		ctx,
		"short",
		"long",
		func(Config) HTTPService { return mockHTTPService{} },
		func(Config) GRPCService { return mockGRPCService{} },
	)
	assert.NotNil(t, cmd)
	assert.NotZero(t, cmd.Use)
	assert.NotZero(t, cmd.Short)
	assert.NotZero(t, cmd.Long)
	assert.True(t, strings.Contains(cmd.Version, c.Version))
	assert.True(t, strings.Contains(cmd.Version, c.GitSHA))
	assert.NotNil(t, cmd.PersistentPreRun)
	assert.NotNil(t, cmd.RunE)
	assert.True(t, cmd.Flags().HasFlags())
	mockPrePost.On("preStart", ctx).Return(ctx, nil)
	mockPrePost.On("postShutdown", ctx).Return(nil)
	defer mockPrePost.AssertExpectations(t)

	done := make(chan bool)
	go func() {
		err := cmd.Execute()
		assert.NoError(t, err)
		close(done)
	}()
	timer := time.NewTimer(100 * time.Millisecond)
	<-timer.C
	assert.NoError(t, syscall.Kill(syscall.Getpid(), syscall.SIGUSR1))
	<-done
}
