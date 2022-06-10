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

package sql

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spothero/tools/log"
	"github.com/spothero/tools/sql/middleware"
	"github.com/spothero/tools/tracing"
)

// DBConnector is the interface which various Database-specific configuration objects must satisfy
// to return a usable database connection to the caller
type DBConnector interface {
	Connect() error
}

type wrappedSQLOptions struct {
	middleware                 middleware.Middleware
	registerer                 prometheus.Registerer
	mustRegister               bool
	metricsCollectionFrequency time.Duration
	driverName                 string
}

func newDefaultWrappedSQLOptions(driverName string) wrappedSQLOptions {
	return wrappedSQLOptions{
		middleware:                 middleware.Middleware{log.SQLMiddleware, tracing.SQLMiddleware},
		registerer:                 prometheus.DefaultRegisterer,
		mustRegister:               true,
		metricsCollectionFrequency: 5 * time.Second,
		driverName:                 driverName,
	}
}

// WrappedSQLOption is a function that adds configuration for wrapping both
// PostgreSQL and MySQL drivers.
type WrappedSQLOption func(*wrappedSQLOptions)

// WithMiddleware adds the provided middleware to the wrapped SQL options. Defaults to log and tracing middleware.
func WithMiddleware(m middleware.Middleware) func(*wrappedSQLOptions) {
	return func(config *wrappedSQLOptions) {
		config.middleware = m
	}
}

// WithMiddleware sets the Prometheus registerer to the wrapped SQL options. Defaults to the prometheus default
// registerer.
func WithMetricsRegisterer(r prometheus.Registerer) func(*wrappedSQLOptions) {
	return func(config *wrappedSQLOptions) {
		config.registerer = r
	}
}

// WithMustRegister sets whether or not metrics must register in Prometheus. Defaults to true.
func WithMustRegister(r bool) func(*wrappedSQLOptions) {
	return func(config *wrappedSQLOptions) {
		config.mustRegister = r
	}
}

// WithMetricsCollectionFrequency sets the frequency at which database metrics will be collected
// and made available in the Prometheus registry. Defaults to 5 seconds.
func WithMetricsCollectionFrequency(f time.Duration) func(*wrappedSQLOptions) {
	return func(config *wrappedSQLOptions) {
		config.metricsCollectionFrequency = f
	}
}

// WithDriverName overrides the default wrapped driver name so that multiple wrapped
// drivers can be registered at once.
func WithDriverName(n string) func(*wrappedSQLOptions) {
	return func(config *wrappedSQLOptions) {
		config.driverName = n
	}
}
