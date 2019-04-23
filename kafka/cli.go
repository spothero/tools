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

package kafka

import "github.com/spf13/pflag"

// RegisterFlags registers Kafka flags with pflags
func (c *Config) RegisterFlags(flags *pflag.FlagSet) {
	flags.StringVarP(&c.Broker, "kafka-broker", "b", "kafka:29092", "Kafka broker Address")
	flags.StringVar(&c.ClientID, "kafka-client-id", "client", "Kafka consumer Client ID")
	flags.StringVar(&c.TLSCaCrtPath, "kafka-server-ca-crt-path", "", "Kafka Server TLS CA Certificate Path")
	flags.StringVar(&c.TLSCrtPath, "kafka-client-crt-path", "", "Kafka Client TLS Certificate Path")
	flags.StringVar(&c.TLSKeyPath, "kafka-client-key-path", "", "Kafka Client TLS Key Path")
	flags.BoolVar(&c.Verbose, "kafka-verbose", false, "When this flag is set Kafka will log verbosely")
	flags.BoolVar(&c.JSONEnabled, "enable-json", true, "When this flag is set, messages from Kafka will be consumed as JSON instead of Avro")
	flags.StringVar(&c.KafkaVersion, "kafka-version", "2.1.0", "Kafka broker version")
	flags.StringVar(&c.ProducerCompressionCodec, "kafka-producer-compression-codec", "none", "Compression codec to use when producing messages, one of: \"none\", \"zstd\", \"snappy\", \"lz4\", \"zstd\", \"gzip\"")
	flags.IntVar(&c.ProducerCompressionLevel, "kafka-producer-compression-level", -1000, "Compression level to use on produced messages, -1000 signifies to use the default level.")
	c.SchemaRegistry = &SchemaRegistryConfig{}
	c.SchemaRegistry.RegisterFlags(flags)
}

// RegisterFlags registers Kafka flags with pflags
func (src *SchemaRegistryConfig) RegisterFlags(flags *pflag.FlagSet) {
	flags.StringVarP(&src.SchemaRegistryURL, "kafka-schema-registry", "r", "http://localhost:8081", "Kafka Schema Registry Address")
}
