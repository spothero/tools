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
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// logKey is the type used to uniquely place the logger within context.Context
type ctxKey int

const logKey ctxKey = iota

// logger is the default zap logger
var logger = zap.NewNop()

// LoggingConfig defines the necessary configuration for instantiating a Logger
//
// Cores provides optional functionality to allow users to Tee log output to additional Log cores
type LoggingConfig struct {
	UseDevelopmentLogger bool
	OutputPaths          []string
	ErrorOutputPaths     []string
	Level                string
	SamplingInitial      int
	SamplingThereafter   int
	Encoding             string                 // One of "json" or "console"
	Fields               map[string]interface{} // These fields will be applied to all logs. Strongly recommend `version` at a minimum.
	Cores                []zapcore.Core
	Options              []zap.Option
	counter              *prometheus.CounterVec
}

// metricsHook is a callback hook used to track logging metrics at runtime
func (lc *LoggingConfig) metricsHook(entry zapcore.Entry) error {
	lc.counter.With(prometheus.Labels{"level": entry.Level.CapitalString()}).Inc()
	return nil
}

// InitializeLogger sets up the logger. This function should be called as soon
// as possible. Any use of the logger provided by this package will be a nop
// until this function is called
func (lc *LoggingConfig) InitializeLogger() error {
	var err error
	var logConfig zap.Config
	var level zapcore.Level
	if lc.Encoding == "" {
		lc.Encoding = "json"
	}
	if err := level.Set(lc.Level); err != nil {
		fmt.Printf("invalid log level %s - using INFO", lc.Level)
		level.Set("info")
	}
	if lc.UseDevelopmentLogger {
		// Initialize logger with default development options
		// which enables debug logging, uses console encoder, writes to
		// stderr, and disables sampling.
		// See https://godoc.org/go.uber.org/zap#NewDevelopmentConfig
		logConfig = zap.NewDevelopmentConfig()
		logConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		logConfig.Level = zap.NewAtomicLevelAt(level)
	} else {
		logConfig = zap.Config{
			Level:             zap.NewAtomicLevelAt(level),
			Development:       false,
			DisableStacktrace: false,
			Sampling: &zap.SamplingConfig{
				Initial:    lc.SamplingInitial,
				Thereafter: lc.SamplingThereafter,
			},
			Encoding:         lc.Encoding,
			EncoderConfig:    zap.NewProductionEncoderConfig(),
			OutputPaths:      append(lc.OutputPaths, "stdout"),
			ErrorOutputPaths: append(lc.ErrorOutputPaths, "stderr"),
			InitialFields:    lc.Fields,
		}
	}

	lc.counter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "logs_emitted",
			Help: "Total number of logs emitted by this application instance",
		},
		[]string{"level"},
	)

	for _, core := range lc.Cores {
		lc.Options = append(lc.Options, zap.WrapCore(func(existingCore zapcore.Core) zapcore.Core {
			return zapcore.NewTee(existingCore, core)
		}))
	}
	lc.Options = append(lc.Options, zap.Hooks(lc.metricsHook))
	if logger, err = logConfig.Build(lc.Options...); err != nil {
		return fmt.Errorf("error initializing logger - %s", err.Error())
	}
	return nil
}

// NewContext creates and returns a new context with a wrapped logger. If fields are specified,
// all future invocations of the context logger will include those fields as well. This concept is
// useful if you wish for all downstream logs from the site of a given context to include some
// contextual information. For example, once your application has unpacked the Trace ID, you may
// wish to log that information with every future request.
func NewContext(ctx context.Context, fields ...zapcore.Field) context.Context {
	return context.WithValue(ctx, logKey, Get(ctx).With(fields...))
}

// Get returns the logger wrapped with the given context. This function is intended to be
// used as a mechanism for adding scoped arbitrary logging information to the logger. If a nil
// context is passed, the default global logger is returned.
func Get(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return logger
	}
	if ctxLogger, ok := ctx.Value(logKey).(*zap.Logger); ok {
		return ctxLogger
	}
	return logger
}
