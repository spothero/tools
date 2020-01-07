// Copyright 2020 SpotHero
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
	"sync"

	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ctxKey is the type used to uniquely place the logger within context.Context
type ctxKey int

// logKey is the value used to uniquely place the logger within context.Context
const logKey ctxKey = iota

// logger is the default zap logger
var logger = zap.NewNop()

// loggerMutex protects the logger from change by multiple threads
var loggerMutex = &sync.Mutex{}

// Config defines the necessary configuration for instantiating a Logger
type Config struct {
	UseDevelopmentLogger bool                   // If true, use default zap development logger settings
	OutputPaths          []string               // List of locations where logs should be placed. Defaults to stdout
	ErrorOutputPaths     []string               // List of locations where error logs should be placed. Defaults to stderr
	Level                string                 // The level at which logs should be produced. Defaults to INFO
	SamplingInitial      int                    // Initial sampling rate for the logger for each cycle
	SamplingThereafter   int                    // Sampling rate after that initial cap is hit, per cycle
	Encoding             string                 // One of "json" or "console"
	Fields               map[string]interface{} // These fields will be applied to all logs. Strongly recommend `version` at a minimum.
	Cores                []zapcore.Core         // Provides optional functionality to allow users to Tee log output to additional Log cores
	Options              []zap.Option           // Users may specify any additional zap logger options which override defaults
	counter              *prometheus.CounterVec // Counter tracks the number of logs by level
}

// metricsHook is a callback hook used to track logging metrics at runtime
func metricsHook(counter *prometheus.CounterVec) func(entry zapcore.Entry) error {
	return func(entry zapcore.Entry) error {
		counter.With(prometheus.Labels{"level": entry.Level.CapitalString()}).Inc()
		return nil
	}
}

// InitializeLogger sets up the logger. This function should be called as soon
// as possible. Any use of the logger provided by this package will be a nop
// until this function is called.
func (c Config) InitializeLogger() error {
	var err error
	var logConfig zap.Config
	var level zapcore.Level
	if c.Encoding == "" {
		c.Encoding = "json"
	}
	if err := level.Set(c.Level); err != nil {
		fmt.Printf("invalid log level %s - using INFO\n", c.Level)
		level = zapcore.InfoLevel
	}
	if c.UseDevelopmentLogger {
		// Initialize logger with default development options
		// which enables debug logging, uses console encoder, writes to
		// stderr, and disables sampling.
		// See https://godoc.org/go.uber.org/zap#NewDevelopmentConfig
		logConfig = zap.NewDevelopmentConfig()
		logConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		logConfig.Level = zap.NewAtomicLevelAt(level)
		logConfig.InitialFields = c.Fields
	} else {
		logConfig = zap.Config{
			Level:             zap.NewAtomicLevelAt(level),
			Development:       false,
			DisableStacktrace: false,
			Sampling: &zap.SamplingConfig{
				Initial:    c.SamplingInitial,
				Thereafter: c.SamplingThereafter,
			},
			Encoding:         c.Encoding,
			EncoderConfig:    zap.NewProductionEncoderConfig(),
			OutputPaths:      append(c.OutputPaths, "stdout"),
			ErrorOutputPaths: append(c.ErrorOutputPaths, "stderr"),
			InitialFields:    c.Fields,
		}
	}

	c.counter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "logs_emitted",
			Help: "Total number of logs emitted by this application instance",
		},
		[]string{"level"},
	)

	for _, core := range c.Cores {
		c.Options = append(c.Options, zap.WrapCore(func(existingCore zapcore.Core) zapcore.Core {
			return zapcore.NewTee(existingCore, core)
		}))
	}
	c.Options = append(c.Options, zap.Hooks(metricsHook(c.counter)))
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	if logger, err = logConfig.Build(c.Options...); err != nil {
		return fmt.Errorf("error initializing logger - %s", err.Error())
	}
	grpc_zap.ReplaceGrpcLoggerV2(logger)
	return nil
}

// NewContext creates and returns a new context with a wrapped logger. If a logger is not
// provided, the context returned will contain the global logger. This concept is
// useful if you wish for all downstream logs from the site of a given context to include some
// contextual fields or use a sub-logger. For example, once your application has
// unpacked the Trace ID, you may wish to log that information with every future request.
func NewContext(ctx context.Context, l *zap.Logger) context.Context {
	if l == nil {
		return context.WithValue(ctx, logKey, logger)
	}
	return context.WithValue(ctx, logKey, l)
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
