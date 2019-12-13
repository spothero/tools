package sql

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/gchaincl/sqlhooks"
	"github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spothero/tools/log"
	"go.uber.org/zap"

	"github.com/spothero/tools/sql/middleware"
)

const (
	wrappedMySQLDriverName = "wrappedMySQL"
	tlsConfigName          = "MySQLTLSConfig"
)

// MySQLConfig adds a path for a CA cert to mysql.Config. When CACertPath is set,
// NewWrappedMySQL will verify the database identity with the provided CA cert.
type MySQLConfig struct {
	mysql.Config
	// Path to the server CA certificate for SSL connections
	CACertPath string
}

// NewWrappedMySQLDrive builds a MySQL driver wrapped with the provided middleware and starts
// a periodic task that collects database connection metrics in the provided prometheus registerer.
// If the CACertPath is set in the passed in config, the database connection will be over SSL.
// The returned function should be called close the database connection instead of calling Close()
// directly as it stops the periodic metrics task.
func NewWrappedMySQL(
	ctx context.Context,
	config MySQLConfig,
	middleware middleware.Middleware,
	registry prometheus.Registerer,
	mustRegister bool,
	collectionFrequency time.Duration,
) (*sql.DB, func() error, error) {
	stdLogger, err := zap.NewStdLogAt(log.Get(ctx).Named("mysql"), zap.ErrorLevel)
	if err != nil {
		log.Get(ctx).Error(
			"mysql driver errors will be output to stderr because the standard logger failed to build",
			zap.Error(err))
	} else {
		// this can only ever error if the logger is nil, but that can only happen if the
		// standard logger failed to build
		_ = mysql.SetLogger(stdLogger)
	}
	sql.Register(wrappedMySQLDriverName, sqlhooks.Wrap(mysql.MySQLDriver{}, &middleware))
	if config.CACertPath != "" && config.TLSConfig == "" {
		if err := config.loadCACert(); err != nil {
			return nil, nil, err
		}
	}
	db, err := sql.Open(wrappedMySQLDriverName, config.FormatDSN())
	if err != nil {
		return nil, nil, err
	}
	metricsChannel := newMetrics(config.DBName, registry, mustRegister).exportMetrics(db, collectionFrequency)
	return db, func() error {
		close(metricsChannel)
		return db.Close()
	}, nil
}

// Read a CA cert file and registers a TLS config with the cert under the constant tlsConfigName name
func (c *MySQLConfig) loadCACert() error {
	rootPool := x509.NewCertPool()
	pem, err := ioutil.ReadFile(c.CACertPath)
	if err != nil {
		return err
	}
	if ok := rootPool.AppendCertsFromPEM(pem); !ok {
		return fmt.Errorf("failed to MySQL CA PEM")
	}
	if err := mysql.RegisterTLSConfig(tlsConfigName, &tls.Config{RootCAs: rootPool}); err != nil {
		return err
	}
	c.TLSConfig = tlsConfigName
	return nil
}
