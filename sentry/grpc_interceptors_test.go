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

package sentry

import (
	"context"
	"testing"

	"github.com/getsentry/sentry-go"
	grpcmock "github.com/spothero/tools/grpc/mock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestGrpcErrToSentry(t *testing.T) {
	ctx := sentry.SetHubOnContext(context.Background(), sentry.CurrentHub().Clone())
	assert.Panics(t, func() {
		defer grpcErrToSentry(ctx)
		panic("test")
	})
}

func TestConfigureHub(t *testing.T) {
	ctx := configureHub(context.Background(), "method.service")
	assert.True(t, sentry.HasHubOnContext(ctx))
	// there's no way to get the contents of scope, so just make sure the hub isn't empty to verify the
	// trace id is on it
	hub := sentry.GetHubFromContext(ctx)
	clone := hub.Clone()
	clone.Scope().Clear()
	assert.NotEqual(t, clone.Scope(), hub.Scope())
}

func TestUnaryServerInterceptor(t *testing.T) {
	info := &grpc.UnaryServerInfo{}
	mockHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return struct{}{}, nil
	}
	assert.NotPanics(t, func() {
		resp, err := UnaryServerInterceptor(context.Background(), nil, info, mockHandler)
		assert.NoError(t, err)
		assert.Equal(t, struct{}{}, resp)
	})
}

func TestStreamServerInterceptor(t *testing.T) {
	info := &grpc.StreamServerInfo{}
	mockHandler := func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}
	mockStream := &grpcmock.MockServerStream{}
	mockStream.On("Context").Return(context.Background())
	assert.NotPanics(t, func() {
		assert.NoError(t, StreamServerInterceptor(nil, mockStream, info, mockHandler))
	})
}
