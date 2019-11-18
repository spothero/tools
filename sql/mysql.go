package sql

import (
	"context"
	"database/sql"
	"time"

	"github.com/gchaincl/sqlhooks"
	"github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spothero/tools/log"
	"go.uber.org/zap"

	"github.com/spothero/tools/sql/middleware"
)

const wrappedMySQLDriverName = "wrappedMySQL"

// NewWrappedMySQLDrive builds a MySQL driver wrapped with the provided middleware and starts
// a periodic task that collects database connection metrics in the provided prometheus registerer.
// The returned function should be called close the database connection instead of calling Close()
// directly as it stops the periodic metrics task.
func NewWrappedMySQL(
	ctx context.Context,
	config mysql.Config,
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
