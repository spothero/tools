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

package tracing

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
)

// UnaryServerInterceptor returns a new unary server interceptor that adds the correlation_id to
// the logger context. Note that this middleware should *always* be placed after the opentracing
// middleware to ensure that an opentracing context is present on the context. Additionally, this
// middleware should always appear *before* the logging middleware to ensure that the
// correlation_id is properly logged.
func UnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return handler(embedCorrelationID(ctx), req)
}

// StreamServerInterceptor returns a new unary server interceptor that adds the correlation_id to
// the logger context. Note that this middleware should *always* be placed after the opentracing
// middleware to ensure that an opentracing context is present on the context. Additionally, this
// middleware should always appear *before* the logging middleware to ensure that the
// correlation_id is properly logged.
func StreamServerInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	wrapped := grpc_middleware.WrapServerStream(stream)
	wrapped.WrappedContext = embedCorrelationID(stream.Context())
	return handler(srv, wrapped)
}
