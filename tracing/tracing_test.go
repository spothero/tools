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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigureTracer(t *testing.T) {
	tests := []struct {
		name      string
		c         Config
		expectErr bool
	}{
		{
			"service name provided leads to no error",
			Config{ServiceName: "service-name"},
			false,
		},
		{
			"no service name provided leads to an error",
			Config{Enabled: true},
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			shutdown, _ := test.c.TracerProvider()
			if test.expectErr {
				//assert.Equal(t, t, opentracing.GlobalTracer())
			} else {
				assert.NotNil(t, shutdown)
				defer shutdown(context.Background())
				assert.NotNil(t, shutdown)
			}
		})
	}
}

func TestEmbedCorrelationID(t *testing.T) {
	octx := context.Background()
	shutdown, _ := GetTracerProvider()
	defer shutdown(octx)
	_, spanCtx := StartSpanFromContext(octx, "test")

	ctx := EmbedCorrelationID(spanCtx)
	correlationId, ok := ctx.Value(CorrelationIDCtxKey).(string)
	assert.Equal(t, true, ok)
	assert.NotNil(t, correlationId)
	assert.NotEqual(t, "", correlationId)
}

func GetTracerProvider() (func(context.Context) error, error) {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, fmt.Errorf("creating stdout exporter: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("tracing-test"),
			semconv.ServiceVersionKey.String("0.0.1"),
		)),
	)
	otel.SetTracerProvider(tracerProvider)
	return tracerProvider.Shutdown, nil
}
