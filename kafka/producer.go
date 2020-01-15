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
	"fmt"
	"sync"

	"github.com/Shopify/sarama"
	"github.com/prometheus/client_golang/prometheus"
)

// AsyncProducer is a drop-in replacement for the sarama AsyncProducer that
// adds Prometheus metrics on its performance.
type AsyncProducer struct {
	sarama.AsyncProducer
	metrics       ProducerMetrics
	successes     chan *sarama.ProducerMessage
	errors        chan *sarama.ProducerError
	asyncShutdown chan bool
	closeWg       *sync.WaitGroup
}

// NewAsyncProducerFromClient creates a new AsyncProducer from a sarama Client
// that pushes metrics to the provided ProducerMetrics. Note that metrics are
// only collected if the producer is configured to return successes and errors.
func NewAsyncProducerFromClient(client sarama.Client, metrics ProducerMetrics) (AsyncProducer, error) {
	p, err := sarama.NewAsyncProducerFromClient(client)
	if err != nil {
		return AsyncProducer{}, err
	}
	ap := AsyncProducer{
		AsyncProducer: p,
		metrics:       metrics,
		successes:     make(chan *sarama.ProducerMessage, cap(p.Successes())),
		errors:        make(chan *sarama.ProducerError, cap(p.Errors())),
		asyncShutdown: make(chan bool),
		closeWg:       &sync.WaitGroup{},
	}
	ap.run()
	return ap, nil
}

// run runs the interceptors that collect prometheus metrics on message production.
func (ap AsyncProducer) run() {
	ap.closeWg.Add(2) // 1 for success, error channels

	// Handle errors returned by the producer
	go func() {
		for err := range ap.AsyncProducer.Errors() {
			ap.metrics.errorsProduced.With(
				prometheus.Labels{"topic": err.Msg.Topic, "partition": fmt.Sprintf("%d", err.Msg.Partition)},
			).Inc()
			ap.errors <- err
		}
		ap.closeWg.Done()
	}()

	// Handle successes returned by the producer
	go func() {
		for msg := range ap.AsyncProducer.Successes() {
			ap.metrics.messagesProduced.With(
				prometheus.Labels{"topic": msg.Topic, "partition": fmt.Sprintf("%d", msg.Partition)},
			).Inc()
			ap.successes <- msg
		}
		ap.closeWg.Done()
	}()
}

// Successes returns the output channel where successfully written
// messages will be returned if ProducerReturnSuccesses was true when
// configuring the client.
func (ap AsyncProducer) Successes() <-chan *sarama.ProducerMessage {
	return ap.successes
}

// Errors returns the output channel where errored messages will be returned
// if ProducerReturnErrors was true when configuring the client.
func (ap AsyncProducer) Errors() <-chan *sarama.ProducerError {
	return ap.errors
}

// AsyncClose triggers a shutdown of the producer. The producer will be shutdown
// when the input, errors, and successes channels are closed.
func (ap AsyncProducer) AsyncClose() {
	ap.AsyncProducer.AsyncClose()
	go func() {
		ap.closeWg.Wait()
		close(ap.errors)
		close(ap.successes)
	}()
}

// Close synchronously shuts down the producer and waits for any buffered
// messages to be flushed before returning.
func (ap AsyncProducer) Close() error {
	err := ap.AsyncProducer.Close()
	ap.closeWg.Wait()
	close(ap.errors)
	close(ap.successes)
	return err
}

// ProducerMetrics is a collection of Prometheus metrics for tracking a Kafka producer's performance
type ProducerMetrics struct {
	messagesProduced *prometheus.GaugeVec
	errorsProduced   *prometheus.GaugeVec
}

// NewProducerMetrics creates and registers metrics for the Kafka Producer
// with the provided prometheus registerer.
func NewProducerMetrics(registerer prometheus.Registerer) (ProducerMetrics, error) {
	labels := []string{"topic", "partition"}
	metrics := ProducerMetrics{
		messagesProduced: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "kafka_messages_produced",
				Help: "Number of Kafka messages produced by the producer",
			},
			labels,
		),
		errorsProduced: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "kafka_errors_produced",
				Help: "Number of errors that occurred while trying to produce a message",
			},
			labels,
		),
	}
	if err := registerer.Register(metrics.messagesProduced); err != nil {
		return ProducerMetrics{}, err
	}
	if err := registerer.Register(metrics.errorsProduced); err != nil {
		return ProducerMetrics{}, err
	}
	return metrics, nil
}
