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
	l := zap.NewNop()
	tests := []struct {
		name            string
		ctx             context.Context
		logger          *zap.Logger
		expectedOutcome context.Context
	}{
		{
			"no logger leads to a default logger context",
			context.Background(),
			nil,
			context.WithValue(context.Background(), logKey, logger),
		}, {
			"providing logger leads to a new context with provided logger",
			context.Background(),
			l.With(zap.Int("zero", 0), zap.Int("one", 1)),
			context.WithValue(
				context.Background(),
				logKey,
				l.With(
					zap.Int("zero", 0),
					zap.Int("one", 1),
				),
			),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectedOutcome, NewContext(test.ctx, test.logger))
		})
	}
}

func TestMetricsHook(t *testing.T) {
	c := Config{
		counter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "logs_emitted",
				Help: "Total number of logs emitted by this application instance",
			},
			[]string{"level"},
		),
	}
	metricsHookFunc := metricsHook(c.counter)
	assert.NoError(t, metricsHookFunc(zapcore.Entry{Level: zapcore.DebugLevel}))
	counter, err := c.counter.GetMetricWith(prometheus.Labels{"level": "DEBUG"})
	assert.NoError(t, err)
	pb := &dto.Metric{}
	assert.NoError(t, counter.Write(pb))
	assert.Equal(t, 1, int(pb.Counter.GetValue()))
	prometheus.Unregister(c.counter)
}

func TestInitializeLogger(t *testing.T) {
	tests := []struct {
		name        string
		c           Config
		expectError bool
	}{
		{
			"debug initialization should create a development logger",
			Config{UseDevelopmentLogger: true},
			false,
		},
		{
			"initialization with a bad level defaults to INFO",
			Config{Level: "DOESNOTEXIST"},
			false,
		},
		{
			"non-debug initialization should create a JSON logger",
			Config{UseDevelopmentLogger: false},
			false,
		},
		{
			"non-debug initialization to a nonexistent path raises an error",
			Config{UseDevelopmentLogger: false, OutputPaths: []string{"C://DNE"}},
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
				assert.Error(t, test.c.InitializeLogger())
				return
			}
			assert.NoError(t, test.c.InitializeLogger())
		})
	}
}
