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
	"github.com/spf13/pflag"
)

// RegisterFlags registers PostgreSQL flags with pflags
func (pc *PostgresConfig) RegisterFlags(flags *pflag.FlagSet) {
	flags.StringVar(&pc.Host, "pg-host", pc.Host, "Postgres Host Address")
	flags.Uint16Var(&pc.Port, "pg-port", pc.Port, "Postgres Port")
	flags.StringVar(&pc.Username, "pg-username", pc.Username, "Postgres Username")
	flags.StringVar(&pc.Password, "pg-password", pc.Password, "Postgres Password")
	flags.StringVar(&pc.Database, "pg-database", pc.Database, "Postgres Database Name")
	flags.DurationVar(&pc.ConnectTimeout, "pg-connect-timeout", pc.ConnectTimeout, "Postgres Connection Timeout")
	flags.BoolVar(&pc.SSL, "pg-ssl", pc.SSL, "If true, use SSL when connecting to Postgres")
	flags.StringVar(&pc.SSLCert, "pg-ssl-cert", pc.SSLCert, "Path of the Postgres SSL Certificate on disk")
	flags.StringVar(&pc.SSLKey, "pg-ssl-key", pc.SSLKey, "Path of the Postgres SSL Key on disk")
	flags.StringVar(&pc.SSLRootCert, "pg-ssl-root-cert", pc.SSLRootCert, "Path of the Postgres SSL Root Cert on disk")
}

// RegisterFlags registers MySQL flags with pflags
func (c *MySQLConfig) RegisterFlags(flags *pflag.FlagSet) {
	flags.StringVar(&c.Addr, "mysql-address", c.Addr, "MySQL address")
	flags.StringVar(&c.DBName, "mysql-database", c.DBName, "MySQL database name")
	flags.StringVar(&c.User, "mysql-user", c.User, "MySQL username")
	flags.StringVar(&c.Passwd, "mysql-password", c.Passwd, "MySQL password")
	flags.StringVar(&c.Net, "mysql-net", "tcp", "MySQL network connection type")
	flags.DurationVar(&c.Timeout, "mysql-dial-timeout", c.Timeout, "MySQL dial timeout")
	flags.DurationVar(&c.ReadTimeout, "mysql-read-timeout", c.ReadTimeout, "MySQL I/O read timeout")
	flags.DurationVar(&c.WriteTimeout, "mysql-write-timeout", c.WriteTimeout, "MySQL I/O write timeout")
	flags.StringVar(&c.CACertPath, "mysql-ca-cert-path", "", "Path to the CA cert for the MySQL server")
}
