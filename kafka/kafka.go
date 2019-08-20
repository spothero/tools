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
	"sync"
	"time"

	"github.com/Shopify/sarama"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rcrowley/go-metrics"
	"github.com/spothero/tools/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// MessageUnmarshaler defines an interface for unmarshaling messages received from Kafka to Go types
type MessageUnmarshaler interface {
	UnmarshalMessage(ctx context.Context, msg *sarama.ConsumerMessage, target interface{}) error
}

// MessageHandler defines an interface for handling new messages received by the Kafka consumer
type MessageHandler interface {
	HandleMessage(ctx context.Context, msg *sarama.ConsumerMessage, unmarshaler MessageUnmarshaler) error
}

// Config contains connection settings and configuration for communicating with a Kafka cluster
type Config struct {
	Broker                   string
	ClientID                 string
	TLSCaCrtPath             string
	TLSCrtPath               string
	TLSKeyPath               string
	Handlers                 map[string]MessageHandler
	JSONEnabled              bool
	Verbose                  bool
	KafkaVersion             string
	ProducerCompressionCodec string
	ProducerCompressionLevel int
	SchemaRegistry           *SchemaRegistryConfig
	messagesProcessed        *prometheus.GaugeVec
	messageErrors            *prometheus.GaugeVec
	messageProcessingTime    *prometheus.SummaryVec
	errorsProcessed          *prometheus.GaugeVec
	brokerMetrics            map[string]*prometheus.GaugeVec
	messagesProduced         *prometheus.GaugeVec
	errorsProduced           *prometheus.GaugeVec
}

// Client wraps a sarama client and Kafka configuration and can be used to create producers and consumers
type Client struct {
	Config
	client sarama.Client
}

// ClientIface is an interface for creating consumers and producers
type ClientIface interface {
	NewConsumer(logger *zap.Logger) (ConsumerIface, error)
	NewProducer(logger *zap.Logger, returnMessages bool) (ProducerIface, error)
}

// Consumer contains a sarama client, consumer, and implementation of the MessageUnmarshaler interface
type Consumer struct {
	Client
	consumer           sarama.Consumer
	messageUnmarshaler MessageUnmarshaler
	logger             *zap.Logger
}

// Producer contains a sarama client and async producer
type Producer struct {
	Client
	producer  sarama.AsyncProducer
	successes chan *sarama.ProducerMessage
	errors    chan *sarama.ProducerError
	logger    *zap.Logger
}

// ConsumerIface is an interface for consuming messages from a Kafka topic
type ConsumerIface interface {
	ConsumeTopic(ctx context.Context, handler MessageHandler, topic string, offsets PartitionOffsets, readResult chan PartitionOffsets, catchupWg *sync.WaitGroup, exitAfterCaughtUp bool) error
	ConsumeTopicFromBeginning(ctx context.Context, handler MessageHandler, topic string, readResult chan PartitionOffsets, catchupWg *sync.WaitGroup, exitAfterCaughtUp bool) error
	ConsumeTopicFromLatest(ctx context.Context, handler MessageHandler, topic string, readResult chan PartitionOffsets) error
	Close()
}

// ProducerIface is an interface for producing Kafka messages
type ProducerIface interface {
	RunProducer(messages chan *sarama.ProducerMessage, done chan bool)
	Successes() chan *sarama.ProducerMessage
	Errors() chan *sarama.ProducerError
}

// NewClient creates a Kafka client with metrics exporting and optional
// TLS that can be used to create consumers or producers
func (c Config) NewClient(ctx context.Context) (Client, error) {
	if c.Verbose {
		saramaLogger, err := zap.NewStdLogAt(log.Get(ctx).Named("sarama"), zapcore.InfoLevel)
		if err != nil {
			panic(err)
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

	c.initKafkaMetrics(prometheus.DefaultRegisterer)

	// Export metrics from Sarama's metrics registry to Prometheus
	kafkaConfig.MetricRegistry = metrics.NewRegistry()
	go c.recordBrokerMetrics(ctx, 500*time.Millisecond, kafkaConfig.MetricRegistry)

	if c.TLSCrtPath != "" && c.TLSKeyPath != "" {
		cer, err := tls.LoadX509KeyPair(c.TLSCrtPath, c.TLSKeyPath)
		if err != nil {
			log.Get(ctx).Panic("Failed to load Kafka Server TLS Certificates", zap.Error(err))
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
				log.Get(ctx).Panic("Failed to load Kafka Server CA Certificate", zap.Error(err))
			}
			if len(caCert) > 0 {
				caCertPool := x509.NewCertPool()
				caCertPool.AppendCertsFromPEM([]byte(caCert))
				kafkaConfig.Net.TLS.Config.RootCAs = caCertPool
				kafkaConfig.Net.TLS.Config.InsecureSkipVerify = false
			}
		}
	}

	saramaClient, err := sarama.NewClient([]string{c.Broker}, kafkaConfig)
	if err != nil {
		return Client{}, err
	}

	return Client{
		Config: c,
		client: saramaClient,
	}, nil
}

// NewConsumer sets up a Kafka consumer
func (c Client) NewConsumer(logger *zap.Logger) (ConsumerIface, error) {
	consumer, err := sarama.NewConsumerFromClient(c.client)
	if err != nil {
		if closeErr := c.client.Close(); closeErr != nil {
			log.Get(context.Background()).Error("Error closing Kafka client", zap.Error(err))
		}
		return Consumer{}, err
	}
	kafkaConsumer := Consumer{
		Client:   c,
		consumer: consumer,
	}
	messageUnmarshaler := &messageDecoder{}
	if c.JSONEnabled {
		kafkaConsumer.messageUnmarshaler = &jsonMessageUnmarshaler{messageUnmarshaler: messageUnmarshaler}
	} else {
		c.Config.SchemaRegistry.client = &schemaRegistryClient{}
		c.Config.SchemaRegistry.messageUnmarshaler = messageUnmarshaler
		kafkaConsumer.messageUnmarshaler = c.Config.SchemaRegistry
	}
	if logger != nil {
		kafkaConsumer.logger = logger
	} else {
		kafkaConsumer.logger = zap.NewNop()
	}
	return kafkaConsumer, nil
}

// NewProducer creates a sarama producer from a client. If the returnMessages flag is true,
// messages from the producer will be produced on the Success or Errors channel depending
// on the outcome of the produced message.
func (c Client) NewProducer(logger *zap.Logger, returnMessages bool) (ProducerIface, error) {
	producer, err := sarama.NewAsyncProducerFromClient(c.client)
	if err != nil {
		if closeErr := producer.Close(); closeErr != nil {
			log.Get(context.Background()).Error("Error closing Kafka producer", zap.Error(err))
		}
		return Producer{}, err
	}

	p := Producer{
		Client:   c,
		producer: producer,
	}
	if returnMessages {
		p.successes = make(chan *sarama.ProducerMessage)
		p.errors = make(chan *sarama.ProducerError)
	}
	if logger != nil {
		p.logger = logger
	} else {
		p.logger = zap.NewNop()
	}
	return p, nil
}

// Close the underlying Kafka client
func (c Client) Close() {
	if err := c.client.Close(); err != nil {
		log.Get(context.Background()).Error("Error closing Kafka client", zap.Error(err))
	}
}

func (c Config) updateBrokerMetrics(registry metrics.Registry) {
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
		gauge, ok := c.brokerMetrics[promMetricName]
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
			c.brokerMetrics[promMetricName] = gauge
		}
		gauge.With(prometheus.Labels{"broker": c.Broker, "client": c.ClientID}).Set(metricVal)
	})
}

func (c Config) recordBrokerMetrics(
	ctx context.Context,
	updateInterval time.Duration,
	registry metrics.Registry,
) {
	ticker := time.NewTicker(updateInterval)
	for {
		select {
		case <-ticker.C:
			c.updateBrokerMetrics(registry)
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}

func (c *Config) initKafkaMetrics(registry prometheus.Registerer) {
	c.brokerMetrics = make(map[string]*prometheus.GaugeVec)
	promLabels := []string{"topic", "partition", "client"}
	c.messageProcessingTime = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "kafka_message_processing_time_seconds",
			Help: "Kafka Message processing duration in seconds",
		},
		promLabels,
	)
	c.messagesProcessed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_messages_processed",
			Help: "Number of Kafka messages processed",
		},
		promLabels,
	)
	c.messageErrors = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_message_errors",
			Help: "Number of Kafka messages that couldn't be processed due to an error",
		},
		promLabels,
	)
	c.errorsProcessed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_errors_processed",
			Help: "Number of errors received from the Kafka broker",
		},
		promLabels,
	)
	c.messagesProduced = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_messages_produced",
			Help: "Number of Kafka messages produced",
		},
		promLabels,
	)
	c.errorsProduced = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_errors_produced",
			Help: "Number of Kafka errors produced",
		},
		promLabels,
	)
	registry.MustRegister(
		c.messageProcessingTime,
		c.messagesProcessed,
		c.messageErrors,
		c.errorsProcessed,
		c.messagesProduced,
		c.errorsProduced,
	)
}

// Close Sarama consumer and client
func (c Consumer) Close() {
	err := c.consumer.Close()
	if err != nil {
		c.logger.Error("Error closing Kafka consumer", zap.Error(err))
	}
}

// PartitionOffsets is a mapping of partition ID to an offset to which a consumer read on that partition
type PartitionOffsets map[int32]int64

// ConsumeTopic consumes a particular Kafka topic from startOffset to endOffset or
// from startOffset to forever
//
// This function will create consumers for all partitions in a topic and read
// from the given offset on each partition to the latest offset when the consumer was started, then notify the caller
// via catchupWg. If exitAfterCaughtUp is true, the consumer will exit after it reads message at the latest offset
// when it started up. When all partition consumers are closed, it will send the last offset read on each partition
// through the readResult channel. If exitAfterCaughtUp is true, the consumer will exit
// after reading to the latest offset.
func (c Consumer) ConsumeTopic(
	ctx context.Context,
	handler MessageHandler,
	topic string,
	offsets PartitionOffsets,
	readResult chan PartitionOffsets,
	catchupWg *sync.WaitGroup,
	exitAfterCaughtUp bool,
) error {
	c.logger.Info("Starting Kafka consumer", zap.String("topic", topic))
	var partitionsCatchupWg sync.WaitGroup
	partitions, err := c.consumer.Partitions(topic)
	if err != nil {
		return err
	}
	readToChan := make(chan consumerLastStatus)

	for _, partition := range partitions {
		startOffset, ok := offsets[partition]
		if !ok {
			return fmt.Errorf("start offset not found for partition %d, topic %s", partition, topic)
		}
		partitionsCatchupWg.Add(1)
		newestOffset, err := c.client.GetOffset(topic, partition, sarama.OffsetNewest)
		if err != nil {
			return err
		}
		// client.GetOffset returns the offset of the next message to be processed
		// so subtract 1 here because if there are no new messages after boot up,
		// we could be waiting indefinitely
		newestOffset--
		go c.consumePartition(
			ctx, handler, topic, partition, startOffset, newestOffset,
			readToChan, &partitionsCatchupWg, exitAfterCaughtUp)
	}

	go func() {
		partitionsCatchupWg.Wait()
		if catchupWg != nil {
			catchupWg.Done()
			c.logger.Info("All partitions caught up", zap.String("topic", topic))
		}

		readToOffsets := make(PartitionOffsets)
		defer func() {
			c.logger.Info("All partition consumers closed", zap.String("topic", topic))
			if readResult != nil {
				readResult <- readToOffsets
			}
		}()

		for messagesAwaiting := len(partitions); messagesAwaiting > 0; {
			read := <-readToChan
			readToOffsets[read.partition] = read.offset
			messagesAwaiting--
		}
	}()

	return nil
}

// ConsumeTopicFromBeginning starts Kafka consumers on all partitions
// in a given topic from the message with the oldest offset.
func (c Consumer) ConsumeTopicFromBeginning(
	ctx context.Context,
	handler MessageHandler,
	topic string,
	readResult chan PartitionOffsets,
	catchupWg *sync.WaitGroup,
	exitAfterCaughtUp bool,
) error {
	partitions, err := c.consumer.Partitions(topic)
	if err != nil {
		return err
	}
	startOffsets := make(PartitionOffsets, len(partitions))
	for _, partition := range partitions {
		startOffsets[partition] = sarama.OffsetOldest
	}
	return c.ConsumeTopic(ctx, handler, topic, startOffsets, readResult, catchupWg, exitAfterCaughtUp)
}

// ConsumeTopicFromLatest starts Kafka consumers on all partitions
// in a given topic from the message with the latest offset.
func (c Consumer) ConsumeTopicFromLatest(
	ctx context.Context,
	handler MessageHandler,
	topic string,
	readResult chan PartitionOffsets,
) error {
	partitions, err := c.consumer.Partitions(topic)
	if err != nil {
		return err
	}
	startOffsets := make(PartitionOffsets, len(partitions))
	for _, partition := range partitions {
		startOffsets[partition] = sarama.OffsetNewest
	}
	return c.ConsumeTopic(ctx, handler, topic, startOffsets, readResult, nil, false)
}

type consumerLastStatus struct {
	offset    int64
	partition int32
}

// Consume a particular topic and partition
//
// When a new message from Kafka is received, handleMessage on the handler
// will be called to process the message. This function will create consumers
// for all partitions in a topic and read from startOffset to caughtUpOffset
// then notify the caller via catchupWg. While reading from startOffset to
// caughtUpOffset, messages will be handled synchronously to ensure that
// all messages are processed before notifying the caller that the consumer
// is caught up. When the consumer shuts down, it returns the last offset to
// which it read through the readResult channel.
func (c Consumer) consumePartition(
	ctx context.Context,
	handler MessageHandler,
	topic string,
	partition int32,
	startOffset int64,
	caughtUpOffset int64,
	readResult chan consumerLastStatus,
	catchupWg *sync.WaitGroup,
	exitAfterCaughtUp bool,
) {
	partitionConsumer, err := c.consumer.ConsumePartition(topic, partition, startOffset)
	if err != nil {
		c.logger.Panic(
			"Failed to create Kafka partition consumer",
			zap.String("topic", topic), zap.Int32("partition", partition),
			zap.Int64("start_offset", startOffset), zap.Error(err))
	}

	curOffset := startOffset

	defer func() {
		err := partitionConsumer.Close()
		if err != nil {
			c.logger.Error(
				"Error closing Kafka partition consumer",
				zap.Error(err), zap.String("topic", topic), zap.Int32("partition", partition))
		} else {
			c.logger.Debug(
				"Kafka partition consumer closed", zap.String("topic", topic),
				zap.Int32("partition", partition))
		}
		readResult <- consumerLastStatus{offset: curOffset, partition: partition}
	}()

	caughtUp := false
	if caughtUpOffset == -1 {
		log.Get(ctx).Debug(
			"No messages on partition for topic, consumer is caught up", zap.String("topic", topic),
			zap.Int32("partition", partition))
		catchupWg.Done()
		caughtUp = true
		if exitAfterCaughtUp {
			return
		}
	}

	promLabels := prometheus.Labels{
		"topic":     topic,
		"partition": fmt.Sprintf("%d", partition),
		"client":    c.Client.ClientID,
	}
	for {
		select {
		case msg, ok := <-partitionConsumer.Messages():
			curOffset = msg.Offset
			if !ok {
				c.Config.messageErrors.With(promLabels).Add(1)
				c.logger.Error(
					"Unable to process message from Kafka",
					zap.ByteString("key", msg.Key), zap.Int64("offset", msg.Offset),
					zap.Int32("partition", msg.Partition), zap.String("topic", msg.Topic),
					zap.Time("message_ts", msg.Timestamp))
				continue
			}
			timer := prometheus.NewTimer(c.Config.messageProcessingTime.With(promLabels))
			if err := handler.HandleMessage(ctx, msg, c.messageUnmarshaler); err != nil {
				c.logger.Error(
					"Error handling message",
					zap.String("topic", topic),
					zap.Int32("partition", partition),
					zap.Int64("offset", msg.Offset),
					zap.ByteString("key", msg.Key),
					zap.String("message", string(msg.Value)),
					zap.Error(err))
			}
			timer.ObserveDuration()
			c.Config.messagesProcessed.With(promLabels).Add(1)
			if msg.Offset == caughtUpOffset {
				caughtUp = true
				catchupWg.Done()
				c.logger.Debug(
					"Successfully read to target Kafka offset",
					zap.String("topic", topic), zap.Int32("partition", partition),
					zap.Int64("offset", msg.Offset))
				if exitAfterCaughtUp {
					return
				}
			}
		case err := <-partitionConsumer.Errors():
			c.Config.errorsProcessed.With(promLabels).Add(1)
			c.logger.Error("Encountered an error from Kafka", zap.Error(err))
		case <-ctx.Done():
			if !caughtUp {
				// signal to the catchup wg that we're done if there's been a cancellation request
				// so that the caller can exit if canceled before being caught up
				catchupWg.Done()
			}
			return
		}
	}
}

// RunProducer wraps the sarama AsyncProducer and adds metrics, logging, and a shutdown procedure
// to the producer. To stop the producer, close the messages channel; when the producer is shutdown a signal will
// be emitted on the done channel. If the messages channel is unbuffered, each message sent to the producer is
// guaranteed to at least have been attempted to be produced to Kafka.
func (p Producer) RunProducer(messages chan *sarama.ProducerMessage, done chan bool) {
	promLabels := prometheus.Labels{
		"client": p.Config.ClientID,
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
			p.Config.errorsProduced.With(promLabels).Add(1)
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
			p.Config.messagesProduced.With(promLabels).Add(1)
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
