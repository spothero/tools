// Copyright 2023 SpotHero
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
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rcrowley/go-metrics"
	"github.com/spothero/tools/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewClient creates a new Sarama Client from the tools configuration. Using this version of NewClient
// enables setting of Sarama configuration from the CLI and environment variables. In addition, this method has
// the side effect of running a periodic task to collect prometheus from the Sarama internal metrics registry.
// Calling the cancel function associated with the provided context stops the periodic metrics collection.
// Note that this method overrides the Producer.Return.Successes, Producer.Return.Errors, and Consumer.Return.Errors
// and sets them all to true since those options are required for metrics collection and tracing. Code that
// uses the client generated by this method must handle those cases appropriately.
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
	c.newClientMetrics().startUpdating(ctx, c.MetricsFrequency)
	return client, nil
}

// populateSaramaConfig adds values to the sarama config that either need to be parsed from flags
// or need to be specified by the caller
func (c *Config) populateSaramaConfig(ctx context.Context) error {
	if c.TLSCrtPath != "" && c.TLSKeyPath != "" {
		cert, err := tls.LoadX509KeyPair(c.TLSCrtPath, c.TLSKeyPath)
		if err != nil {
			return fmt.Errorf("failed to load Kafka TLS key pair: %w", err)
		}
		c.Net.TLS.Config = &tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: true,
		}
		c.Net.TLS.Enable = true
		if c.TLSCaCrtPath != "" {
			caCert, readErr := os.ReadFile(c.TLSCaCrtPath)
			if readErr != nil {
				return fmt.Errorf("failed to load Kafka CA certificate: %w", readErr)
			}
			if len(caCert) > 0 {
				caCertPool := x509.NewCertPool()
				caCertPool.AppendCertsFromPEM(caCert)
				c.Net.TLS.Config.RootCAs = caCertPool
				c.Net.TLS.Config.InsecureSkipVerify = false
			}
		}

	}
	c.Producer.RequiredAcks = sarama.RequiredAcks(c.ProducerRequiredAcks)
	if c.ProducerCompressionCodec != "" {
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
	}
	if c.KafkaVersion != "" {
		kafkaVersion, err := sarama.ParseKafkaVersion(c.KafkaVersion)
		if err != nil {
			return err
		}
		c.Version = kafkaVersion
	}
	// creating a standard logger can only fail if an invalid error level is supplied which
	// will never be the case here
	if c.Verbose {
		saramaLogger, _ := zap.NewStdLogAt(log.Get(ctx).Named("sarama"), zapcore.InfoLevel)
		sarama.Logger = saramaLogger
	}

	// set options that cannot be set by flags
	c.Producer.Partitioner = sarama.NewReferenceHashPartitioner
	c.MetricRegistry = metrics.NewRegistry()
	c.Producer.Return.Successes = true
	c.Producer.Return.Errors = true
	c.Consumer.Return.Errors = true

	return nil
}

// clientMetrics collects metrics from the sarama internal metrics registry and publishes them as prometheus metrics
type clientMetrics struct {
	labels     prometheus.Labels
	registry   metrics.Registry
	registerer prometheus.Registerer
	gauges     map[string]*prometheus.GaugeVec
	mutex      *sync.RWMutex
}

func (c Config) newClientMetrics() clientMetrics {
	return clientMetrics{
		labels:     prometheus.Labels{"broker": strings.Join(c.BrokerAddrs, ","), "client": c.ClientID},
		registry:   c.MetricRegistry,
		registerer: c.Registerer,
		gauges:     make(map[string]*prometheus.GaugeVec),
		mutex:      &sync.RWMutex{},
	}
}

// updateOnce pulls metrics from the registry and translates them to prometheus metrics, dynamically creating
// prometheus metrics from the internal metrics registry
func (m clientMetrics) updateOnce(ctx context.Context) {
	m.registry.Each(func(name string, i interface{}) {
		var metricVal float64
		switch metric := i.(type) {
		// Sarama only collects meters, histograms, and counters
		case metrics.Meter:
			metricVal = metric.Snapshot().Rate1()
		case metrics.Histogram:
			// Prometheus histograms are incompatible with go-metrics histograms
			// so just get the last value for use in gauge
			histValues := metric.Snapshot().Sample().Values()
			if len(histValues) > 0 {
				metricVal = float64(histValues[len(histValues)-1])
			}
		case metrics.Counter:
			metricVal = float64(metric.Snapshot().Count())
		default:
			log.Get(context.Background()).Warn(
				"unknown metric type found while exporting sarama metrics",
				zap.String("type", reflect.TypeOf(metric).String()))
			return
		}
		promMetricName := strings.Replace(name, "-", "_", -1)
		m.mutex.RLock()
		gauge, ok := m.gauges[promMetricName]
		m.mutex.RUnlock()
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
			m.mutex.Lock()
			if err := m.registerer.Register(gauge); err != nil {
				log.Get(ctx).Error("error registering sarama metric", zap.Error(err))
				// add a nil entry to the map so that the error doesn't continually show up in the logs on each
				// subsequent iteration
				m.gauges[promMetricName] = nil
			} else {
				m.gauges[promMetricName] = gauge
			}
			m.mutex.Unlock()
		}
		if gauge != nil {
			gauge.With(m.labels).Set(metricVal)
		}
	})
}

// startUpdating collects metrics from the registry at the specified frequency. Calling the cancel associated
// with the provided context stops the collection.
func (m clientMetrics) startUpdating(ctx context.Context, frequency time.Duration) {
	ticker := time.NewTicker(frequency)
	go func() {
		for {
			select {
			case <-ticker.C:
				m.updateOnce(ctx)
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}
