package sql

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spothero/tools/log"
	"github.com/spothero/tools/sql/middleware"
	"github.com/spothero/tools/tracing"
	"github.com/stretchr/testify/assert"
)

func TestNewDefaultWrappedSQLOptions(t *testing.T) {
	expected := wrappedSQLOptions{
		middleware:                 middleware.Middleware{log.SQLMiddleware, tracing.SQLMiddleware},
		registerer:                 prometheus.DefaultRegisterer,
		mustRegister:               true,
		metricsCollectionFrequency: 5 * time.Second,
		driverName:                 "driver",
	}
	actual := newDefaultWrappedSQLOptions("driver")
	assert.Equal(t, len(expected.middleware), len(actual.middleware))
	assert.Equal(t, expected.registerer, actual.registerer)
	assert.Equal(t, expected.mustRegister, actual.mustRegister)
	assert.Equal(t, expected.metricsCollectionFrequency, actual.metricsCollectionFrequency)
	assert.Equal(t, expected.driverName, actual.driverName)
}

func TestWrappedSQLConfigOptions(t *testing.T) {
	tests := []struct {
		name            string
		option          WrappedSQLOption
		expectedOptions wrappedSQLOptions
	}{
		{
			"middleware",
			WithMiddleware(middleware.Middleware{}),
			wrappedSQLOptions{middleware: middleware.Middleware{}},
		}, {
			"registerer",
			WithMetricsRegisterer(prometheus.DefaultRegisterer),
			wrappedSQLOptions{registerer: prometheus.DefaultRegisterer},
		}, {
			"must register",
			WithMustRegister(true),
			wrappedSQLOptions{mustRegister: true},
		}, {
			"metrics collection frequency",
			WithMetricsCollectionFrequency(time.Second),
			wrappedSQLOptions{metricsCollectionFrequency: time.Second},
		}, {
			"driver name",
			WithDriverName("driver"),
			wrappedSQLOptions{driverName: "driver"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opts := wrappedSQLOptions{}
			test.option(&opts)
			assert.Equal(t, test.expectedOptions, opts)
		})
	}
}
