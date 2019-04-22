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

// KafkaMessageUnmarshaler defines an interface for unmarshaling messages received from Kafka to Go types
type KafkaMessageUnmarshaler interface {
	UnmarshalMessage(ctx context.Context, msg *sarama.ConsumerMessage, target interface{}) error
}

// KafkaMessageHandler defines an interface for handling new messages received by the Kafka consumer
type KafkaMessageHandler interface {
	HandleMessage(ctx context.Context, msg *sarama.ConsumerMessage, unmarshaler KafkaMessageUnmarshaler) error
}

// KafkaConfig contains connection settings and configuration for communicating with a Kafka cluster
type KafkaConfig struct {
	Broker                   string
	ClientID                 string
	TLSCaCrtPath             string
	TLSCrtPath               string
	TLSKeyPath               string
	Handlers                 map[string]KafkaMessageHandler
	JSONEnabled              bool
	Verbose                  bool
	KafkaVersion             string
	ProducerCompressionCodec string
	ProducerCompressionLevel int
	SchemaRegistry           *SchemaRegistryConfig
	kafkaMetrics
}

// KafkaClient wraps a sarama client and Kafka configuration and can be used to create producers and consumers
type KafkaClient struct {
	KafkaConfig
	client sarama.Client
}

// KafkaConsumer contains a sarama client, consumer, and implementation of the KafkaMessageUnmarshaler interface
type KafkaConsumer struct {
	KafkaClient
	consumer           sarama.Consumer
	messageUnmarshaler KafkaMessageUnmarshaler
}

// KafkaProducer contains a sarama client and async producer
type KafkaProducer struct {
	KafkaClient
	producer sarama.AsyncProducer
}

type kafkaMetrics struct {
	messagesProcessed     *prometheus.GaugeVec
	messageErrors         *prometheus.GaugeVec
	messageProcessingTime *prometheus.SummaryVec
	errorsProcessed       *prometheus.GaugeVec
	brokerMetrics         map[string]*prometheus.GaugeVec
	messagesProduced      *prometheus.GaugeVec
	errorsProduced        *prometheus.GaugeVec
}

// KafkaConsumerIface is an interface for consuming messages from a Kafka topic
type KafkaConsumerIface interface {
	ConsumeTopic(ctx context.Context, handler KafkaMessageHandler, topic string, offsets PartitionOffsets, readResult chan PartitionOffsets, catchupWg *sync.WaitGroup, exitAfterCaughtUp bool) error
	ConsumeTopicFromBeginning(ctx context.Context, handler KafkaMessageHandler, topic string, readResult chan PartitionOffsets, catchupWg *sync.WaitGroup, exitAfterCaughtUp bool) error
	ConsumeTopicFromLatest(ctx context.Context, handler KafkaMessageHandler, topic string, readResult chan PartitionOffsets) error
	Close()
}

// NewKafkaClient creates a Kafka client with metrics exporting and optional
// TLS that can be used to create consumers or producers
func (kc KafkaConfig) NewKafkaClient(ctx context.Context) (KafkaClient, error) {
	if kc.Verbose {
		saramaLogger, err := zap.NewStdLogAt(log.Get(ctx).Named("sarama"), zapcore.InfoLevel)
		if err != nil {
			panic(err)
		}
		sarama.Logger = saramaLogger
	}
	kafkaConfig := sarama.NewConfig()
	kafkaVersion, err := sarama.ParseKafkaVersion(kc.KafkaVersion)
	if err != nil {
		return KafkaClient{}, err
	}
	kafkaConfig.Version = kafkaVersion
	kafkaConfig.Consumer.Return.Errors = true
	kafkaConfig.ClientID = kc.ClientID
	kafkaConfig.Producer.RequiredAcks = sarama.WaitForAll
	kafkaConfig.Producer.Return.Successes = true
	kafkaConfig.Producer.Return.Errors = true
	var compressionCodec sarama.CompressionCodec
	switch kc.ProducerCompressionCodec {
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
		return KafkaClient{}, fmt.Errorf("unknown compression codec %v", kc.ProducerCompressionCodec)
	}
	kafkaConfig.Producer.Compression = compressionCodec
	kafkaConfig.Producer.CompressionLevel = kc.ProducerCompressionLevel

	kc.initKafkaMetrics(prometheus.DefaultRegisterer)

	// Export metrics from Sarama's metrics registry to Prometheus
	kafkaConfig.MetricRegistry = metrics.NewRegistry()
	go kc.recordBrokerMetrics(ctx, 500*time.Millisecond, kafkaConfig.MetricRegistry)

	if kc.TLSCrtPath != "" && kc.TLSKeyPath != "" {
		cer, err := tls.LoadX509KeyPair(kc.TLSCrtPath, kc.TLSKeyPath)
		if err != nil {
			log.Get(ctx).Panic("Failed to load Kafka Server TLS Certificates", zap.Error(err))
		}
		kafkaConfig.Net.TLS.Config = &tls.Config{
			Certificates:       []tls.Certificate{cer},
			InsecureSkipVerify: true,
		}
		kafkaConfig.Net.TLS.Config.BuildNameToCertificate()
		kafkaConfig.Net.TLS.Enable = true

		if kc.TLSCaCrtPath != "" {
			caCert, err := ioutil.ReadFile(kc.TLSCaCrtPath)
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

	saramaClient, err := sarama.NewClient([]string{kc.Broker}, kafkaConfig)
	if err != nil {
		return KafkaClient{}, err
	}

	return KafkaClient{
		KafkaConfig: kc,
		client:      saramaClient,
	}, nil
}

// NewKafkaConsumer sets up a Kafka consumer
func (kc KafkaClient) NewKafkaConsumer() (KafkaConsumer, error) {
	consumer, err := sarama.NewConsumerFromClient(kc.client)
	if err != nil {
		if closeErr := kc.client.Close(); closeErr != nil {
			log.Get(context.Background()).Error("Error closing Kafka client", zap.Error(err))
		}
		return KafkaConsumer{}, err
	}

	kafkaConsumer := KafkaConsumer{
		KafkaClient: kc,
		consumer:    consumer,
	}
	messageUnmarshaler := &kafkaMessageDecoder{}
	if kc.JSONEnabled {
		kafkaConsumer.messageUnmarshaler = &jsonMessageUnmarshaler{messageUnmarshaler: messageUnmarshaler}
	} else {
		kc.KafkaConfig.SchemaRegistry.client = &schemaRegistryClient{}
		kc.KafkaConfig.SchemaRegistry.messageUnmarshaler = messageUnmarshaler
		kafkaConsumer.messageUnmarshaler = kc.KafkaConfig.SchemaRegistry
	}
	return kafkaConsumer, nil
}

// NewKafkaProducer creates a sarama producer from a client
func (kc KafkaClient) NewKafkaProducer() (KafkaProducer, error) {
	producer, err := sarama.NewAsyncProducerFromClient(kc.client)
	if err != nil {
		if closeErr := producer.Close(); closeErr != nil {
			log.Get(context.Background()).Error("Error closing Kafka producer", zap.Error(err))
		}
		return KafkaProducer{}, err
	}

	return KafkaProducer{
		KafkaClient: kc,
		producer:    producer,
	}, nil
}

// Close the underlying Kafka client
func (kc KafkaClient) Close() {
	if err := kc.client.Close(); err != nil {
		log.Get(context.Background()).Error("Error closing Kafka client", zap.Error(err))
	}
}

func (kc KafkaConfig) updateBrokerMetrics(registry metrics.Registry) {
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
		gauge, ok := kc.brokerMetrics[promMetricName]
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
			kc.brokerMetrics[promMetricName] = gauge
		}
		gauge.With(prometheus.Labels{"broker": kc.Broker, "client": kc.ClientID}).Set(metricVal)
	})
}

func (kc KafkaConfig) recordBrokerMetrics(
	ctx context.Context,
	updateInterval time.Duration,
	registry metrics.Registry,
) {
	ticker := time.NewTicker(updateInterval)
	for {
		select {
		case <-ticker.C:
			kc.updateBrokerMetrics(registry)
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}

func (kc *KafkaConfig) initKafkaMetrics(registry prometheus.Registerer) {
	kc.brokerMetrics = make(map[string]*prometheus.GaugeVec)
	promLabels := []string{"topic", "partition", "client"}
	kc.messageProcessingTime = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "kafka_message_processing_time_seconds",
			Help: "Kafka Message processing duration in seconds",
		},
		promLabels,
	)
	kc.messagesProcessed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_messages_processed",
			Help: "Number of Kafka messages processed",
		},
		promLabels,
	)
	kc.messageErrors = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_message_errors",
			Help: "Number of Kafka messages that couldn't be processed due to an error",
		},
		promLabels,
	)
	kc.errorsProcessed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_errors_processed",
			Help: "Number of errors received from the Kafka broker",
		},
		promLabels,
	)
	kc.messagesProduced = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_messages_produced",
			Help: "Number of Kafka messages produced",
		},
		promLabels,
	)
	kc.errorsProduced = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_errors_produced",
			Help: "Number of Kafka errors produced",
		},
		promLabels,
	)
	registry.MustRegister(
		kc.messageProcessingTime,
		kc.messagesProcessed,
		kc.messageErrors,
		kc.errorsProcessed,
		kc.messagesProduced,
		kc.errorsProduced,
	)
}

// Close Sarama consumer and client
func (kc KafkaConsumer) Close() {
	err := kc.consumer.Close()
	if err != nil {
		log.Get(context.Background()).Error("Error closing Kafka consumer", zap.Error(err))
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
func (kc KafkaConsumer) ConsumeTopic(
	ctx context.Context,
	handler KafkaMessageHandler,
	topic string,
	offsets PartitionOffsets,
	readResult chan PartitionOffsets,
	catchupWg *sync.WaitGroup,
	exitAfterCaughtUp bool,
) error {
	log.Get(ctx).Info("Starting Kafka consumer", zap.String("topic", topic))
	var partitionsCatchupWg sync.WaitGroup
	partitions, err := kc.consumer.Partitions(topic)
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
		newestOffset, err := kc.client.GetOffset(topic, partition, sarama.OffsetNewest)
		if err != nil {
			return err
		}
		// client.GetOffset returns the offset of the next message to be processed
		// so subtract 1 here because if there are no new messages after boot up,
		// we could be waiting indefinitely
		newestOffset--
		go kc.consumePartition(
			ctx, handler, topic, partition, startOffset, newestOffset,
			readToChan, &partitionsCatchupWg, exitAfterCaughtUp)
	}

	go func() {
		partitionsCatchupWg.Wait()
		if catchupWg != nil {
			catchupWg.Done()
			log.Get(ctx).Info("All partitions caught up", zap.String("topic", topic))
		}

		readToOffsets := make(PartitionOffsets)
		defer func() {
			log.Get(ctx).Info("All partition consumers closed", zap.String("topic", topic))
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
func (kc KafkaConsumer) ConsumeTopicFromBeginning(
	ctx context.Context,
	handler KafkaMessageHandler,
	topic string,
	readResult chan PartitionOffsets,
	catchupWg *sync.WaitGroup,
	exitAfterCaughtUp bool,
) error {
	partitions, err := kc.consumer.Partitions(topic)
	if err != nil {
		return err
	}
	startOffsets := make(PartitionOffsets, len(partitions))
	for _, partition := range partitions {
		startOffsets[partition] = sarama.OffsetOldest
	}
	return kc.ConsumeTopic(ctx, handler, topic, startOffsets, readResult, catchupWg, exitAfterCaughtUp)
}

// ConsumeTopicFromLatest starts Kafka consumers on all partitions
// in a given topic from the message with the latest offset.
func (kc KafkaConsumer) ConsumeTopicFromLatest(
	ctx context.Context,
	handler KafkaMessageHandler,
	topic string,
	readResult chan PartitionOffsets,
) error {
	partitions, err := kc.consumer.Partitions(topic)
	if err != nil {
		return err
	}
	startOffsets := make(PartitionOffsets, len(partitions))
	for _, partition := range partitions {
		startOffsets[partition] = sarama.OffsetNewest
	}
	return kc.ConsumeTopic(ctx, handler, topic, startOffsets, readResult, nil, false)
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
func (kc KafkaConsumer) consumePartition(
	ctx context.Context,
	handler KafkaMessageHandler,
	topic string,
	partition int32,
	startOffset int64,
	caughtUpOffset int64,
	readResult chan consumerLastStatus,
	catchupWg *sync.WaitGroup,
	exitAfterCaughtUp bool,
) {
	partitionConsumer, err := kc.consumer.ConsumePartition(topic, partition, startOffset)
	if err != nil {
		log.Get(ctx).Panic(
			"Failed to create Kafka partition consumer",
			zap.String("topic", topic), zap.Int32("partition", partition),
			zap.Int64("start_offset", startOffset), zap.Error(err))
	}

	curOffset := startOffset

	defer func() {
		err := partitionConsumer.Close()
		if err != nil {
			log.Get(ctx).Error(
				"Error closing Kafka partition consumer",
				zap.Error(err), zap.String("topic", topic), zap.Int32("partition", partition))
		} else {
			log.Get(ctx).Debug(
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
		"client":    kc.KafkaClient.ClientID,
	}
	for {
		select {
		case msg, ok := <-partitionConsumer.Messages():
			curOffset = msg.Offset
			if !ok {
				kc.KafkaConfig.messageErrors.With(promLabels).Add(1)
				log.Get(ctx).Error(
					"Unable to process message from Kafka",
					zap.ByteString("key", msg.Key), zap.Int64("offset", msg.Offset),
					zap.Int32("partition", msg.Partition), zap.String("topic", msg.Topic),
					zap.Time("message_ts", msg.Timestamp))
				continue
			}
			timer := prometheus.NewTimer(kc.KafkaConfig.messageProcessingTime.With(promLabels))
			if err := handler.HandleMessage(ctx, msg, kc.messageUnmarshaler); err != nil {
				log.Get(ctx).Error(
					"Error handling message",
					zap.String("topic", topic),
					zap.Int32("partition", partition),
					zap.Int64("offset", msg.Offset),
					zap.ByteString("key", msg.Key),
					zap.String("message", string(msg.Value)),
					zap.Error(err))
			}
			timer.ObserveDuration()
			kc.KafkaConfig.messagesProcessed.With(promLabels).Add(1)
			if msg.Offset == caughtUpOffset {
				caughtUp = true
				catchupWg.Done()
				log.Get(ctx).Debug(
					"Successfully read to target Kafka offset",
					zap.String("topic", topic), zap.Int32("partition", partition),
					zap.Int64("offset", msg.Offset))
				if exitAfterCaughtUp {
					return
				}
			}
		case err := <-partitionConsumer.Errors():
			kc.KafkaConfig.errorsProcessed.With(promLabels).Add(1)
			log.Get(ctx).Error("Encountered an error from Kafka", zap.Error(err))
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
func (kp KafkaProducer) RunProducer(messages <-chan *sarama.ProducerMessage, done chan bool) {
	promLabels := prometheus.Labels{
		"client": kp.KafkaConfig.ClientID,
	}
	var closeWg sync.WaitGroup
	closeWg.Add(2) // 1 for success, error channels

	// Handle producer messages
	go func() {
		defer func() {
			// channel closed, initiate producer shutdown
			log.Get(context.Background()).Debug("closing kafka producer")
			// wait for error and successes channels to close
			kp.producer.AsyncClose()
			closeWg.Wait()
			log.Get(context.Background()).Debug("kafka producer closed")
			done <- true
		}()
		for message := range messages {
			kp.producer.Input() <- message
		}
	}()

	// Handle errors returned by the producer
	go func() {
		defer closeWg.Done()
		for err := range kp.producer.Errors() {
			var key []byte
			if err.Msg.Key != nil {
				if _key, err := err.Msg.Key.Encode(); err == nil {
					key = _key
				} else {
					log.Get(context.Background()).Error("could not encode produced message key", zap.Error(err))
				}
			}
			log.Get(context.Background()).Error(
				"Error producing Kafka message",
				zap.String("topic", err.Msg.Topic),
				zap.String("key", string(key)),
				zap.Int32("partition", err.Msg.Partition),
				zap.Int64("offset", err.Msg.Offset),
				zap.Error(err))
			promLabels["partition"] = fmt.Sprintf("%d", err.Msg.Partition)
			promLabels["topic"] = err.Msg.Topic
			kp.KafkaConfig.errorsProduced.With(promLabels).Add(1)
		}
	}()

	// Handle successes returned by the producer
	go func() {
		defer closeWg.Done()
		for msg := range kp.producer.Successes() {
			promLabels["partition"] = fmt.Sprintf("%d", msg.Partition)
			promLabels["topic"] = msg.Topic
			kp.KafkaConfig.messagesProduced.With(promLabels).Add(1)
		}
	}()
}
