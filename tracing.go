// Copyright 2018 SpotHero
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

package core

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/opentracing/opentracing-go"
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
		Sampler:  &samplerConfig,
		Reporter: &reporterConfig,
		Disabled: !tc.Enabled,
	}

	tracer, closer, err := jaegerConfig.New(
		tc.ServiceName, jaegercfg.Logger(jaegerzap.NewLogger(Logger.Named("jaeger"))))
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
