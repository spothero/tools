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

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rcrowley/go-metrics"
	"github.com/spf13/pflag"
	"github.com/spothero/tools/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/xerrors"
)

// ClientConfig contains connection settings and configuration for communicating with a Kafka cluster
type ClientConfig struct {
	Broker                   string
	ClientID                 string
	TLSCaCrtPath             string
	TLSCrtPath               string
	TLSKeyPath               string
	Verbose                  bool
	KafkaVersion             string
	ProducerCompressionCodec string
	ProducerCompressionLevel int
}

// Registers Kafka client flags with pflags
func (c *ClientConfig) RegisterFlags(flags *pflag.FlagSet) {
	flags.StringVarP(&c.Broker, "kafka-broker", "b", "kafka:29092", "Kafka broker Address")
	flags.StringVar(&c.ClientID, "kafka-client-id", "client", "Kafka consumer Client ID")
	flags.StringVar(&c.TLSCaCrtPath, "kafka-server-ca-crt-path", "", "Kafka Server TLS CA Certificate Path")
	flags.StringVar(&c.TLSCrtPath, "kafka-client-crt-path", "", "Kafka Client TLS Certificate Path")
	flags.StringVar(&c.TLSKeyPath, "kafka-client-key-path", "", "Kafka Client TLS Key Path")
	flags.BoolVar(&c.Verbose, "kafka-verbose", false, "When this flag is set Kafka will log verbosely")
	flags.StringVar(&c.KafkaVersion, "kafka-version", "2.1.0", "Kafka broker version")
	flags.StringVar(&c.ProducerCompressionCodec, "kafka-producer-compression-codec", "none", "Compression codec to use when producing messages, one of: \"none\", \"zstd\", \"snappy\", \"lz4\", \"zstd\", \"gzip\"")
	flags.IntVar(&c.ProducerCompressionLevel, "kafka-producer-compression-level", -1000, "Compression level to use on produced messages, -1000 signifies to use the default level.")
}

// Client wraps a sarama client and Kafka configuration and can be used to create producers and consumers
type Client struct {
	ClientConfig
	SaramaClient  sarama.Client
	metricsCancel context.CancelFunc
}

// NewClient creates a Sarama client from configuration and starts a periodic task for capturing
// Kafka broker metrics.
func (c ClientConfig) NewClient(ctx context.Context) (Client, error) {
	if c.Verbose {
		saramaLogger, err := zap.NewStdLogAt(log.Get(ctx).Named("sarama"), zapcore.InfoLevel)
		if err != nil {
			return Client{}, xerrors.Errorf("verbose was requested but failed to create zap standard logger: %w", err)
		}
		sarama.Logger = saramaLogger
	}
	kafkaConfig := sarama.NewConfig()
	kafkaVersion, err := sarama.ParseKafkaVersion(c.KafkaVersion)
	if err != nil {
		return Client{}, err
	}
	kafkaConfig.Version = kafkaVersion
	kafkaConfig.Consumer.Return.Errors = true
	kafkaConfig.ClientID = c.ClientID
	kafkaConfig.Producer.RequiredAcks = sarama.WaitForAll
	kafkaConfig.Producer.Return.Successes = true
	kafkaConfig.Producer.Return.Errors = true
	var compressionCodec sarama.CompressionCodec
	switch c.ProducerCompressionCodec {
	case "zstd":
		compressionCodec = sarama.CompressionZSTD
	case "snappy":
		compressionCodec = sarama.CompressionSnappy
	case "lz4":
		compressionCodec = sarama.CompressionLZ4
	case "gzip":
		compressionCodec = sarama.CompressionGZIP
	case "none":
		compressionCodec = sarama.CompressionNone
	default:
		return Client{}, fmt.Errorf("unknown compression codec %v", c.ProducerCompressionCodec)
	}
	kafkaConfig.Producer.Compression = compressionCodec
	kafkaConfig.Producer.CompressionLevel = c.ProducerCompressionLevel

	if c.TLSCrtPath != "" && c.TLSKeyPath != "" {
		cer, err := tls.LoadX509KeyPair(c.TLSCrtPath, c.TLSKeyPath)
		if err != nil {
			return Client{}, xerrors.Errorf("failed to load Kafka server TLS key pair: %w", err)
		}
		kafkaConfig.Net.TLS.Config = &tls.Config{
			Certificates:       []tls.Certificate{cer},
			InsecureSkipVerify: true,
		}
		kafkaConfig.Net.TLS.Config.BuildNameToCertificate()
		kafkaConfig.Net.TLS.Enable = true

		if c.TLSCaCrtPath != "" {
			caCert, err := ioutil.ReadFile(c.TLSCaCrtPath)
			if err != nil {
				return Client{}, xerrors.Errorf("failed to load Kafka server CA certificate: %w", err)
			}
			if len(caCert) > 0 {
				caCertPool := x509.NewCertPool()
				caCertPool.AppendCertsFromPEM(caCert)
				kafkaConfig.Net.TLS.Config.RootCAs = caCertPool
				kafkaConfig.Net.TLS.Config.InsecureSkipVerify = false
			}
		}
	}

	saramaClient, err := sarama.NewClient([]string{c.Broker}, kafkaConfig)
	if err != nil {
		return Client{}, xerrors.Errorf("failed to create Kafka client: %w", err)
	}

	// Export metrics from Sarama's metrics registry to Prometheus
	ctx, cancel := context.WithCancel(ctx)
	kafkaConfig.MetricRegistry = metrics.NewRegistry()
	c.recordBrokerMetrics(ctx, 500*time.Millisecond, kafkaConfig.MetricRegistry)

	return Client{
		ClientConfig:  c,
		SaramaClient:  saramaClient,
		metricsCancel: cancel,
	}, nil
}

// Close the underlying Kafka client and stop the Kafka broker metrics gathering task. If an error occurs closing
// the client, the error is logged.
func (c Client) Close(ctx context.Context) {
	c.metricsCancel()
	if err := c.SaramaClient.Close(); err != nil {
		log.Get(ctx).Error("Error closing Kafka client", zap.Error(err))
	}
}

// map the metrics from Sarama's metrics registry to Prometheus metrics
func (c ClientConfig) updateBrokerMetrics(registry metrics.Registry, gauges map[string]*prometheus.GaugeVec) {
	registry.Each(func(name string, i interface{}) {
		var metricVal float64
		switch metric := i.(type) {
		// Sarama only collects meters and histograms
		case metrics.Meter:
			metricVal = metric.Snapshot().Rate1()
		case metrics.Histogram:
			// Prometheus histograms are incompatible with go-metrics histograms
			// so just get the last value for use in gauge
			histValues := metric.Snapshot().Sample().Values()
			if len(histValues) > 0 {
				metricVal = float64(histValues[len(histValues)-1])
			}
		default:
			log.Get(context.Background()).Warn(
				"Unknown metric type found while exporting Sarama metrics",
				zap.String("type", reflect.TypeOf(metric).String()))
			return
		}
		promMetricName := strings.Replace(name, "-", "_", -1)
		gauge, ok := gauges[promMetricName]
		if !ok {
			// We haven't seen this gauge before; create it
			gauge = prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: "sarama",
					Name:      promMetricName,
					Help:      name,
				},
				[]string{"broker", "client"},
			)
			prometheus.MustRegister(gauge)
			gauges[promMetricName] = gauge
		}
		gauge.With(prometheus.Labels{"broker": c.Broker, "client": c.ClientID}).Set(metricVal)
	})
}

// run a periodic task until the context is canceled that updates broker metrics
func (c ClientConfig) recordBrokerMetrics(
	ctx context.Context,
	updateInterval time.Duration,
	registry metrics.Registry,
) {
	ticker := time.NewTicker(updateInterval)
	gauges := make(map[string]*prometheus.GaugeVec)
	go func() {
		for {
			select {
			case <-ticker.C:
				c.updateBrokerMetrics(registry, gauges)
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}
