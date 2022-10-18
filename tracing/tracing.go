package tracing

import (
	"context"
	"github.com/spothero/tools/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"strconv"
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
	logger := log.Get(context.Background()).Named("jaeger-exporter")

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

	// Set sampler for the traceprovider
	sampler := tracesdk.AlwaysSample()

	tracerProvider := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp,
			tracesdk.WithMaxQueueSize(c.ReporterMaxQueueSize)),

		tracesdk.WithSampler(sampler),

		// Record information about this application in a Resource.
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(c.ServiceName),
		)),
	)
	otel.SetTracerProvider(tracerProvider)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tracerProvider.Shutdown, nil
}

// EmbedCorrelationID embeds the current Trace ID as the correlation ID in the context logger
func EmbedCorrelationID(ctx context.Context) context.Context {
	// While this removes the veneer of OpenTracing abstraction, the current specification does not
	// provide a method of accessing Trace ID directly. Until OpenTracing 2.0 is released with
	// support for abstract access for Trace ID we will coerce the type to the underlying tracer.
	// See: https://github.com/opentracing/specification/issues/123
	if span := trace.SpanFromContext(ctx); span != nil {
		sc := span.SpanContext()
		// Embed the Trace ID in the logging context for all future requests
		correlationID := sc.TraceID().String()
		ctx = log.NewContext(ctx, log.Get(ctx).With(zap.String("correlation_id", correlationID)))
		ctx = context.WithValue(ctx, CorrelationIDCtxKey, correlationID)
	}
	return ctx
}

func StartSpanFromContext(ctx context.Context, operationName string, opts ...trace.SpanStartOption) (trace.Span, context.Context) {
	tracer := otel.GetTracerProvider().Tracer(operationName)
	returnCtx, span := tracer.Start(ctx, operationName, opts...)
	return span, returnCtx
}
