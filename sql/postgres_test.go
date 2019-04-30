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

	"github.com/stretchr/testify/assert"
)

func TestNewPostgresConfig(t *testing.T) {
	assert.Equal(
		t,
		NewPostgresConfig("testdb"),
		PostgresConfig{
			Host:             "localhost",
			Port:             5432,
			Database:         "testdb",
			ConnectTimeout:   5 * time.Second,
			MetricsFrequency: 5 * time.Second,
		},
	)
}

func TestPostgresConfigBuildConnectionString(t *testing.T) {
	tests := []struct {
		name        string
		c           PostgresConfig
		expectError bool
		expectedURL string
	}{
		{
			"empty database name results in an error",
			PostgresConfig{Database: ""},
			true,
			"",
		}, {
			"no options returns a basic postgres url",
			PostgresConfig{
				Database: "test",
				Host:     "localhost",
				Port:     5432,
			},
			false,
			"postgres://localhost:5432/test?sslmode=disable",
		}, {
			"username and password are encoded into the URL",
			PostgresConfig{
				Database: "test",
				Host:     "localhost",
				Port:     5432,
				Username: "user",
				Password: "pass",
			},
			false,
			"postgres://user:pass@localhost:5432/test?sslmode=disable",
		}, {
			"ssl options are encoded",
			PostgresConfig{
				Database:    "test",
				Host:        "localhost",
				Port:        5432,
				SSL:         true,
				SSLCert:     "/ssl/cert/path",
				SSLKey:      "/ssl/key/path",
				SSLRootCert: "/ssl/root/cert/path",
			},
			false,
			"postgres://localhost:5432/test?sslmode=verify-full&sslcert=/ssl/cert/path&sslkey=/ssl/key/path&sslrootcert=/ssl/root/cert/path",
		}, {
			"connect timeout is properly encoded when specified",
			PostgresConfig{
				Database:       "test",
				Host:           "localhost",
				Port:           5432,
				ConnectTimeout: 3 * time.Second,
			},
			false,
			"postgres://localhost:5432/test?sslmode=disable&connect_timeout=3",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			url, err := test.c.buildConnectionString()
			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, test.expectedURL, url)
		})
	}
}

func TestInstrumentPostgres(t *testing.T) {
	assert.False(t, pgWrapped)
	err := instrumentPostgres()
	assert.NoError(t, err)
	assert.True(t, pgWrapped)
	err = instrumentPostgres()
	assert.Error(t, err)
}

func TestPostgresConfigConnect(t *testing.T) {

}
