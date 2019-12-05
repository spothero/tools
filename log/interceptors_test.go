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

package log

import (
	"context"
	"testing"
	"time"

	grpcmock "github.com/spothero/tools/grpc/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
)

// verifyRequestResponseLogs is used to DRY verify both Stream and Unary Server interceptor log
// fields. Both Stream and Unary interceptors log the same fields
func verifyRequestResponseLogs(t *testing.T, recordedLogs *observer.ObservedLogs) {
	// Test that request parameters are appropriately logged to our standards
	currLogs := recordedLogs.All()
	assert.Len(t, currLogs, 2)
	foundLogKeysRequest := make([]string, len(currLogs[0].Context))
	for idx, field := range currLogs[0].Context {
		foundLogKeysRequest[idx] = field.Key
	}
	assert.ElementsMatch(
		t,
		[]string{
			"system",
			"span.kind",
			"grpc.service",
			"grpc.method",
			"grpc.start_time",
		},
		foundLogKeysRequest,
	)

	// Test that response parameters are appropriately logged to our standards
	foundLogKeysResponse := make([]string, len(currLogs[1].Context))
	for idx, field := range currLogs[1].Context {
		foundLogKeysResponse[idx] = field.Key
	}
	assert.ElementsMatch(
		t,
		[]string{
			"grpc.code",
			"grpc.duration",
			"system",
			"span.kind",
			"grpc.service",
			"grpc.method",
			"grpc.start_time",
			"", // No error, so the error tag is empty
		},
		foundLogKeysResponse,
	)
}

// TODO: Test different types of GRPC codes
func TestUnaryServerInterceptor(t *testing.T) {
	recordedLogs := makeLoggerObservable(t, zapcore.InfoLevel)
	ctx := context.Background()
	info := &grpc.UnaryServerInfo{}
	mockHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return struct{}{}, nil
	}
	resp, err := UnaryServerInterceptor(ctx, nil, info, mockHandler)
	assert.NoError(t, err)
	assert.Equal(t, struct{}{}, resp)
	verifyRequestResponseLogs(t, recordedLogs)
}

// TODO: Test different types of GRPC codes
func TestStreamServerInterceptor(t *testing.T) {
	recordedLogs := makeLoggerObservable(t, zapcore.InfoLevel)
	info := &grpc.StreamServerInfo{}
	mockHandler := func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}
	mockStream := grpcmock.MockServerStream{}
	mockStream.On("Context").Return(context.Background())
	err := StreamServerInterceptor(nil, mockStream, info, mockHandler)
	assert.NoError(t, err)
	verifyRequestResponseLogs(t, recordedLogs)
}

// TODO: Test that the fields are correctly set for the logger
func TestSetLogCtx(t *testing.T) {
	startCtx := context.Background()
	ctx := setLogCtx(startCtx, "service.method", time.Now())
	// TODO: Actually test this
	assert.NotNil(t, ctx)
}
