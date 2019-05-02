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
	"github.com/spf13/pflag"
)

// RegisterFlags registers SQL flags with pflags
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
	flags.DurationVar(&pc.MetricsFrequency, "pg-metrics-frequency", pc.MetricsFrequency, "Postgres metrics export frequency")
}
