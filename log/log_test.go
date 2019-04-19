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

package log

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name            string
		ctx             context.Context
		expectedOutcome *zap.Logger
	}{
		{
			"nil context returns the default logger",
			nil,
			logger,
		}, {
			"populated context without a logger returns the global logger",
			context.Background(),
			logger,
		}, {
			"populated context with logger returns that logger",
			context.WithValue(context.Background(), logKey, logger.Named("not global")),
			logger.Named("not global"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectedOutcome, Get(test.ctx))
		})
	}
}

func TestNewContext(t *testing.T) {
	tests := []struct {
		name            string
		ctx             context.Context
		fields          []zapcore.Field
		expectedOutcome context.Context
	}{
		{
			"no fields leads to a default logger context",
			context.Background(),
			nil,
			context.WithValue(context.Background(), logKey, logger),
		}, {
			"providing fields leads to a new context with a new logger with the provided fields",
			context.Background(),
			[]zapcore.Field{
				zap.Int("zero", 0),
				zap.Int("one", 1),
			},
			context.WithValue(
				context.Background(),
				logKey,
				logger.With(
					zap.Int("zero", 0),
					zap.Int("one", 1),
				),
			),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectedOutcome, NewContext(test.ctx, test.fields...))
		})
	}
}

func TestMetricsHook(t *testing.T) {
	tests := []struct {
		name string
		lc   LoggingConfig

		expectedOutput int
	}{
		{
			"log level counter should return with correct value",
			LoggingConfig{},
			1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.lc.counter = prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: "logs_emitted",
					Help: "Total number of logs emitted by this application instance",
				},
				[]string{"level"},
			)
			for i := 0; i < test.expectedOutput; i++ {
				test.lc.metricsHook(zapcore.Entry{Level: zapcore.DebugLevel})
			}
			counter, err := test.lc.counter.GetMetricWith(prometheus.Labels{"level": "DEBUG"})
			assert.NoError(t, err)
			pb := &dto.Metric{}
			counter.Write(pb)
			assert.Equal(t, test.expectedOutput, int(pb.Counter.GetValue()))
			prometheus.Unregister(test.lc.counter)
		})
	}
}

func TestInitializeLogger(t *testing.T) {
	tests := []struct {
		name        string
		lc          LoggingConfig
		expectError bool
	}{
		{
			"debug initialization should create a development logger",
			LoggingConfig{UseDevelopmentLogger: true},
			false,
		},
		{
			"initialization with a bad level defaults to INFO",
			LoggingConfig{Level: "DOESNOTEXIST"},
			false,
		},
		{
			"non-debug initialization should create a JSON logger",
			LoggingConfig{UseDevelopmentLogger: false},
			false,
		},
		{
			"non-debug initialization to a nonexistent path raises an error",
			LoggingConfig{UseDevelopmentLogger: false, OutputPaths: []string{"C://DNE"}},
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Are these tests basically worthless? Yes, but Zap doesnt export any fields on the
			// logger, so unfortunately there's not much for us to test here. We could optionally
			// wrap the zap logger in our own struct and pack along a series of our own fields for
			// testing, but we have opted not to do this.
			if test.expectError {
				assert.Error(t, test.lc.InitializeLogger())
				return
			}
			assert.NoError(t, test.lc.InitializeLogger())
		})
	}
}
