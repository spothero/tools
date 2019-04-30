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
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus"
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
			metrics := newMetrics(registry, test.mustRegister)
			if test.duplicate {
				if test.mustRegister {
					assert.Panics(t, func() { newMetrics(registry, test.mustRegister) })
				} else {
					assert.NotPanics(t, func() { _ = newMetrics(registry, test.mustRegister) })
				}
			}
			assert.NotNil(t, metrics.maxOpenConnections)
			assert.NotNil(t, metrics.openConnections)
			assert.NotNil(t, metrics.inUseConnections)
			assert.NotNil(t, metrics.idleConnections)
			assert.NotNil(t, metrics.waitCountConnections)
			assert.NotNil(t, metrics.waitDurationConnections)
			assert.NotNil(t, metrics.maxIdleClosedConnections)
			assert.NotNil(t, metrics.maxLifetimeClosedConnections)
		})
	}
}

func TestExportMetrics(t *testing.T) {
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	assert.NotPanics(t, func() {
		metrics := newMetrics(nil, false)
		cancelChannel := metrics.exportMetrics(&sqlx.DB{DB: db, Mapper: nil}, "test", 5*time.Millisecond)
		timer := time.NewTimer(10 * time.Millisecond)
		<-timer.C
		cancelChannel <- struct{}{}
	})
}
