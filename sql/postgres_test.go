// Copyright 2021 SpotHero
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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaultPostgresConfig(t *testing.T) {
	assert.Equal(
		t,
		NewDefaultPostgresConfig("test", "testdb"),
		PostgresConfig{
			ApplicationName: "test",
			Host:            "127.0.0.1",
			Port:            5432,
			Database:        "testdb",
			ConnectTimeout:  5 * time.Second,
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
			"empty application name results in an error",
			PostgresConfig{Database: "testdb", ApplicationName: ""},
			true,
			"",
		}, {
			"no options returns a basic postgres url",
			PostgresConfig{
				ApplicationName: "test",
				Database:        "test",
				Host:            "127.0.0.1",
				Port:            5432,
			},
			false,
			"postgres://127.0.0.1:5432/test?application_name=test&sslmode=disable",
		}, {
			"username and password are encoded into the URL",
			PostgresConfig{
				ApplicationName: "test",
				Database:        "test",
				Host:            "127.0.0.1",
				Port:            5432,
				Username:        "user",
				Password:        "pass",
			},
			false,
			"postgres://user:pass@127.0.0.1:5432/test?application_name=test&sslmode=disable",
		}, {
			"ssl options are encoded",
			PostgresConfig{
				ApplicationName: "test",
				Database:        "test",
				Host:            "127.0.0.1",
				Port:            5432,
				SSL:             true,
				SSLCert:         "/ssl/cert/path",
				SSLKey:          "/ssl/key/path",
				SSLRootCert:     "/ssl/root/cert/path",
			},
			false,
			"postgres://127.0.0.1:5432/test?application_name=test&sslmode=verify-full&sslcert=/ssl/cert/path&sslkey=/ssl/key/path&sslrootcert=/ssl/root/cert/path",
		}, {
			"connect timeout is properly encoded when specified",
			PostgresConfig{
				ApplicationName: "test",
				Database:        "test",
				Host:            "127.0.0.1",
				Port:            5432,
				ConnectTimeout:  3 * time.Second,
			},
			false,
			"postgres://127.0.0.1:5432/test?application_name=test&sslmode=disable&connect_timeout=3",
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

func TestErrorConnect(t *testing.T) {
	config := NewDefaultPostgresConfig("test", "testdb")
	_, _, err := config.Connect(context.Background())
	assert.NotNil(t, err)
}
