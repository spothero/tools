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

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
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
