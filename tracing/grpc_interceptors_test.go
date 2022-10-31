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

package tracing

import (
	"context"
	grpcmock "github.com/spothero/tools/grpc/mock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"testing"
)

func TestUnaryServerInterceptor(t *testing.T) {
	shutdown, _ := GetTracerProvider()
	ctx := context.Background()
	defer func() {
		if err := shutdown(ctx); err != nil {
			assert.Error(t, err)
		}
	}()

	_, spanCtx := StartSpanFromContext(ctx, "test")
	info := &grpc.UnaryServerInfo{}
	mockHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		correlationId, ok := ctx.Value(CorrelationIDCtxKey).(string)
		assert.Equal(t, true, ok)
		assert.NotNil(t, correlationId)
		assert.NotEqual(t, "", correlationId)
		return struct{}{}, nil
	}
	resp, err := UnaryServerInterceptor(spanCtx, nil, info, mockHandler)
	assert.NoError(t, err)
	assert.Equal(t, struct{}{}, resp)
}

func TestStreamServerInterceptor(t *testing.T) {
	shutdown, _ := GetTracerProvider()
	ctx := context.Background()
	defer func() {
		if err := shutdown(ctx); err != nil {
			assert.Error(t, err)
		}
	}()

	_, spanCtx := StartSpanFromContext(context.Background(), "test")
	info := &grpc.StreamServerInfo{}
	mockHandler := func(srv interface{}, stream grpc.ServerStream) error {
		correlationId, ok := stream.Context().Value(CorrelationIDCtxKey).(string)
		assert.Equal(t, true, ok)
		assert.NotNil(t, correlationId)
		assert.NotEqual(t, "", correlationId)
		return nil
	}
	mockStream := &grpcmock.MockServerStream{}
	mockStream.On("Context").Return(spanCtx)
	err := StreamServerInterceptor(nil, mockStream, info, mockHandler)
	assert.NoError(t, err)
}

func TestUnaryClientInterceptor(t *testing.T) {
	shutdown, _ := GetTracerProvider()
	ctx := context.Background()
	defer func() {
		if err := shutdown(ctx); err != nil {
			assert.Error(t, err)
		}
	}()

	mockInvoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		correlationId, ok := ctx.Value(CorrelationIDCtxKey).(string)
		assert.Equal(t, true, ok)
		assert.NotNil(t, correlationId)
		assert.NotEqual(t, "", correlationId)
		return nil
	}

	_, spanCtx := StartSpanFromContext(context.Background(), "test")
	assert.NoError(
		t,
		UnaryClientInterceptor(
			spanCtx,
			"method",
			struct{}{}, struct{}{},
			&grpc.ClientConn{},
			mockInvoker,
		),
	)
}

func TestStreamClientInterceptor(t *testing.T) {
	shutdown, _ := GetTracerProvider()
	ctx := context.Background()
	defer func() {
		if err := shutdown(ctx); err != nil {
			assert.Error(t, err)
		}
	}()
	_, spanCtx := StartSpanFromContext(context.Background(), "test")

	mockStreamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		correlationId, ok := ctx.Value(CorrelationIDCtxKey).(string)
		assert.Equal(t, true, ok)
		assert.NotNil(t, correlationId)
		assert.NotEqual(t, "", correlationId)
		return nil, nil
	}

	stream, err := StreamClientInterceptor(
		spanCtx,
		&grpc.StreamDesc{},
		&grpc.ClientConn{},
		"method",
		mockStreamer,
	)
	assert.NoError(t, err)
	assert.Nil(t, stream)
}
