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
	goSQL "database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gchaincl/sqlhooks"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spothero/tools/log"
	"go.uber.org/zap"
)

// pgDriverName is the name used to register the wrapped postgres SQL Driver with sql interface
var pgDriverName = "instrumentedPostgres"

// pgWrapped keeps track of whether or not the postgres client has already been wrapped
var pgWrapped = false

// defaultTimeout defines the default timeout for SQL connections to be established
const defaultTimeout = 5 * time.Second

// defaultMetricsFrequency defines the default core SQL metrics scrape frequency
const defaultMetricsFrequency = 5 * time.Second

// PostgresConfig defines Postgres SQL connection information
type PostgresConfig struct {
	Host             string        // The host where the database is located
	Port             uint16        // The port on which the database is listening
	Username         string        // The username for the database
	Password         string        // The password for the database
	Database         string        // The name of the database
	ConnectTimeout   time.Duration // Amount of time to wait before timing out
	SSL              bool          // If true, connect to the database with SSL
	SSLCert          string        // Path to the SSL Certificate, if any
	SSLKey           string        // Path to the SSL Key, if any
	SSLRootCert      string        // Path to the SSL Root Certificate, if any
	MetricsFrequency time.Duration // How often to export core database metrics
	Middleware       Middleware    // List of SQL Middlewares to apply, if any
}

// NewPostgresConfig creates and return a default postgres configuration.
func NewPostgresConfig(dbName string) PostgresConfig {
	return PostgresConfig{
		Host:             "localhost",
		Port:             5432,
		Database:         dbName,
		ConnectTimeout:   defaultTimeout,
		MetricsFrequency: defaultMetricsFrequency,
	}
}

// buildConnectionString transforms the PostgresConfig into a usable connection string for lib/pq.
// If a missing or invalid field is provided, an error is returned.
func (pc PostgresConfig) buildConnectionString() (string, error) {
	if pc.Database == "" {
		return "", fmt.Errorf("postgres database name was not specified")
	}
	auth := ""
	if pc.Username != "" || pc.Password != "" {
		auth = fmt.Sprintf("%s:%s@", pc.Username, pc.Password)
	}
	url := fmt.Sprintf(
		"postgres://%s%s:%d/%s",
		auth,
		pc.Host,
		pc.Port,
		pc.Database,
	)
	options := make([]string, 0)
	if pc.SSL {
		options = append(options, "sslmode=verify-full")
		if pc.SSLCert != "" {
			options = append(options, fmt.Sprintf("sslcert=%s", pc.SSLCert))
		}
		if pc.SSLKey != "" {
			options = append(options, fmt.Sprintf("sslkey=%s", pc.SSLKey))
		}
		if pc.SSLRootCert != "" {
			options = append(options, fmt.Sprintf("sslrootcert=%s", pc.SSLRootCert))
		}
	} else {
		options = append(options, "sslmode=disable")
	}
	if pc.ConnectTimeout.Seconds() > 0 {
		timeoutStr := strconv.Itoa(int(pc.ConnectTimeout.Seconds()))
		options = append(options, fmt.Sprintf("connect_timeout=%s", timeoutStr))
	}
	if len(options) > 0 {
		url = fmt.Sprintf("%s?%s", url, strings.Join(options, "&"))
	}
	return url, nil
}

// instrumentPostgres registers an instrumented and wrapped Postgres driver with the SQL library
// so that all calls capture metrics, are traced, and capture debug logs.
func (pc PostgresConfig) instrumentPostgres() error {
	if pgWrapped {
		return fmt.Errorf("postgres already instrumented")
	}
	goSQL.Register(pgDriverName, sqlhooks.Wrap(&pq.Driver{}, &pc.Middleware))
	pgWrapped = true
	return nil
}

// Connect uses the given Config struct to establish a connection with the database.
// Optionally, a prometheus registry may be provided. If no registry is provided, the global
// registry will be used. Additionally, users may specify that failed metrics registration should
// result in a panic via the `mustRegister` flag.
//
// The database connection, deferable close function, and error are returned
func (pc PostgresConfig) Connect(ctx context.Context, registry prometheus.Registerer, mustRegister bool) (*sqlx.DB, func(), error) {
	if err := pc.instrumentPostgres(); err != nil {
		log.Get(ctx).Warn(
			"attempted to instrument sql.DB when it is already instrumented",
			zap.Error(err),
		)
	}
	log.Get(ctx).Info(
		"connecting to postgres",
		zap.String("database", pc.Database),
		zap.String("host", pc.Host),
		zap.Uint16("port", pc.Port),
	)
	url, err := pc.buildConnectionString()
	if err != nil {
		return nil, nil, err
	}
	db, err := sqlx.ConnectContext(ctx, pgDriverName, url)
	if err != nil {
		log.Get(ctx).Error("unable to connect to postgres")
		return nil, nil, err
	}
	log.Get(ctx).Info(
		"connected to postgres",
		zap.String("database", pc.Database),
		zap.String("host", pc.Host),
		zap.Uint16("port", pc.Port),
	)
	dbMetricsChannel := newMetrics(pc.Database, registry, mustRegister).exportMetrics(db, pc.MetricsFrequency)
	return db, func() {
		dbMetricsChannel <- struct{}{}
		db.Close()
	}, nil
}
