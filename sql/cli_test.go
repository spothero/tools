// Copyright 2022 SpotHero
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

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresConfigRegisterFlags(t *testing.T) {
	flags := pflag.NewFlagSet("pflags", pflag.PanicOnError)
	pc := PostgresConfig{}
	pc.RegisterFlags(flags)
	err := flags.Parse(nil)
	assert.NoError(t, err)

	h, err := flags.GetString("pg-host")
	assert.NoError(t, err)
	assert.Equal(t, "", h)

	p, err := flags.GetUint16("pg-port")
	assert.NoError(t, err)
	assert.Equal(t, uint16(0), p)

	u, err := flags.GetString("pg-username")
	assert.NoError(t, err)
	assert.Equal(t, "", u)

	pw, err := flags.GetString("pg-password")
	assert.NoError(t, err)
	assert.Equal(t, "", pw)

	db, err := flags.GetString("pg-database")
	assert.NoError(t, err)
	assert.Equal(t, "", db)

	ct, err := flags.GetDuration("pg-connect-timeout")
	assert.NoError(t, err)
	assert.Equal(t, 0*time.Second, ct)

	ssl, err := flags.GetBool("pg-ssl")
	assert.NoError(t, err)
	assert.False(t, ssl)

	sslCert, err := flags.GetString("pg-ssl-cert")
	assert.NoError(t, err)
	assert.Equal(t, "", sslCert)

	sslKey, err := flags.GetString("pg-ssl-key")
	assert.NoError(t, err)
	assert.Equal(t, "", sslKey)

	sslRootCert, err := flags.GetString("pg-ssl-root-cert")
	assert.NoError(t, err)
	assert.Equal(t, "", sslRootCert)
}

func TestMySQLConfig_RegisterFlags(t *testing.T) {
	flags := pflag.NewFlagSet("pflags", pflag.PanicOnError)
	m := MySQLConfig{}
	m.RegisterFlags(flags)
	err := flags.Parse([]string{
		"--mysql-address", "address",
		"--mysql-database", "database",
		"--mysql-user", "user",
		"--mysql-password", "password",
		"--mysql-net", "net",
		"--mysql-dial-timeout", "12s",
		"--mysql-read-timeout", "34s",
		"--mysql-write-timeout", "56s",
		"--mysql-ca-cert-path", "/path/to/the/ca.pem",
	})
	require.NoError(t, err)

	assert.Equal(t, m.Addr, "address")
	assert.Equal(t, m.DBName, "database")
	assert.Equal(t, m.User, "user")
	assert.Equal(t, m.Passwd, "password")
	assert.Equal(t, m.Net, "net")
	assert.Equal(t, m.Timeout, 12*time.Second)
	assert.Equal(t, m.ReadTimeout, 34*time.Second)
	assert.Equal(t, m.WriteTimeout, 56*time.Second)
	assert.Equal(t, m.CACertPath, "/path/to/the/ca.pem")
}
