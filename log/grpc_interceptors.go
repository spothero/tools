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

package log

import (
	"context"
	"path"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor returns a new unary server interceptor that includes a logger in context
// This code is heavily inspired by the excellent open-source grpc middleware library:
// https://github.com/grpc-ecosystem/go-grpc-middleware/blob/master/logging/zap/server_interceptors.go
// We have chosen to reimplement some aspects of this functionality to more specifically fit our
// logging needs.
func UnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	startTime := time.Now()
	ctx = setLogCtx(ctx, info.FullMethod, startTime)
	requestLogger := Get(ctx)
	logger := requestLogger.Named("grpc")
	logger.Debug("request received")
	newCtx := NewContext(ctx, requestLogger)
	resp, err := handler(newCtx, req)
	code := status.Code(err)
	Get(newCtx).With(
		zap.String("grpc.code", code.String()),
		zap.Duration("grpc.duration", time.Since(startTime)),
		zap.Error(err),
	).Check(grpc_zap.DefaultCodeToLevel(code), "returning response").Write()
	return resp, err
}

// StreamServerInterceptor returns a new streaming server interceptor that includes a logger in context
// This code is heavily inspired by the excellent open-source grpc middleware library:
// https://github.com/grpc-ecosystem/go-grpc-middleware/blob/master/logging/zap/server_interceptors.go
// We have chosen to reimplement some aspects of this functionality to more specifically fit our
// logging needs.
func StreamServerInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	startTime := time.Now()
	ctx := setLogCtx(stream.Context(), info.FullMethod, startTime)
	requestLogger := Get(ctx)
	logger := requestLogger.Named("grpc")
	logger.Debug("request received")
	newCtx := NewContext(ctx, requestLogger)
	wrapped := grpc_middleware.WrapServerStream(stream)
	wrapped.WrappedContext = newCtx
	err := handler(srv, wrapped)
	code := status.Code(err)
	Get(newCtx).With(
		zap.String("grpc.code", code.String()),
		zap.Duration("grpc.duration", time.Since(startTime)),
		zap.Error(err),
	).Check(grpc_zap.DefaultCodeToLevel(code), "returning response").Write()
	return err
}

// setLogCtx configures and sets common grpc information on the context logger. Again, this
// function is very much borrowed directly from the work of the grpc-ecosystem project and adapted
// to meet our specific needs.
// https://github.com/grpc-ecosystem/go-grpc-middleware/blob/master/logging/zap/server_interceptors.go#L88
func setLogCtx(ctx context.Context, fullMethodName string, startTime time.Time) context.Context {
	fields := []zapcore.Field{
		zap.String("system", "grpc"),
		zap.String("span.kind", "server"),
		zap.String("grpc.service", path.Dir(fullMethodName)[1:]),
		zap.String("grpc.method", path.Base(fullMethodName)),
		zap.String("grpc.start_time", startTime.Format(time.RFC3339)),
	}
	if d, ok := ctx.Deadline(); ok {
		fields = append(fields, zap.String("grpc.request.deadline", d.Format(time.RFC3339)))
	}
	return NewContext(ctx, Get(ctx).With(fields...))
}
