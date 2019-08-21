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
	"fmt"
	"sync"

	"github.com/Shopify/sarama"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// ProducerMetrics is a collection of Prometheus metrics for tracking a Kafka producer's performance
type ProducerMetrics struct {
	MessagesProduced *prometheus.GaugeVec
	ErrorsProduced   *prometheus.GaugeVec
}

// ProducerConfig contains producer-specific configuration. At present, there are no producer-specific
// configuration options.
type ProducerConfig struct{}

// Producer is a wrapped Sarama producer that tracks producer metrics and provides optional logging.
type Producer struct {
	metrics   ProducerMetrics
	client    Client
	producer  sarama.AsyncProducer
	successes chan *sarama.ProducerMessage
	errors    chan *sarama.ProducerError
	logger    *zap.Logger
}

// ProducerIface is an interface for producing Kafka messages
type ProducerIface interface {
	RunProducer(messages chan *sarama.ProducerMessage, done chan bool)
	Successes() chan *sarama.ProducerMessage
	Errors() chan *sarama.ProducerError
}

// NewProducer creates a sarama producer from a client. If the returnMessages flag is true,
// messages from the producer will be produced on the Success or Errors channel depending
// on the outcome of the produced message.
func (p ProducerConfig) NewProducer(client Client, logger *zap.Logger, returnMessages bool) (ProducerIface, error) {
	saramaProducer, err := sarama.NewAsyncProducerFromClient(client.SaramaClient)
	if err != nil {
		return Producer{}, err
	}
	producer := Producer{
		client:   client,
		producer: saramaProducer,
		metrics:  RegisterProducerMetrics(prometheus.DefaultRegisterer),
	}
	if returnMessages {
		producer.successes = make(chan *sarama.ProducerMessage)
		producer.errors = make(chan *sarama.ProducerError)
	}
	if logger != nil {
		producer.logger = logger
	} else {
		producer.logger = zap.NewNop()
	}
	return producer, nil
}

func RegisterProducerMetrics(registerer prometheus.Registerer) ProducerMetrics {
	promLabels := []string{"topic", "partition", "client"}
	p := ProducerMetrics{
		MessagesProduced: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "kafka_messages_produced",
				Help: "Number of Kafka messages produced",
			},
			promLabels,
		),
		ErrorsProduced: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "kafka_errors_produced",
				Help: "Number of Kafka errors produced",
			},
			promLabels,
		),
	}
	registerer.MustRegister(p.MessagesProduced, p.ErrorsProduced)
	return p
}

// RunProducer wraps the sarama AsyncProducer and adds metrics, logging, and a shutdown procedure
// to the producer. To stop the producer, close the messages channel; when the producer is shutdown a signal will
// be emitted on the done channel. If the messages channel is unbuffered, each message sent to the producer is
// guaranteed to at least have been attempted to be produced to Kafka.
func (p Producer) RunProducer(messages chan *sarama.ProducerMessage, done chan bool) {
	promLabels := prometheus.Labels{
		"client": p.client.ClientID,
	}
	var closeWg sync.WaitGroup
	closeWg.Add(2) // 1 for success, error channels

	// Handle producer messages
	go func() {
		defer func() {
			// channel closed, initiate producer shutdown
			p.logger.Debug("closing kafka producer")
			// wait for error and successes channels to close
			p.producer.AsyncClose()
			closeWg.Wait()
			p.logger.Debug("kafka producer closed")
			done <- true
		}()
		for message := range messages {
			p.producer.Input() <- message
		}
	}()

	// Handle errors returned by the producer
	go func() {
		defer closeWg.Done()
		if p.errors != nil {
			defer close(p.errors)
		}
		for err := range p.producer.Errors() {
			var key []byte
			if err.Msg.Key != nil {
				if _key, err := err.Msg.Key.Encode(); err == nil {
					key = _key
				} else {
					p.logger.Error("could not encode produced message key", zap.Error(err))
				}
			}
			p.logger.Error(
				"Error producing Kafka message",
				zap.String("topic", err.Msg.Topic),
				zap.String("key", string(key)),
				zap.Int32("partition", err.Msg.Partition),
				zap.Int64("offset", err.Msg.Offset),
				zap.Error(err))
			promLabels["partition"] = fmt.Sprintf("%d", err.Msg.Partition)
			promLabels["topic"] = err.Msg.Topic
			p.metrics.ErrorsProduced.With(promLabels).Add(1)
			if p.errors != nil {
				p.errors <- err
			}
		}
	}()

	// Handle successes returned by the producer
	go func() {
		defer closeWg.Done()
		if p.successes != nil {
			defer close(p.successes)
		}
		for msg := range p.producer.Successes() {
			promLabels["partition"] = fmt.Sprintf("%d", msg.Partition)
			promLabels["topic"] = msg.Topic
			p.metrics.MessagesProduced.With(promLabels).Add(1)
			if p.successes != nil {
				p.successes <- msg
			}
		}
	}()
}

// Successes returns the channel on which successfully published messages will be returned
func (p Producer) Successes() chan *sarama.ProducerMessage {
	return p.successes
}

// Errors returns the channel on which messages that could not be published will be returned
func (p Producer) Errors() chan *sarama.ProducerError {
	return p.errors
}
