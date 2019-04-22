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

package tools

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/spothero/tools/http/writer"
	"github.com/spothero/tools/log"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerzap "github.com/uber/jaeger-client-go/log/zap"
	"go.uber.org/zap"
)

// TracingConfig defines the necessary configuration for instantiating a Tracer
type TracingConfig struct {
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
func (tc *TracingConfig) ConfigureTracer() io.Closer {
	samplerConfig := jaegercfg.SamplerConfig{}
	if tc.SamplerType == "" {
		tc.SamplerType = jaeger.SamplerTypeConst
	}
	samplerConfig.Type = tc.SamplerType
	samplerConfig.Param = tc.SamplerParam

	reporterConfig := jaegercfg.ReporterConfig{}
	reporterConfig.LogSpans = tc.ReporterLogSpans
	reporterConfig.QueueSize = tc.ReporterMaxQueueSize
	reporterConfig.BufferFlushInterval = tc.ReporterFlushInterval
	reporterConfig.LocalAgentHostPort = fmt.Sprintf("%s:%d", tc.AgentHost, tc.AgentPort)

	jaegerConfig := jaegercfg.Configuration{
		ServiceName: tc.ServiceName,
		Sampler:     &samplerConfig,
		Reporter:    &reporterConfig,
		Disabled:    !tc.Enabled,
	}

	tracer, closer, err := jaegerConfig.NewTracer(
		jaegercfg.Logger(jaegerzap.NewLogger(Logger.Named("jaeger"))))
	if err != nil {
		Logger.Error("Couldn't initialize Jaeger tracer", zap.Error(err))
		return nil
	}
	if !tc.Enabled {
		Logger.Info("Jaeger tracer configured but disabled")
	} else {
		Logger.Info("Configured Jaeger tracer")
	}
	opentracing.SetGlobalTracer(tracer)
	return closer
}

// TraceOutbound injects outbound HTTP requests with OpenTracing headers
func TraceOutbound(r *http.Request, span opentracing.Span) {
	opentracing.GlobalTracer().Inject(
		span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(r.Header))
}

// TracingMiddleware extracts the OpenTracing context on all incoming HTTP requests, if present. if
// no trace ID is present in the headers, a trace is initiated.
//
// The following tags are placed on all incoming HTTP requests:
// * http.method
// * http.hostname
// * http.port
// * http.remote_address
//
// Outbound responses will be tagged with the following tags, if applicable:
// * http.status_code
// * error (if the status code is >= 500)
//
// The returned HTTP Request includes the wrapped OpenTracing Span Context.
func TracingMiddleware(sr *writer.StatusRecorder, r *http.Request) (func(), *http.Request) {
	wireContext, err := opentracing.GlobalTracer().Extract(
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(r.Header))
	if err != nil {
		log.Get(r.Context()).Debug("failed to extract opentracing context on an incoming http request")
	}
	span, spanCtx := opentracing.StartSpanFromContext(r.Context(), writer.FetchRoutePathTemplate(r), ext.RPCServerOption(wireContext))
	span = span.SetTag("http.method", r.Method)
	span = span.SetTag("http.url", r.URL.String())

	// While this removes the veneer of OpenTracing abstraction, the current specification does not
	// provide a method of accessing Trace ID directly. Until OpenTracing 2.0 is released with
	// support for abstract access for Trace ID we will coerce the type to the underlying tracer.
	// See: https://github.com/opentracing/specification/issues/123
	if sc, ok := span.Context().(jaeger.SpanContext); ok {
		// Embed the Trace ID in the logging context for all future requests
		spanCtx = log.NewContext(spanCtx, zap.String("trace_id", sc.TraceID().String()))
	}
	return func() {
		span.SetTag("http.status_code", strconv.Itoa(sr.StatusCode))
		// 5XX Errors are our fault -- note that this span belongs to an errored request
		if sr.StatusCode >= http.StatusInternalServerError {
			span.SetTag("error", true)
		}
		span.Finish()
	}, r.WithContext(spanCtx)
}
