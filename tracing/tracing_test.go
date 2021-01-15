// Copyright 2021 SpotHero
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
	"net/http"
	"testing"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	jaeger "github.com/uber/jaeger-client-go"
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
			closer := test.c.ConfigureTracer()
			if test.expectErr {
				assert.Equal(t, &opentracing.NoopTracer{}, opentracing.GlobalTracer())
			} else {
				assert.NotNil(t, closer)
				defer closer.Close()
				assert.NotNil(t, opentracing.GlobalTracer())
			}
		})
	}
}

func TestTraceOutbound(t *testing.T) {
	req, err := http.NewRequest("GET", "/fake", nil)
	assert.NoError(t, err)
	span := opentracing.StartSpan("test-span")
	defer span.Finish()
	err = TraceOutbound(req, span)
	assert.NoError(t, err)
}

func TestEmbedCorrelationID(t *testing.T) {
	tracer, closer := jaeger.NewTracer("t", jaeger.NewConstSampler(false), jaeger.NewInMemoryReporter())
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)
	_, spanCtx := opentracing.StartSpanFromContext(context.Background(), "test")

	ctx := EmbedCorrelationID(spanCtx)
	correlationId, ok := ctx.Value(CorrelationIDCtxKey).(string)
	assert.Equal(t, true, ok)
	assert.NotNil(t, correlationId)
	assert.NotEqual(t, "", correlationId)
}
