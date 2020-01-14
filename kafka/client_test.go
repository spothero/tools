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
	"time"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
)

func TestConfig_populateSaramaConfig(t *testing.T) {
	tests := []struct {
		name      string
		input     Config
		expected  sarama.Config
		expectErr bool
	}{
		{
			"basic configuration is populated",
			Config{
				Config:                   sarama.Config{},
				ConsumerReturnErrors:     true,
				ProducerReturnErrors:     true,
				ProducerReturnSuccesses:  true,
				Verbose:                  true,
				KafkaVersion:             "2.3.0",
				ProducerRequiredAcks:     1,
				ProducerCompressionCodec: "none",
			},
			sarama.Config{
				Producer: struct {
					MaxMessageBytes  int
					RequiredAcks     sarama.RequiredAcks
					Timeout          time.Duration
					Compression      sarama.CompressionCodec
					CompressionLevel int
					Partitioner      sarama.PartitionerConstructor
					Idempotent       bool
					Return           struct {
						Successes bool
						Errors    bool
					}
					Flush struct {
						Bytes       int
						Messages    int
						Frequency   time.Duration
						MaxMessages int
					}
					Retry struct {
						Max         int
						Backoff     time.Duration
						BackoffFunc func(retries int, maxRetries int) time.Duration
					}
				}{
					RequiredAcks: sarama.WaitForLocal,
					Compression:  sarama.CompressionNone,
					Return: struct {
						Successes bool
						Errors    bool
					}{
						Successes: true, Errors: true,
					},
				},
				Consumer: struct {
					Group struct {
						Session   struct{ Timeout time.Duration }
						Heartbeat struct{ Interval time.Duration }
						Rebalance struct {
							Strategy sarama.BalanceStrategy
							Timeout  time.Duration
							Retry    struct {
								Max     int
								Backoff time.Duration
							}
						}
						Member struct{ UserData []byte }
					}
					Retry struct {
						Backoff     time.Duration
						BackoffFunc func(retries int) time.Duration
					}
					Fetch struct {
						Min     int32
						Default int32
						Max     int32
					}
					MaxWaitTime       time.Duration
					MaxProcessingTime time.Duration
					Return            struct{ Errors bool }
					Offsets           struct {
						CommitInterval time.Duration
						Initial        int64
						Retention      time.Duration
						Retry          struct{ Max int }
					}
					IsolationLevel sarama.IsolationLevel
				}{
					Return: struct{ Errors bool }{Errors: true},
				},
				Version: sarama.V2_3_0_0,
				Admin:   struct{ Timeout time.Duration }{Timeout: 3 * time.Second},
			},
			false,
		}, {
			"bad version returns an error",
			Config{KafkaVersion: "not.a.real.version"},
			sarama.Config{},
			true,
		}, {
			"zstd compression is properly set",
			Config{ProducerCompressionCodec: "zstd", KafkaVersion: "2.3.0"},
			sarama.Config{
				Producer: struct {
					MaxMessageBytes  int
					RequiredAcks     sarama.RequiredAcks
					Timeout          time.Duration
					Compression      sarama.CompressionCodec
					CompressionLevel int
					Partitioner      sarama.PartitionerConstructor
					Idempotent       bool
					Return           struct {
						Successes bool
						Errors    bool
					}
					Flush struct {
						Bytes       int
						Messages    int
						Frequency   time.Duration
						MaxMessages int
					}
					Retry struct {
						Max         int
						Backoff     time.Duration
						BackoffFunc func(retries int, maxRetries int) time.Duration
					}
				}{
					Compression: sarama.CompressionZSTD,
				},
				Version: sarama.V2_3_0_0,
				Admin:   struct{ Timeout time.Duration }{Timeout: 3 * time.Second},
			},
			false,
		}, {
			"snappy compression is properly set",
			Config{ProducerCompressionCodec: "snappy", KafkaVersion: "2.3.0"},
			sarama.Config{
				Producer: struct {
					MaxMessageBytes  int
					RequiredAcks     sarama.RequiredAcks
					Timeout          time.Duration
					Compression      sarama.CompressionCodec
					CompressionLevel int
					Partitioner      sarama.PartitionerConstructor
					Idempotent       bool
					Return           struct {
						Successes bool
						Errors    bool
					}
					Flush struct {
						Bytes       int
						Messages    int
						Frequency   time.Duration
						MaxMessages int
					}
					Retry struct {
						Max         int
						Backoff     time.Duration
						BackoffFunc func(retries int, maxRetries int) time.Duration
					}
				}{
					Compression: sarama.CompressionSnappy,
				},
				Version: sarama.V2_3_0_0,
				Admin:   struct{ Timeout time.Duration }{Timeout: 3 * time.Second},
			},
			false,
		}, {
			"lz4 compression is properly set",
			Config{ProducerCompressionCodec: "lz4", KafkaVersion: "2.3.0"},
			sarama.Config{
				Producer: struct {
					MaxMessageBytes  int
					RequiredAcks     sarama.RequiredAcks
					Timeout          time.Duration
					Compression      sarama.CompressionCodec
					CompressionLevel int
					Partitioner      sarama.PartitionerConstructor
					Idempotent       bool
					Return           struct {
						Successes bool
						Errors    bool
					}
					Flush struct {
						Bytes       int
						Messages    int
						Frequency   time.Duration
						MaxMessages int
					}
					Retry struct {
						Max         int
						Backoff     time.Duration
						BackoffFunc func(retries int, maxRetries int) time.Duration
					}
				}{
					Compression: sarama.CompressionLZ4,
				},
				Version: sarama.V2_3_0_0,
				Admin:   struct{ Timeout time.Duration }{Timeout: 3 * time.Second},
			},
			false,
		}, {
			"gzip compression is properly set",
			Config{ProducerCompressionCodec: "gzip", KafkaVersion: "2.3.0"},
			sarama.Config{
				Producer: struct {
					MaxMessageBytes  int
					RequiredAcks     sarama.RequiredAcks
					Timeout          time.Duration
					Compression      sarama.CompressionCodec
					CompressionLevel int
					Partitioner      sarama.PartitionerConstructor
					Idempotent       bool
					Return           struct {
						Successes bool
						Errors    bool
					}
					Flush struct {
						Bytes       int
						Messages    int
						Frequency   time.Duration
						MaxMessages int
					}
					Retry struct {
						Max         int
						Backoff     time.Duration
						BackoffFunc func(retries int, maxRetries int) time.Duration
					}
				}{
					Compression: sarama.CompressionGZIP,
				},
				Version: sarama.V2_3_0_0,
				Admin:   struct{ Timeout time.Duration }{Timeout: 3 * time.Second},
			},
			false,
		}, {
			"unknown compression returns an error",
			Config{ProducerCompressionCodec: "beepboop", KafkaVersion: "2.3.0"},
			sarama.Config{},
			true,
		}, {
			"TLS configuration is loaded",
			Config{
				ProducerCompressionCodec: "none",
				KafkaVersion:             "2.3.0",
				TLSCrtPath:               "../testdata/fake-crt.pem",
				TLSKeyPath:               "../testdata/fake-key.pem",
			},
			sarama.Config{Version: sarama.V2_3_0_0, Admin: struct{ Timeout time.Duration }{Timeout: 3 * time.Second}},
			false,
		}, {
			"TLS CA cert is loaded",
			Config{
				ProducerCompressionCodec: "none",
				KafkaVersion:             "2.3.0",
				TLSCrtPath:               "../testdata/fake-crt.pem",
				TLSKeyPath:               "../testdata/fake-key.pem",
				TLSCaCrtPath:             "../testdata/fake-ca.pem",
			},
			sarama.Config{Version: sarama.V2_3_0_0, Admin: struct{ Timeout time.Duration }{Timeout: 3 * time.Second}},
			false,
		}, {
			"error loading TLS certs returns an error",
			Config{
				ProducerCompressionCodec: "none",
				KafkaVersion:             "2.3.0",
				TLSCrtPath:               "../testdata/bad-path.pem",
				TLSKeyPath:               "../testdata/bad-path.pem",
			},
			sarama.Config{},
			true,
		}, {
			"error loading TLS CA cert returns an error",
			Config{
				ProducerCompressionCodec: "none",
				KafkaVersion:             "2.3.0",
				TLSCrtPath:               "../testdata/fake-crt.pem",
				TLSKeyPath:               "../testdata/fake-key.pem",
				TLSCaCrtPath:             "../testdata/bad-path",
			},
			sarama.Config{},
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
			// make sure the partitioner and metrics registry were created but then remove them for the
			// configuration equality comparison
			assert.NotNil(t, test.input.MetricRegistry)
			test.input.MetricRegistry = nil
			assert.NotNil(t, test.input.Producer.Partitioner)
			test.input.Producer.Partitioner = nil

			// if TLS was set, just make sure the certificates go loaded then reset for the equality comparison
			if test.input.TLSKeyPath != "" {
				assert.True(t, test.input.Net.TLS.Enable)
				assert.NotNil(t, test.input.Net.TLS.Config)
			}
			if test.input.TLSCaCrtPath != "" {
				assert.False(t, test.input.Net.TLS.Config.InsecureSkipVerify)
				assert.NotNil(t, test.input.Net.TLS.Config.RootCAs)
			}
			test.input.Net.TLS.Config = nil
			test.input.Net.TLS.Enable = false

			assert.GreaterOrEqual(t, test.input.Admin.Timeout.Microseconds(), int64(0))

			assert.Equal(t, test.expected, test.input.Config)
		})
	}
}
