//// Copyright 2019 SpotHero
////
//// Licensed under the Apache License, Version 2.0 (the "License");
//// you may not use this file except in compliance with the License.
//// You may obtain a copy of the License at
////
////     http://www.apache.org/licenses/LICENSE-2.0
////
//// Unless required by applicable law or agreed to in writing, software
//// distributed under the License is distributed on an "AS IS" BASIS,
//// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//// See the License for the specific language governing permissions and
//// limitations under the License.
//
package kafka

//
//import (
//	"fmt"
//	"sync"
//
//	"github.com/Shopify/sarama"
//	"github.com/prometheus/client_golang/prometheus"
//	"github.com/spf13/pflag"
//	"go.uber.org/zap"
//)
//
//// ProducerMetrics is a collection of Prometheus metrics for tracking a Kafka producer's performance
//type ProducerMetrics struct {
//	MessagesProduced *prometheus.GaugeVec
//	ErrorsProduced   *prometheus.GaugeVec
//}
//
//// Producer is a wrapped Sarama producer that tracks producer metrics and provides optional logging.
//type Producer struct {
//	metrics   ProducerMetrics
//	client    Client
//	producer  sarama.AsyncProducer
//	successes chan *sarama.ProducerMessage
//	errors    chan *sarama.ProducerError
//	logger    *zap.Logger
//}
//
//// ProducerIface is an interface for producing Kafka messages
//type ProducerIface interface {
//	RunProducer(messages chan *sarama.ProducerMessage, done chan bool)
//	Successes() chan *sarama.ProducerMessage
//	Errors() chan *sarama.ProducerError
//}
//
//// Constant values that represent the required acks setting for produced messages. These map to
//// the sarama.RequiredAcks constants
//const (
//	// RequiredAcksAll waits for all in-sync replicas to commit before responding.
//	RequiredAcksAll int = -1
//	// RequiredAcksNone waits for no acknowledgements.
//	RequiredAcksNone = 0
//	// RequiredAcksLocal waits for only the local commit to succeed before responding.
//	RequiredAcksLocal = 1
//)
//
//// ProducerConfig contains producer-specific configuration information
//type ProducerConfig struct {
//	ProducerCompressionCodec string
//	ProducerCompressionLevel int
//	ProducerRequiredAcks     int
//}
//
//// RegisterFlags registers producer flags with pflags
//func (c *ProducerConfig) RegisterFlags(flags *pflag.FlagSet) {
//	flags.StringVar(&c.ProducerCompressionCodec, "kafka-producer-compression-codec", "none", "Compression codec to use when producing messages, one of: \"none\", \"zstd\", \"snappy\", \"lz4\", \"zstd\", \"gzip\"")
//	flags.IntVar(&c.ProducerCompressionLevel, "kafka-producer-compression-level", -1000, "Compression level to use on produced messages, -1000 signifies to use the default level.")
//	flags.IntVar(&c.ProducerRequiredAcks, "kafka-producer-required-acks", -1, "Required acks setting for produced messages, -1=all, 0=none, 1=local. Default is -1.")
//}
//
//// NewProducer creates a sarama producer from a client. If the returnMessages flag is true,
//// messages from the producer will be produced on the Success or Errors channel depending
//// on the outcome of the produced message. This method also registers producer metrics with the default
//// Prometheus registerer. Note that this method has the side effect of setting the compression level and
//// codec on the provided client's underlying configuration.
//func (c Client) NewProducer(config ProducerConfig, logger *zap.Logger, returnMessages bool) (ProducerIface, error) {
//	saramaProducer, err := sarama.NewAsyncProducerFromClient(c.SaramaClient)
//	if err != nil {
//		return Producer{}, err
//	}
//	producer := Producer{
//		client:   c,
//		producer: saramaProducer,
//		metrics:  RegisterProducerMetrics(prometheus.DefaultRegisterer),
//	}
//	if returnMessages {
//		producer.successes = make(chan *sarama.ProducerMessage)
//		producer.errors = make(chan *sarama.ProducerError)
//	}
//	if logger != nil {
//		producer.logger = logger
//	} else {
//		producer.logger = zap.NewNop()
//	}
//	var compressionCodec sarama.CompressionCodec
//	switch config.ProducerCompressionCodec {
//	case "zstd":
//		compressionCodec = sarama.CompressionZSTD
//	case "snappy":
//		compressionCodec = sarama.CompressionSnappy
//	case "lz4":
//		compressionCodec = sarama.CompressionLZ4
//	case "gzip":
//		compressionCodec = sarama.CompressionGZIP
//	case "none":
//		compressionCodec = sarama.CompressionNone
//	default:
//		return Producer{}, fmt.Errorf("unknown compression codec %v", config.ProducerCompressionCodec)
//	}
//
//	var requiredAcks sarama.RequiredAcks
//	switch config.ProducerRequiredAcks {
//	case RequiredAcksAll:
//		requiredAcks = sarama.WaitForAll
//	case RequiredAcksNone:
//		requiredAcks = sarama.NoResponse
//	case RequiredAcksLocal:
//		requiredAcks = sarama.WaitForLocal
//	default:
//		return Producer{}, fmt.Errorf("unknown required acks config %v", config.ProducerRequiredAcks)
//	}
//
//	c.SaramaClient.Config().Producer.Compression = compressionCodec
//	c.SaramaClient.Config().Producer.CompressionLevel = config.ProducerCompressionLevel
//	c.SaramaClient.Config().Producer.RequiredAcks = requiredAcks
//	return producer, nil
//}
//
//// RegisterProducerMetrics registers Kafka producer metrics with the provided registerer and returns
//// a struct containing gauges for the number of messages and errors produced.
//func RegisterProducerMetrics(registerer prometheus.Registerer) ProducerMetrics {
//	promLabels := []string{"topic", "partition", "client"}
//	p := ProducerMetrics{
//		MessagesProduced: prometheus.NewGaugeVec(
//			prometheus.GaugeOpts{
//				Name: "kafka_messages_produced",
//				Help: "Number of Kafka messages produced",
//			},
//			promLabels,
//		),
//		ErrorsProduced: prometheus.NewGaugeVec(
//			prometheus.GaugeOpts{
//				Name: "kafka_errors_produced",
//				Help: "Number of Kafka errors produced",
//			},
//			promLabels,
//		),
//	}
//	registerer.MustRegister(p.MessagesProduced, p.ErrorsProduced)
//	return p
//}
//
//// RunProducer wraps the sarama AsyncProducer and adds metrics and optional logging
//// to the producer. To stop the producer, close the messages channel; when the producer is shutdown the done
//// channel will be closed. If the messages channel is unbuffered, each message sent to the producer is
//// guaranteed to at least have been attempted to be produced to Kafka.
//func (p Producer) RunProducer(messages chan *sarama.ProducerMessage, done chan bool) {
//	promLabels := prometheus.Labels{
//		"client": p.client.ClientID,
//	}
//	var closeWg sync.WaitGroup
//	closeWg.Add(2) // 1 for success, error channels
//
//	// Handle producer messages
//	go func() {
//		defer func() {
//			// channel closed, initiate producer shutdown
//			p.logger.Debug("closing kafka producer")
//			// wait for error and successes channels to close
//			p.producer.AsyncClose()
//			closeWg.Wait()
//			p.logger.Debug("kafka producer closed")
//			close(done)
//		}()
//		for message := range messages {
//			p.producer.Input() <- message
//		}
//	}()
//
//	// Handle errors returned by the producer
//	go func() {
//		defer closeWg.Done()
//		if p.errors != nil {
//			defer close(p.errors)
//		}
//		for err := range p.producer.Errors() {
//			var key []byte
//			if err.Msg.Key != nil {
//				if _key, err := err.Msg.Key.Encode(); err == nil {
//					key = _key
//				} else {
//					p.logger.Error("could not encode produced message key", zap.Error(err))
//				}
//			}
//			p.logger.Error(
//				"Error producing Kafka message",
//				zap.String("topic", err.Msg.Topic),
//				zap.String("key", string(key)),
//				zap.Int32("partition", err.Msg.Partition),
//				zap.Int64("offset", err.Msg.Offset),
//				zap.Error(err))
//			promLabels["partition"] = fmt.Sprintf("%d", err.Msg.Partition)
//			promLabels["topic"] = err.Msg.Topic
//			p.metrics.ErrorsProduced.With(promLabels).Add(1)
//			if p.errors != nil {
//				p.errors <- err
//			}
//		}
//	}()
//
//	// Handle successes returned by the producer
//	go func() {
//		defer closeWg.Done()
//		if p.successes != nil {
//			defer close(p.successes)
//		}
//		for msg := range p.producer.Successes() {
//			promLabels["partition"] = fmt.Sprintf("%d", msg.Partition)
//			promLabels["topic"] = msg.Topic
//			p.metrics.MessagesProduced.With(promLabels).Add(1)
//			if p.successes != nil {
//				p.successes <- msg
//			}
//		}
//	}()
//}
//
//// Successes returns the channel on which successfully published messages will be returned
//func (p Producer) Successes() chan *sarama.ProducerMessage {
//	return p.successes
//}
//
//// Errors returns the channel on which messages that could not be published will be returned
//func (p Producer) Errors() chan *sarama.ProducerError {
//	return p.errors
//}
