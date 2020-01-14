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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/Shopify/sarama"
	prometheusmetrics "github.com/deathowl/go-metrics-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rcrowley/go-metrics"
	"github.com/spothero/tools/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewClient creates a new Sarama Client from the tools configuration. Using this version of NewClient
// enables setting of Sarama configuration from the CLI and environment variables. In addition, this method has
// the side effect of running a periodic task to collect prometheus from the Sarama internal metrics registry.
func (c *Config) NewClient(ctx context.Context) (sarama.Client, error) {
	if err := c.populateSaramaConfig(ctx); err != nil {
		return nil, err
	}
	client, err := sarama.NewClient(c.BrokerAddrs, &c.Config)
	if err != nil {
		return nil, err
	}
	if c.Registerer == nil {
		c.Registerer = prometheus.DefaultRegisterer
	}
	go prometheusmetrics.NewPrometheusProvider(
		c.MetricRegistry, "sarama", "", c.Registerer, c.MetricsFrequency,
	).UpdatePrometheusMetrics()
	return client, nil
}

// populateSaramaConfig adds values to the sarama config that either need to be parsed from flags
// or need to be specified by the caller
func (c *Config) populateSaramaConfig(ctx context.Context) error {
	if c.Verbose {
		// creating a standard logger can only fail if an invalid error level is supplied which
		// will never be the case here
		saramaLogger, _ := zap.NewStdLogAt(log.Get(ctx).Named("sarama"), zapcore.InfoLevel)
		sarama.Logger = saramaLogger
	}
	// set options that cannot be set by flags in a way that matches sarama.NewConfig
	c.Producer.Partitioner = sarama.NewReferenceHashPartitioner
	c.MetricRegistry = metrics.NewRegistry()
	c.Producer.Return.Successes = c.ProducerReturnSuccesses
	c.Producer.Return.Errors = c.ProducerReturnErrors
	c.Consumer.Return.Errors = c.ConsumerReturnErrors

	// If the admin flags weren't registered, the admin timeout is 0 and that is not allowed
	if c.Admin.Timeout == 0 {
		c.Admin.Timeout = 3 * time.Second
	}

	// parse options that need to be parsed
	kafkaVersion, err := sarama.ParseKafkaVersion(c.KafkaVersion)
	if err != nil {
		return err
	}
	c.Version = kafkaVersion
	c.Producer.RequiredAcks = sarama.RequiredAcks(c.ProducerRequiredAcks)
	switch c.ProducerCompressionCodec {
	case "zstd":
		c.Producer.Compression = sarama.CompressionZSTD
	case "snappy":
		c.Producer.Compression = sarama.CompressionSnappy
	case "lz4":
		c.Producer.Compression = sarama.CompressionLZ4
	case "gzip":
		c.Producer.Compression = sarama.CompressionGZIP
	case "none":
		c.Producer.Compression = sarama.CompressionNone
	default:
		return fmt.Errorf("unknown compression codec %v provided", c.ProducerCompressionCodec)
	}

	// load TLS configs if cert paths provided
	if c.TLSCrtPath != "" && c.TLSKeyPath != "" {
		cert, err := tls.LoadX509KeyPair(c.TLSCrtPath, c.TLSKeyPath)
		if err != nil {
			return fmt.Errorf("failed to load Kafka TLS key pair: %w", err)
		}
		c.Net.TLS.Config = &tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: true,
		}
		c.Net.TLS.Config.BuildNameToCertificate()
		c.Net.TLS.Enable = true
		if c.TLSCaCrtPath != "" {
			caCert, err := ioutil.ReadFile(c.TLSCaCrtPath)
			if err != nil {
				return fmt.Errorf("failed to load Kafka CA certificate: %w", err)
			}
			if len(caCert) > 0 {
				caCertPool := x509.NewCertPool()
				caCertPool.AppendCertsFromPEM(caCert)
				c.Net.TLS.Config.RootCAs = caCertPool
				c.Net.TLS.Config.InsecureSkipVerify = false
			}
		}
	}
	return nil
}
