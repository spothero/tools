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

package sql

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gchaincl/sqlhooks"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/spothero/tools/log"
	"go.uber.org/zap"
)

const (
	// If no override option is provided, this is the name of the wrapped Postgres driver that will be
	// registered when calling PostgresConfig.Connect.
	DefaultWrappedPgDriverName = "instrumentedPostgres"
	// defaultTimeout defines the default timeout for SQL connections to be established
	defaultTimeout = 5 * time.Second
)

// PostgresConfig defines Postgres SQL connection information
type PostgresConfig struct {
	ApplicationName string        // The name of the application connecting. Useful for attributing db load.
	Host            string        // The host where the database is located
	Port            uint16        // The port on which the database is listening
	Username        string        // The username for the database
	Password        string        // The password for the database
	Database        string        // The name of the database
	ConnectTimeout  time.Duration // Amount of time to wait before timing out
	SSL             bool          // If true, connect to the database with SSL
	SSLCert         string        // Path to the SSL Certificate, if any
	SSLKey          string        // Path to the SSL Key, if any
	SSLRootCert     string        // Path to the SSL Root Certificate, if any
}

// NewDefaultPostgresConfig creates and return a default postgres configuration.
func NewDefaultPostgresConfig(appName, dbName string) PostgresConfig {
	return PostgresConfig{
		ApplicationName: appName,
		Host:            "localhost",
		Port:            5432,
		Database:        dbName,
		ConnectTimeout:  defaultTimeout,
	}
}

// buildConnectionString transforms the PostgresConfig into a usable connection string for lib/pq.
// If a missing or invalid field is provided, an error is returned.
func (pc PostgresConfig) buildConnectionString() (string, error) {
	if pc.Database == "" {
		return "", fmt.Errorf("postgres database name was not specified")
	}
	if pc.ApplicationName == "" {
		return "", fmt.Errorf("application name must be specified to connect to postgres")
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
	options := []string{fmt.Sprintf("application_name=%s", pc.ApplicationName)}
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
	return fmt.Sprintf("%s?%s", url, strings.Join(options, "&")), nil
}

// Connect uses the given Config struct to establish a connection with the database.
// See the documentation for WrappedSQLOption functions to configure how the driver
// gets wrapped. Note that calling Connect multiple times is not allowed with the
// same driver name option.
//
// If no error occurs, the database connection, and a close function are returned
func (pc PostgresConfig) Connect(ctx context.Context, options ...WrappedSQLOption) (*sqlx.DB, func() error, error) {
	opts := newDefaultWrappedSQLOptions(DefaultWrappedPgDriverName)
	for _, option := range options {
		option(&opts)
	}
	sql.Register(opts.driverName, sqlhooks.Wrap(&pq.Driver{}, opts.middleware))

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
	db, err := sqlx.ConnectContext(ctx, opts.driverName, url)
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
	dbMetricsChannel := newMetrics(
		pc.Database, opts.registerer, opts.mustRegister).exportMetrics(db.DB, opts.metricsCollectionFrequency)
	return db, func() error {
		close(dbMetricsChannel)
		return db.Close()
	}, nil
}
