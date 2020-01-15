// Copyright 2020 SpotHero
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

import (
	"context"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
)

func TestConfig_populateSaramaConfig(t *testing.T) {
	tests := []struct {
		name      string
		input     Config
		check     func(t *testing.T, cfg *sarama.Config)
		expectErr bool
	}{
		{
			"base configuration is populated",
			Config{
				Config:                  *sarama.NewConfig(),
				Verbose:                 true,
				KafkaVersion:            "2.3.0",
				ProducerReturnErrors:    true,
				ProducerReturnSuccesses: true,
				ConsumerReturnErrors:    true,
			},
			func(t *testing.T, cfg *sarama.Config) {
				assert.Equal(t, sarama.V2_3_0_0, cfg.Version)
				assert.True(t, cfg.Producer.Return.Successes)
				assert.True(t, cfg.Producer.Return.Errors)
				assert.True(t, cfg.Consumer.Return.Errors)
			},
			false,
		}, {
			"no registered flags returns the default configuration",
			Config{Config: *sarama.NewConfig()},
			func(t *testing.T, cfg *sarama.Config) {
				expected := sarama.NewConfig()
				expected.Producer.Partitioner = nil

				// partitioner is a function pointer so this will never pass an equality check; just make sure it isn't nil
				assert.NotNil(t, cfg.Producer.Partitioner)
				cfg.Producer.Partitioner = nil

				// unset in the variables that are set by default but get overridden by not setting our config
				expected.Consumer.Return.Errors = false
				expected.Producer.Return.Errors = false
				expected.Producer.RequiredAcks = 0

				assert.Equal(t, expected, cfg)
			},
			false,
		}, {
			"bad version returns an error",
			Config{KafkaVersion: "not.a.real.version"},
			func(t *testing.T, cfg *sarama.Config) {},
			true,
		}, {
			"zstd compression is properly set",
			Config{ProducerCompressionCodec: "zstd"},
			func(t *testing.T, cfg *sarama.Config) {
				assert.Equal(t, sarama.CompressionZSTD, cfg.Producer.Compression)
			},
			false,
		}, {
			"snappy compression is properly set",
			Config{ProducerCompressionCodec: "snappy"},
			func(t *testing.T, cfg *sarama.Config) {
				assert.Equal(t, sarama.CompressionSnappy, cfg.Producer.Compression)
			},
			false,
		}, {
			"lz4 compression is properly set",
			Config{ProducerCompressionCodec: "lz4"},
			func(t *testing.T, cfg *sarama.Config) {
				assert.Equal(t, sarama.CompressionLZ4, cfg.Producer.Compression)
			},
			false,
		}, {
			"gzip compression is properly set",
			Config{ProducerCompressionCodec: "gzip"},
			func(t *testing.T, cfg *sarama.Config) {
				assert.Equal(t, sarama.CompressionGZIP, cfg.Producer.Compression)
			},
			false,
		}, {
			"unknown compression returns an error",
			Config{ProducerCompressionCodec: "beepboop"},
			func(*testing.T, *sarama.Config) {},
			true,
		}, {
			"TLS configuration is loaded",
			Config{
				ProducerCompressionCodec: "none",
				TLSCrtPath:               "../testdata/fake-crt.pem",
				TLSKeyPath:               "../testdata/fake-key.pem",
			},
			func(t *testing.T, cfg *sarama.Config) {
				assert.True(t, cfg.Net.TLS.Enable)
				assert.NotNil(t, cfg.Net.TLS.Config)
			},
			false,
		}, {
			"TLS CA cert is loaded",
			Config{
				ProducerCompressionCodec: "none",
				TLSCrtPath:               "../testdata/fake-crt.pem",
				TLSKeyPath:               "../testdata/fake-key.pem",
				TLSCaCrtPath:             "../testdata/fake-ca.pem",
			},
			func(t *testing.T, cfg *sarama.Config) {
				assert.NotNil(t, cfg.Net.TLS.Config.RootCAs)
				assert.False(t, cfg.Net.TLS.Config.InsecureSkipVerify)
			},
			false,
		}, {
			"error loading TLS certs returns an error",
			Config{
				ProducerCompressionCodec: "none",
				TLSCrtPath:               "../testdata/bad-path.pem",
				TLSKeyPath:               "../testdata/bad-path.pem",
			},
			func(*testing.T, *sarama.Config) {},
			true,
		}, {
			"error loading TLS CA cert returns an error",
			Config{
				ProducerCompressionCodec: "none",
				TLSCrtPath:               "../testdata/fake-crt.pem",
				TLSKeyPath:               "../testdata/fake-key.pem",
				TLSCaCrtPath:             "../testdata/bad-path",
			},
			func(*testing.T, *sarama.Config) {},
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.input.populateSaramaConfig(context.Background())
			if test.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			// make sure the partitioner and metrics registry were created for every test
			assert.NotNil(t, test.input.MetricRegistry)
			assert.NotNil(t, test.input.Producer.Partitioner)
			test.check(t, &test.input.Config)
		})
	}
}
