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
}

// newMetrics initializes and returns a metrics object
func newMetrics(registry prometheus.Registerer, mustRegister bool) metrics {
	labelNames := []string{"db_name"}
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
	}
	return metrics{
		maxOpenConnections:           maxOpenConnections,
		openConnections:              openConnections,
		inUseConnections:             inUseConnections,
		idleConnections:              idleConnections,
		waitCountConnections:         waitCountConnections,
		waitDurationConnections:      waitDurationConnections,
		maxIdleClosedConnections:     maxIdleClosedConnections,
		maxLifetimeClosedConnections: maxLifetimeClosedConnections,
	}
}

// exportMetrics creates a goroutine which periodically scrapes the core database driver for
// connection details, exporting those metrics for prometheus scraping
func (m metrics) exportMetrics(db *sqlx.DB, dbName string, frequency time.Duration) chan<- struct{} {
	ticker := time.NewTicker(frequency)
	kill := make(chan struct{})
	labels := prometheus.Labels{"db_name": dbName}
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
