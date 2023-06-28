package sql

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"os"

	"github.com/gchaincl/sqlhooks"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/spothero/tools/log"
	"go.uber.org/zap"
)

const (
	// If no override option is provided, this is the name of the wrapped MySQL driver that will be
	// registered when calling MySQLConfig.Connect.
	DefaultWrappedMySQLDriverName = "wrappedMySQL"
	tlsConfigName                 = "MySQLTLSConfig"
)

// MySQLConfig adds a path for a CA cert to mysql.Config. When CACertPath is set,
// NewWrappedMySQL will verify the database identity with the provided CA cert.
type MySQLConfig struct {
	// Path to the server CA certificate for SSL connections
	CACertPath string
	mysql.Config
}

// Connect uses the given Config struct to establish a connection with the database.
// See the documentation for WrappedSQLOption functions to configure how the driver
// gets wrapped. Note that calling Connect multiple times is not allowed with the
// same driver name option.
//
// If no error occurs, the database connection, and a close function are returned
func (c MySQLConfig) Connect(ctx context.Context, options ...WrappedSQLOption) (*sqlx.DB, func() error, error) {
	opts := newDefaultWrappedSQLOptions(DefaultWrappedMySQLDriverName)
	for _, option := range options {
		option(&opts)
	}
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
	sql.Register(opts.driverName, sqlhooks.Wrap(mysql.MySQLDriver{}, &opts.middleware))
	if c.CACertPath != "" && c.TLSConfig == "" {
		if certErr := c.loadCACert(); certErr != nil {
			return nil, nil, certErr
		}
	}
	db, err := sqlx.ConnectContext(ctx, opts.driverName, c.FormatDSN())
	if err != nil {
		return nil, nil, err
	}
	metricsChannel := newMetrics(
		c.DBName, opts.registerer, opts.mustRegister).exportMetrics(db.DB, opts.metricsCollectionFrequency)
	return db, func() error {
		close(metricsChannel)
		return db.Close()
	}, nil
}

// Read a CA cert file and registers a TLS config with the cert under the constant tlsConfigName name
func (c *MySQLConfig) loadCACert() error {
	rootPool := x509.NewCertPool()
	pem, err := os.ReadFile(c.CACertPath)
	if err != nil {
		return err
	}
	if ok := rootPool.AppendCertsFromPEM(pem); !ok {
		return fmt.Errorf("failed to MySQL CA PEM")
	}
	if registrationErr := mysql.RegisterTLSConfig(tlsConfigName, &tls.Config{RootCAs: rootPool}); registrationErr != nil {
		return registrationErr
	}
	c.TLSConfig = tlsConfigName
	return nil
}
