// Copyright 2023 SpotHero
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
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func TestNewMetrics(t *testing.T) {
	tests := []struct {
		name         string
		mustRegister bool
		duplicate    bool
	}{
		{
			"when must register is true and we do not duplicate registration no panic occurs",
			true,
			false,
		},
		{
			"when must register is true and we duplicate registration a panic occurs",
			true,
			true,
		},
		{
			"when must register is false and we do not duplicate registration no panic occurs",
			false,
			false,
		},
		{
			"when must register is false and we duplicate registration a panic occurs",
			false,
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry := prometheus.NewRegistry()
			testMetrics := newMetrics("test", registry, test.mustRegister)
			if test.duplicate {
				if test.mustRegister {
					assert.Panics(t, func() { newMetrics("test", registry, test.mustRegister) })
				} else {
					assert.NotPanics(t, func() { _ = newMetrics("test", registry, test.mustRegister) })
				}
			}
			assert.NotNil(t, testMetrics.maxOpenConnections)
			assert.NotNil(t, testMetrics.openConnections)
			assert.NotNil(t, testMetrics.inUseConnections)
			assert.NotNil(t, testMetrics.idleConnections)
			assert.NotNil(t, testMetrics.waitCountConnections)
			assert.NotNil(t, testMetrics.waitDurationConnections)
			assert.NotNil(t, testMetrics.maxIdleClosedConnections)
			assert.NotNil(t, testMetrics.maxLifetimeClosedConnections)
		})
	}
}

func TestExportMetrics(t *testing.T) {
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	assert.NotPanics(t, func() {
		testMetrics := newMetrics("test", nil, false)
		cancelChannel := testMetrics.exportMetrics(db, 5*time.Millisecond)
		timer := time.NewTimer(10 * time.Millisecond)
		<-timer.C
		cancelChannel <- true
	})
}

func TestMiddleware(t *testing.T) {
	tests := []struct {
		name      string
		expectErr bool
	}{
		{
			"middleware counts queries and measures time",
			false,
		},
		{
			"middleware recognizes errors and labels appropriately",
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry := prometheus.NewRegistry()
			m := metrics{
				dbName: "test",
				queryDuration: prometheus.NewHistogramVec(
					prometheus.HistogramOpts{
						Name: "db_query_duration_seconds",
						Help: "Total duration histgoram for the DB Query",
						// Power of 2 time - 1ms, 2ms, 4ms, ... 32768ms, +Inf ms
						Buckets: prometheus.ExponentialBuckets(0.001, 2.0, 16),
					},
					[]string{"db_name", "query_name", "outcome"},
				),
				queryCount: prometheus.NewCounterVec(
					prometheus.CounterOpts{
						Name: "db_query_count",
						Help: "Total number of times the query has executed",
					},
					[]string{"db_name", "query_name", "outcome"},
				),
			}
			registry.MustRegister(m.queryDuration)
			registry.MustRegister(m.queryCount)
			ctx, mwEnd, err := m.Middleware(context.Background(), "query-name", "query")
			assert.NoError(t, err)
			assert.NotNil(t, ctx)
			assert.NotNil(t, mwEnd)

			var queryErr error
			if test.expectErr {
				queryErr = fmt.Errorf("query-error")
			}
			ctx, err = mwEnd(ctx, "query-name", "query", queryErr)
			assert.NoError(t, err)
			assert.NotNil(t, ctx)

			expectedOutcome := "success"
			if test.expectErr {
				expectedOutcome = "error"
			}
			labels := prometheus.Labels{
				"db_name":    "test",
				"query_name": "query-name",
				"outcome":    expectedOutcome,
			}

			histogram, err := m.queryDuration.GetMetricWith(labels)
			assert.NoError(t, err)
			pb := &dto.Metric{}
			assert.NoError(t, histogram.(prometheus.Histogram).Write(pb))
			buckets := pb.Histogram.GetBucket()
			assert.NotEmpty(t, buckets)
			for _, bucket := range pb.Histogram.GetBucket() {
				// Choose a bucket which gives a full second to this test and ensure we have a count of at
				// least one. This just ensures that our timer is working. This request should never take
				// longer than a millisecond, but we hugely increase the threshold to ensure we dont
				// introduce tests that periodically fail for no clear reason.
				if bucket.GetUpperBound() >= 1.0 {
					assert.Equal(t, uint64(1), bucket.GetCumulativeCount())
					break
				}
			}
			counter, err := m.queryCount.GetMetricWith(labels)
			assert.NoError(t, err)
			pb = &dto.Metric{}
			assert.NoError(t, counter.Write(pb))
			assert.Equal(t, 1, int(pb.Counter.GetValue()))
		})
	}
}
