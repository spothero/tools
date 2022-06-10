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
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/spothero/tools/log"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerzap "github.com/uber/jaeger-client-go/log/zap"
	"go.uber.org/zap"
)

// CorrelationIDCtxKeyType is the type used to uniquely place the trace header in contexts
type CorrelationIDCtxKeyType int

// CorrelationIDCtxKey is the key into any context.Context  which maps to the
// correlation id of the given context. This correlation ID can be
// conveyed to external clients in order to correlate external systems with
// SpotHero tracing and logging.
const CorrelationIDCtxKey CorrelationIDCtxKeyType = iota

// Config defines the necessary configuration for instantiating a Tracer
type Config struct {
	Enabled               bool
	SamplerType           string
	SamplerParam          float64
	ReporterLogSpans      bool
	ReporterMaxQueueSize  int
	ReporterFlushInterval time.Duration
	AgentHost             string
	AgentPort             int
	ServiceName           string
}

// ConfigureTracer instantiates and configures the OpenTracer and returns the tracer closer
func (c Config) ConfigureTracer() io.Closer {
	samplerConfig := jaegercfg.SamplerConfig{}
	if c.SamplerType == "" {
		c.SamplerType = jaeger.SamplerTypeConst
	}
	samplerConfig.Type = c.SamplerType
	samplerConfig.Param = c.SamplerParam

	reporterConfig := jaegercfg.ReporterConfig{}
	reporterConfig.LogSpans = c.ReporterLogSpans
	reporterConfig.QueueSize = c.ReporterMaxQueueSize
	reporterConfig.BufferFlushInterval = c.ReporterFlushInterval
	reporterConfig.LocalAgentHostPort = fmt.Sprintf("%s:%d", c.AgentHost, c.AgentPort)

	jaegerConfig := jaegercfg.Configuration{
		ServiceName: c.ServiceName,
		Sampler:     &samplerConfig,
		Reporter:    &reporterConfig,
		Disabled:    !c.Enabled,
	}

	logger := log.Get(context.Background()).Named("jaeger")
	tracer, closer, err := jaegerConfig.NewTracer(
		jaegercfg.Logger(jaegerzap.NewLogger(logger)))
	if err != nil {
		logger.Error("could not initialize jaeger tracer", zap.Error(err))
		return nil
	}
	logger.Info("jaeger tracer configured", zap.Bool("enabled", c.Enabled))
	opentracing.SetGlobalTracer(tracer)
	return closer
}

// TraceOutbound injects outbound HTTP requests with OpenTracing headers
func TraceOutbound(r *http.Request, span opentracing.Span) error {
	return opentracing.GlobalTracer().Inject(
		span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(r.Header))
}

// EmbedCorrelationID embeds the current Trace ID as the correlation ID in the context logger
func EmbedCorrelationID(ctx context.Context) context.Context {
	// While this removes the veneer of OpenTracing abstraction, the current specification does not
	// provide a method of accessing Trace ID directly. Until OpenTracing 2.0 is released with
	// support for abstract access for Trace ID we will coerce the type to the underlying tracer.
	// See: https://github.com/opentracing/specification/issues/123
	if span := opentracing.SpanFromContext(ctx); span != nil {
		if sc, ok := span.Context().(jaeger.SpanContext); ok {
			// Embed the Trace ID in the logging context for all future requests
			correlationID := sc.TraceID().String()
			ctx = log.NewContext(ctx, log.Get(ctx).With(zap.String("correlation_id", correlationID)))
			ctx = context.WithValue(ctx, CorrelationIDCtxKey, correlationID)
		}
	}
	return ctx
}
