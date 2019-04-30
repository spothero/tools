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
		logger.Error("Couldn't initialize Jaeger tracer", zap.Error(err))
		return nil
	}
	if !c.Enabled {
		logger.Info("Jaeger tracer configured but disabled")
	} else {
		logger.Info("Configured Jaeger tracer")
	}
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
