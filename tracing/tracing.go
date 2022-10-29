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
	"errors"
	"github.com/spothero/tools/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"os"
	"strconv"
	"strings"
	"time"
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

// TracerProvider returns an OpenTelemetry TracerProvider configured to use
// the Jaeger exporter that will send spans to the provided url. The returned
// TracerProvider will also use a Resource configured with all the information
// about the application.
func (c Config) TracerProvider() (func(context.Context) error, error) {
	ctx := context.Background()
	logger := log.Get(ctx).Named("otel-tracer-provider")

	// check serviceName is provided or not.
	// If not provided throw the error.
	if c.ServiceName == "" {
		return nil, errors.New("tracing ServiceName can't be empty")
	}

	// Create the Jaeger exporter
	agentPort := "6831" //default port for Jaeger
	if c.AgentPort > 0 {
		agentPort = strconv.Itoa(c.AgentPort)
	}

	exp, err := jaeger.New(
		jaeger.WithAgentEndpoint(jaeger.WithAgentHost(c.AgentHost), jaeger.WithAgentPort(agentPort)))
	if err != nil {
		logger.Error("could not initialize Jaeger OTEL exporter", zap.Error(err))
		return nil, err
	}

	// Set sampler for the trace Provider
	sampler := tracesdk.AlwaysSample()
	if strings.ToLower(c.SamplerType) == "ratio" {
		sampler = tracesdk.TraceIDRatioBased(c.SamplerParam)
	} else if strings.ToLower(c.SamplerType) == "never" {
		sampler = tracesdk.NeverSample()
	}

	tpResource := tracesdk.WithResource(resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(c.ServiceName),
		attribute.String("process.pid", "1"),
		attribute.String("ip", os.Getenv("podIP")),
		attribute.String("hostname", os.Getenv("HOSTNAME")),
		attribute.String("telemetry.sdk.language", "go-telemetry"),
		attribute.String("telemetry.sdk.name", "opentelemetry"),
		attribute.String("telemetry.sdk.version", "1.11.0"),
	))

	tracerProvider := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp,
			tracesdk.WithMaxQueueSize(c.ReporterMaxQueueSize)),
		tracesdk.WithSampler(sampler),
		tpResource,
	)
	otel.SetTracerProvider(tracerProvider)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tracerProvider.Shutdown, nil
}

// EmbedCorrelationID embeds the current Trace ID as the correlation ID in the context logger
// Continuing this function for backward compatability.
func EmbedCorrelationID(ctx context.Context) context.Context {
	// While this removes the veneer of OpenTelemetry abstraction, the current specification does not
	// provide a method of accessing Trace ID directly.
	if span := trace.SpanFromContext(ctx); span != nil {
		sc := span.SpanContext()
		// Embed the Trace ID in the logging context for all future requests
		correlationID := sc.TraceID().String()
		ctx = log.NewContext(ctx, log.Get(ctx).With(zap.String("correlation_id", correlationID)))
		ctx = context.WithValue(ctx, CorrelationIDCtxKey, correlationID)
	}
	return ctx
}

// StartSpanFromContext Start the span from the provided context with provided options
func StartSpanFromContext(ctx context.Context, operationName string, opts ...trace.SpanStartOption) (trace.Span, context.Context) {
	tracer := otel.GetTracerProvider().Tracer(operationName)
	returnCtx, span := tracer.Start(ctx, operationName, opts...)
	return span, returnCtx
}
