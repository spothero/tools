// Copyright 2020 SpotHero
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
	"path"

	"github.com/getsentry/sentry-go"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"google.golang.org/grpc"
)

// grpcErrToSentry recovers panics, flushes the panic data to Sentry and then re-panics the error
func grpcErrToSentry(ctx context.Context) {
	if err := recover(); err != nil {
		if hub := sentry.GetHubFromContext(ctx); hub != nil {
			if eventID := hub.RecoverWithContext(ctx, err); eventID != nil {
				hub.Flush(flushTimeout)
			}
		}
		panic(err)
	}
}

// configureHub creates a Sentry hub and configures it with appropriate GRPC tags. The hub is then
// embedded into the context and returned to the user.
func configureHub(ctx context.Context, fullMethodName string) context.Context {
	hub := sentry.CurrentHub().Clone()
	hub.Scope().SetTags(map[string]string{
		"grpc.service": path.Dir(fullMethodName)[1:],
		"grpc.method":  path.Base(fullMethodName),
	})
	if span := opentracing.SpanFromContext(ctx); span != nil {
		if sc, ok := span.Context().(jaeger.SpanContext); ok {
			hub.Scope().SetTag("correlation_id", sc.TraceID().String())
		}
	}
	return sentry.SetHubOnContext(ctx, hub)
}

// UnaryServerInterceptor returns a new unary server interceptor with the sentry hub placed in the
// context
func UnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	ctx = configureHub(ctx, info.FullMethod)
	defer grpcErrToSentry(ctx)
	return handler(ctx, req)
}

// StreamServerInterceptor returns a new streaming server interceptor with the sentry hub placed in
// the context
func StreamServerInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	ctx := configureHub(stream.Context(), info.FullMethod)
	wrapped := grpc_middleware.WrapServerStream(stream)
	wrapped.WrappedContext = ctx
	defer grpcErrToSentry(ctx)
	return handler(srv, wrapped)
}
