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

// Consumer is a drop-in replacement for the sarama consumer that adds
// Prometheus metrics on the number of messages read and errors received.
// This consumer implementation creates the drop-in PartitionConsumer from
// this package.
type Consumer struct {
	sarama.Consumer
	metrics ConsumerMetrics
}

// PartitionConsumer is a drop-in replacement for the sarama partition consumer.
type PartitionConsumer struct {
	sarama.PartitionConsumer
	messages chan *sarama.ConsumerMessage
	errors   chan *sarama.ConsumerError
	metrics  ConsumerMetrics
	closeWg  *sync.WaitGroup
}

// NewConsumerFromClient creates a new Consumer from a sarama Client with the
// given set of metrics. Note that error metrics can only be collected if the
// consumer is configured to return errors.
func NewConsumerFromClient(client sarama.Client, metrics ConsumerMetrics) (Consumer, error) {
	c, err := sarama.NewConsumerFromClient(client)
	if err != nil {
		return Consumer{}, err
	}
	return Consumer{
		Consumer: c,
		metrics:  metrics,
	}, nil
}

// ConsumePartition creates a wrapped PartitionConsumer on the given topic and
// partition starting at the given offset.
func (c Consumer) ConsumePartition(topic string, partition int32, offset int64) (sarama.PartitionConsumer, error) {
	partitionConsumer, err := c.Consumer.ConsumePartition(topic, partition, offset)
	if err != nil {
		return nil, err
	}
	pc := PartitionConsumer{
		PartitionConsumer: partitionConsumer,
		messages:          make(chan *sarama.ConsumerMessage, cap(partitionConsumer.Messages())),
		errors:            make(chan *sarama.ConsumerError, cap(partitionConsumer.Errors())),
		metrics:           c.metrics,
		closeWg:           &sync.WaitGroup{},
	}
	pc.run(topic, partition)
	return pc, nil
}

// run listens to the messages and errors topic from the underlying Sarama
// consumer and collects metrics before forwarding the messages and errors
// through.
func (pc PartitionConsumer) run(topic string, partition int32) {
	labels := prometheus.Labels{
		"topic":     topic,
		"partition": fmt.Sprintf("%d", partition),
	}
	pc.closeWg.Add(2)
	go func() {
		for msg := range pc.PartitionConsumer.Messages() {
			pc.metrics.messagesConsumed.With(labels).Inc()
			pc.messages <- msg
		}
		pc.closeWg.Done()
	}()
	go func() {
		for err := range pc.PartitionConsumer.Errors() {
			pc.metrics.errorsConsumed.With(labels).Inc()
			pc.errors <- err
		}
		pc.closeWg.Done()
	}()
}

// Messages returns the read channel for the messages returned by the broker
func (pc PartitionConsumer) Messages() <-chan *sarama.ConsumerMessage {
	return pc.messages
}

// Errors returns the read channel of errors that occurred during consumption
// if ConsumerReturnErrors was true when configuring the client.
func (pc PartitionConsumer) Errors() <-chan *sarama.ConsumerError {
	return pc.errors
}

// AsyncClose initiates a shutdown of the PartitionConsumer. This method
// returns immediately. Once the consumer is shutdown, the message and
// error channels are closed.
func (pc PartitionConsumer) AsyncClose() {
	go func() {
		pc.closeWg.Wait()
		close(pc.messages)
		close(pc.errors)
	}()
	pc.PartitionConsumer.AsyncClose()
}

// Close synchronously shuts down the partition consumer and returns any
// outstanding errors.
func (pc PartitionConsumer) Close() error {
	err := pc.PartitionConsumer.Close()
	close(pc.messages)
	close(pc.errors)
	return err
}

// ConsumerMetrics is a collection of Prometheus metrics for tracking a Kafka consumer's performance
type ConsumerMetrics struct {
	messagesConsumed *prometheus.GaugeVec
	errorsConsumed   *prometheus.GaugeVec
}

// NewConsumerMetrics creates and registers metrics for the Kafka Consumer with
// the provided prometheus registerer
func NewConsumerMetrics(registerer prometheus.Registerer) (ConsumerMetrics, error) {
	labels := []string{"topic", "partition"}
	metrics := ConsumerMetrics{
		messagesConsumed: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "kafka_messages_consumed",
				Help: "Number of Kafka messages processed by the consumer",
			},
			labels,
		),
		errorsConsumed: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "kafka_errors_consumed",
				Help: "Number of consumer errors received from the Kafka broker",
			},
			labels,
		),
	}
	if err := registerer.Register(metrics.messagesConsumed); err != nil {
		return ConsumerMetrics{}, err
	}
	if err := registerer.Register(metrics.errorsConsumed); err != nil {
		return ConsumerMetrics{}, err
	}
	return metrics, nil
}
