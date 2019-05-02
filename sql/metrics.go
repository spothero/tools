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

package sql

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spothero/tools/log"
	"github.com/spothero/tools/sql/middleware"
)

// metrics contains the prometheus metrics measuring connections
type metrics struct {
	maxOpenConnections           *prometheus.GaugeVec
	openConnections              *prometheus.GaugeVec
	inUseConnections             *prometheus.GaugeVec
	idleConnections              *prometheus.GaugeVec
	waitCountConnections         *prometheus.GaugeVec
	waitDurationConnections      *prometheus.GaugeVec
	maxIdleClosedConnections     *prometheus.GaugeVec
	maxLifetimeClosedConnections *prometheus.GaugeVec
	queryDuration                *prometheus.HistogramVec
	queryCount                   *prometheus.CounterVec
	dbName                       string
}

// newMetrics initializes and returns a metrics object
func newMetrics(dbName string, registry prometheus.Registerer, mustRegister bool) metrics {
	labelNames := []string{"db_name"}
	queryLabelNames := append(labelNames, "query_name", "outcome")
	maxOpenConnections := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "db_max_open_connections_count",
			Help: "maximum number of open database connections",
		},
		labelNames,
	)
	openConnections := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "db_open_connections_count",
			Help: "number of open database connections",
		},
		labelNames,
	)
	inUseConnections := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "db_in_use_connections_count",
			Help: "number of in-use database connections",
		},
		labelNames,
	)
	idleConnections := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "db_idle_connections_count",
			Help: "number of idle database connections",
		},
		labelNames,
	)
	waitCountConnections := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "db_wait_connections_count_total",
			Help: "total number database connections waited for",
		},
		labelNames,
	)
	waitDurationConnections := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "db_wait_connections_seconds_sum",
			Help: "number of waiting database connections",
		},
		labelNames,
	)
	maxIdleClosedConnections := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "db_max_idle_closed_connections_total",
			Help: "total number of closed database connections due to SetMaxIdleConns",
		},
		labelNames,
	)
	maxLifetimeClosedConnections := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "db_max_lifetime_closed_connections_total",
			Help: "total number of closed database connections due to SetConnMaxLifetime",
		},
		labelNames,
	)
	queryDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "db_query_duration_seconds",
			Help: "Total duration histgoram for the DB Query",
			// Power of 2 time - 1ms, 2ms, 4ms, ... 32768ms, +Inf ms
			Buckets: prometheus.ExponentialBuckets(0.001, 2.0, 16),
		},
		queryLabelNames,
	)
	queryCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_query_count",
			Help: "Total number of times the query has executed",
		},
		queryLabelNames,
	)

	// If the user hasnt provided a Prometheus Registry, use the global Registry
	if registry == nil {
		registry = prometheus.DefaultRegisterer
	}
	if mustRegister {
		registry.MustRegister(maxOpenConnections)
		registry.MustRegister(openConnections)
		registry.MustRegister(inUseConnections)
		registry.MustRegister(idleConnections)
		registry.MustRegister(waitCountConnections)
		registry.MustRegister(waitDurationConnections)
		registry.MustRegister(maxIdleClosedConnections)
		registry.MustRegister(maxLifetimeClosedConnections)
		registry.MustRegister(queryDuration)
		registry.MustRegister(queryCount)
	} else {
		if err := registry.Register(maxOpenConnections); err != nil {
			log.Get(context.Background()).Error("failed to register db max open connections")
		}
		if err := registry.Register(openConnections); err != nil {
			log.Get(context.Background()).Error("failed to register db open connections")
		}
		if err := registry.Register(inUseConnections); err != nil {
			log.Get(context.Background()).Error("failed to register db in use connections")
		}
		if err := registry.Register(idleConnections); err != nil {
			log.Get(context.Background()).Error("failed to register db idle connections")
		}
		if err := registry.Register(waitCountConnections); err != nil {
			log.Get(context.Background()).Error("failed to register db wait connection count")
		}
		if err := registry.Register(waitDurationConnections); err != nil {
			log.Get(context.Background()).Error("failed to register db wait connection duration")
		}
		if err := registry.Register(maxIdleClosedConnections); err != nil {
			log.Get(context.Background()).Error("failed to register db max idle closed connections")
		}
		if err := registry.Register(maxLifetimeClosedConnections); err != nil {
			log.Get(context.Background()).Error("failed to register db max lifetime closed connections")
		}
		if err := registry.Register(queryDuration); err != nil {
			log.Get(context.Background()).Error("failed to register db query duration")
		}
		if err := registry.Register(queryCount); err != nil {
			log.Get(context.Background()).Error("failed to register db query count")
		}
	}
	return metrics{
		dbName:                       dbName,
		maxOpenConnections:           maxOpenConnections,
		openConnections:              openConnections,
		inUseConnections:             inUseConnections,
		idleConnections:              idleConnections,
		waitCountConnections:         waitCountConnections,
		waitDurationConnections:      waitDurationConnections,
		maxIdleClosedConnections:     maxIdleClosedConnections,
		maxLifetimeClosedConnections: maxLifetimeClosedConnections,
		queryDuration:                queryDuration,
		queryCount:                   queryCount,
	}
}

// exportMetrics creates a goroutine which periodically scrapes the core database driver for
// connection details, exporting those metrics for prometheus scraping
func (m metrics) exportMetrics(db *sqlx.DB, frequency time.Duration) chan<- bool {
	ticker := time.NewTicker(frequency)
	kill := make(chan bool)
	labels := prometheus.Labels{"db_name": m.dbName}
	go func() {
		for {
			select {
			case <-ticker.C:
				stats := db.Stats()
				m.maxOpenConnections.With(labels).Set(float64(stats.MaxOpenConnections))
				m.openConnections.With(labels).Set(float64(stats.OpenConnections))
				m.inUseConnections.With(labels).Set(float64(stats.InUse))
				m.idleConnections.With(labels).Set(float64(stats.Idle))
				m.waitCountConnections.With(labels).Set(float64(stats.WaitCount))
				m.waitDurationConnections.With(labels).Set(float64(stats.WaitDuration.Seconds()))
				m.maxIdleClosedConnections.With(labels).Set(float64(stats.MaxIdleClosed))
				m.maxLifetimeClosedConnections.With(labels).Set(float64(stats.MaxLifetimeClosed))
			case <-kill:
				ticker.Stop()
				return
			}
		}
	}()
	return kill
}

// Middleware defines SQL Middleware for capturing metrics around SQL queries. Using this
// middleware will ensure that prometheus exports on a per-queryName basis a histogram of
// duration, as well as a lifetime call counter. Query outcome is captured as a label on both
// metrics.
func (m metrics) Middleware(ctx context.Context, queryName, query string, args ...interface{}) (context.Context, middleware.MiddlewareEnd, error) {
	startTime := time.Now()
	mwEnd := func(ctx context.Context, queryName, query string, queryErr error, args ...interface{}) (context.Context, error) {
		outcome := "success"
		if queryErr != nil {
			outcome = "error"
		}
		labels := prometheus.Labels{
			"db_name":    m.dbName,
			"query_name": queryName,
			"outcome":    outcome,
		}
		m.queryCount.With(labels).Inc()
		m.queryDuration.With(labels).Observe(time.Since(startTime).Seconds())
		return ctx, nil
	}
	return ctx, mwEnd, nil
}
