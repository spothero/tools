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
	"log"

	"go.uber.org/zap"

	"go.uber.org/zap/zapcore"
)

// LoggingConfig defines the necessary configuration for instantiating a Logger
type LoggingConfig struct {
	SentryLoggingEnabled bool
	UseDevelopmentLogger bool
	OutputPaths          []string
	ErrorOutputPaths     []string
	Level                string
	SamplingInitial      int
	SamplingThereafter   int
	AppVersion           string
	GitSha               string
}

// Logger is a zap logger. If performance is a concern, use this logger.
var Logger = zap.NewNop()

// SugaredLogger abstracts away types and lets the zap library figure
// them out so that the caller doesn't have to import zap into their package
// but is slightly slower and creates more garbage.
var SugaredLogger = Logger.Sugar()

// InitializeLogger sets up the logger. This function should be called as soon
// as possible. Any use of the logger provided by this package will be a nop
// until this function is called
func (lc *LoggingConfig) InitializeLogger() {
	var err error
	var logConfig zap.Config
	var level zapcore.Level
	if err := level.Set(lc.Level); err != nil {
		fmt.Printf("Invalid log level %s. Using INFO.", lc.Level)
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
		Logger, err = zap.NewDevelopment()
	} else {
		logConfig = zap.Config{
			Level:             zap.NewAtomicLevelAt(level),
			Development:       false,
			DisableStacktrace: false,
			Sampling: &zap.SamplingConfig{
				Initial:    lc.SamplingInitial,
				Thereafter: lc.SamplingThereafter,
			},
			Encoding:         "json",
			EncoderConfig:    zap.NewProductionEncoderConfig(),
			OutputPaths:      append(lc.OutputPaths, "stdout"),
			ErrorOutputPaths: append(lc.ErrorOutputPaths, "stderr"),
			InitialFields:    map[string]interface{}{"appVersion": lc.AppVersion, "gitSha": lc.GitSha},
		}
	}
	Logger, err = logConfig.Build()
	if err != nil {
		fmt.Printf("Error initializing Logger: %s\n", err.Error())
	}
	if lc.SentryLoggingEnabled && err == nil {
		// attach Sentry core
		Logger = Logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewTee(core, &SentryCore{})
		}))
	}
	SugaredLogger = Logger.Sugar()
}

// CreateStdLogger returns a standard-library compatible logger
func CreateStdLogger(zapLogger *zap.Logger, logLevel string) (*log.Logger, error) {
	switch {
	case logLevel == "debug":
		return zap.NewStdLogAt(zapLogger, zapcore.DebugLevel)
	case logLevel == "info":
		return zap.NewStdLogAt(zapLogger, zapcore.InfoLevel)
	case logLevel == "warn":
		return zap.NewStdLogAt(zapLogger, zapcore.WarnLevel)
	case logLevel == "error":
		return zap.NewStdLogAt(zapLogger, zapcore.ErrorLevel)
	case logLevel == "panic":
		return zap.NewStdLogAt(zapLogger, zapcore.PanicLevel)
	case logLevel == "fatal":
		return zap.NewStdLogAt(zapLogger, zapcore.FatalLevel)
	}
	return nil, fmt.Errorf("Unknown log level %s", logLevel)
}
