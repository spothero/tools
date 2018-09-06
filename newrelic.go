package core

import (
	"github.com/newrelic/go-agent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewRelicConfig defines the necessary configuration for instantiating a New Relic Client
type NewRelicConfig struct {
	AppName    string
	Enabled    bool
	LicenseKey string
	LogLevel   string
}

var (
	newRelicApp newrelic.Application
)

// InitializeNewRelic Configures the New Relic Logging Reporter for Usage
func (nc *NewRelicConfig) InitializeNewRelic() {
	nrConfig := newrelic.NewConfig(nc.AppName, nc.LicenseKey)
	nrConfig.Enabled = nc.Enabled
	nrConfig.Logger = nc
	app, err := newrelic.NewApplication(nrConfig)
	if err != nil {
		Logger.Error("Failed to initialize New Relic", zap.Error(err))
	}
	newRelicApp = app
}

// NewRelicApp returns the global New Relic Application Logger, if any
func NewRelicApp() newrelic.Application {
	return newRelicApp
}

// Error logs error messages to the New Relic Reporter
func (nc *NewRelicConfig) Error(msg string, context map[string]interface{}) {
	Logger.Named("new-relic").Error(msg, zap.Reflect("context", context))
}

// Warn logs warning messages to the New Relic Reporter
func (nc *NewRelicConfig) Warn(msg string, context map[string]interface{}) {
	Logger.Named("new-relic").Warn(msg, zap.Reflect("context", context))
}

// Info logs info messages to the New Relic Reporter
func (nc *NewRelicConfig) Info(msg string, context map[string]interface{}) {
	Logger.Named("new-relic").Info(msg, zap.Reflect("context", context))
}

// Debug logs debug messages to the New Relic Reporter
func (nc *NewRelicConfig) Debug(msg string, context map[string]interface{}) {
	Logger.Named("new-relic").Debug(msg, zap.Reflect("context", context))
}

// DebugEnabled returns true if debug logging is enabled, else false
func (nc *NewRelicConfig) DebugEnabled() bool {
	var level zapcore.Level
	if err := level.Set(nc.LogLevel); err != nil {
		level.Set("info")
	}
	return level <= zapcore.DebugLevel
}
