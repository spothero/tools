package core

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
	"go.uber.org/zap"
)

// KafkaMessageUnmarshaler defines an interface for unmarshaling messages received from Kafka to Go types
type KafkaMessageUnmarshaler interface {
	UnmarshalMessage(ctx context.Context, msg *sarama.ConsumerMessage, target interface{}) error
}

// KafkaMessageHandler defines an interface for handling new messages received by the Kafka consumer
type KafkaMessageHandler interface {
	HandleMessage(ctx context.Context, msg *sarama.ConsumerMessage, unmarshaler KafkaMessageUnmarshaler, caughtUpOffset int64) error
}

// KafkaConfig contains connection settings and configuration for communicating with a Kafka cluster
type KafkaConfig struct {
	Broker       string
	ClientID     string
	TLSCaCrtPath string
	TLSCrtPath   string
	TLSKeyPath   string
	Handlers     map[string]KafkaMessageHandler
	JSONEnabled  bool
	Verbose      bool
	kafkaMetrics
}

type kafkaClient struct {
	client      sarama.Client
	kafkaConfig *KafkaConfig
}

// KafkaConsumer contains a sarama client, consumer, and implementation of the KafkaMessageUnmarshaler interface
type KafkaConsumer struct {
	kafkaClient
	consumer           sarama.Consumer
	messageUnmarshaler KafkaMessageUnmarshaler
}

// KafkaProducer contains a sarama client and async producer
type KafkaProducer struct {
	kafkaClient
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

// InitKafkaClient creates a Kafka client with metrics exporting and optional
// TLS that can be used to create consumers or producers
func (kc *KafkaConfig) InitKafkaClient(ctx context.Context) (sarama.Client, error) {
	if kc.Verbose {
		saramaLogger, err := CreateStdLogger(Logger.Named("sarama"), "info")
		if err != nil {
			panic(err)
		}
		sarama.Logger = saramaLogger
	}
	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Consumer.Return.Errors = true
	kafkaConfig.Version = sarama.V1_0_0_0
	kafkaConfig.ClientID = kc.ClientID

	kc.initKafkaMetrics(prometheus.DefaultRegisterer)

	// Export metrics from Sarama's metrics registry to Prometheus
	kafkaConfig.MetricRegistry = metrics.NewRegistry()
	go kc.recordBrokerMetrics(ctx, 500*time.Millisecond, kafkaConfig.MetricRegistry)

	if kc.TLSCrtPath != "" && kc.TLSKeyPath != "" {
		cer, err := tls.LoadX509KeyPair(kc.TLSCrtPath, kc.TLSKeyPath)
		if err != nil {
			Logger.Panic("Failed to load Kafka Server TLS Certificates", zap.Error(err))
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
				Logger.Panic("Failed to load Kafka Server CA Certificate", zap.Error(err))
			}
			if len(caCert) > 0 {
				caCertPool := x509.NewCertPool()
				caCertPool.AppendCertsFromPEM([]byte(caCert))
				kafkaConfig.Net.TLS.Config.RootCAs = caCertPool
				kafkaConfig.Net.TLS.Config.InsecureSkipVerify = false
			}
		}
	}

	return sarama.NewClient([]string{kc.Broker}, kafkaConfig)
}

// InitKafkaConsumer sets up Kafka client and consumer
func (kc *KafkaConfig) InitKafkaConsumer(
	client sarama.Client,
	schemaRegistryConfig *SchemaRegistryConfig,
	initialOffset int64,
) (*KafkaConsumer, error) {
	consumer, err := sarama.NewConsumerFromClient(client)
	if err != nil {
		if closeErr := client.Close(); closeErr != nil {
			Logger.Error("Error closing Kafka client", zap.Error(err))
		}
		return nil, err
	}
	client.Config().Consumer.Offsets.Initial = initialOffset

	kafkaConsumer := &KafkaConsumer{
		kafkaClient: kafkaClient{
			client:      client,
			kafkaConfig: kc,
		},
		consumer: consumer,
	}
	messageUnmarshaler := &kafkaMessageDecoder{}
	if kc.JSONEnabled {
		kafkaConsumer.messageUnmarshaler = &jsonMessageUnmarshaler{messageUnmarshaler: messageUnmarshaler}
	} else {
		schemaRegistryConfig.client = &schemaRegistryClient{}
		schemaRegistryConfig.messageUnmarshaler = messageUnmarshaler
		kafkaConsumer.messageUnmarshaler = schemaRegistryConfig
	}
	return kafkaConsumer, nil
}

// InitKafkaProducer creates a sarama producer from a client
func (kc *KafkaConfig) InitKafkaProducer(client sarama.Client) (*KafkaProducer, error) {
	producer, err := sarama.NewAsyncProducerFromClient(client)
	if err != nil {
		if closeErr := producer.Close(); closeErr != nil {
			Logger.Error("Error closing Kafka producer", zap.Error(err))
		}
		return nil, err
	}

	kafkaProducer := &KafkaProducer{
		kafkaClient: kafkaClient{
			client:      client,
			kafkaConfig: kc,
		},
		producer: producer,
	}
	return kafkaProducer, nil
}

func (kc *KafkaConfig) updateBrokerMetrics(registry metrics.Registry) {
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
			Logger.Warn(
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

func (kc *KafkaConfig) recordBrokerMetrics(
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
	registry.MustRegister(kc.messageProcessingTime, kc.messagesProcessed, kc.messageErrors, kc.errorsProcessed)
}

// Close Sarama consumer and client
func (kc *KafkaConsumer) Close() {
	kc.consumer.Close()
	if !kc.client.Closed() {
		kc.client.Close()
	}
}

// ConsumeTopic consumes a particular Kafka topic from startOffset to endOffset or
// from startOffset to forever
//
// This function will create consumers for all partitions in a topic and read
// from startOffset to caughtUpOffset then notify the caller via catchupWg. When
// all partition consumers are closed, it will notify the caller on wg.
func (kc *KafkaConsumer) ConsumeTopic(
	ctx context.Context,
	handlers map[string]KafkaMessageHandler,
	topic string,
	startOffset int64,
	wg *sync.WaitGroup,
	catchupWg *sync.WaitGroup,
) {
	Logger.Info("Starting Kafka consumer", zap.String("topic", topic))

	var partitionsWg sync.WaitGroup
	var partitionsCatchupWg sync.WaitGroup

	// Create a partition consumer for each partition in the topic
	handler, ok := handlers[topic]
	if !ok {
		Logger.Panic("No handler defined for topic, messages cannot be processed!", zap.String("topic", topic))
	}
	partitions, err := kc.consumer.Partitions(topic)
	if err != nil {
		Logger.Panic("Couldn't get Kafka partitions", zap.String("topic", topic), zap.Error(err))
	}
	for _, partition := range partitions {
		partitionsWg.Add(1)
		partitionsCatchupWg.Add(1)
		newestOffset, err := kc.client.GetOffset(topic, partition, sarama.OffsetNewest)
		if err != nil {
			Logger.Panic(
				"Failed to get the newest offset from Kafka",
				zap.String("topic", topic), zap.Int32("partition", partition),
				zap.Error(err))
		}
		// client.GetOffset returns the offset of the next message to be processed
		// so subtract 1 here because if there are no new messages after boot up,
		// we could be waiting indefinitely
		newestOffset--
		go kc.consumePartition(
			ctx, handler, topic, partition, startOffset, newestOffset,
			&partitionsWg, &partitionsCatchupWg)
	}
	partitionsCatchupWg.Wait()
	if catchupWg != nil {
		catchupWg.Done()
	}
	Logger.Info("All partitions caught up", zap.String("topic", topic))
	partitionsWg.Wait()
	Logger.Info("All partition consumers closed", zap.String("topic", topic))
	wg.Done()
}

// Consume a particular topic and partition
//
// When a new message from Kafka is received, handleMessage on the handler
// will be called to process the message. This function will create consumers
// for all partitions in a topic and read from startOffset to caughtUpOffset
// then notify the caller via catchupWg. While reading from startOffset to
// caughtUpOffset, messages will be handled synchronously to ensure that
// all messages are processed before notifying the caller that the consumer
// is caught up.
func (kc *KafkaConsumer) consumePartition(
	ctx context.Context,
	handler KafkaMessageHandler,
	topic string,
	partition int32,
	startOffset int64,
	caughtUpOffset int64,
	wg *sync.WaitGroup,
	catchupWg *sync.WaitGroup,
) {
	partitionConsumer, err := kc.consumer.ConsumePartition(topic, partition, startOffset)
	if err != nil {
		Logger.Panic(
			"Failed to create Kafka partition consumer",
			zap.String("topic", topic), zap.Int32("partition", partition),
			zap.Int64("start_offset", startOffset), zap.Error(err))
	}
	defer partitionConsumer.Close()

	if caughtUpOffset == -1 {
		Logger.Debug(
			"No messages on partition for topic, consumer is caught up", zap.String("topic", topic),
			zap.Int32("partition", partition))
		catchupWg.Done()
	}

	promLabels := prometheus.Labels{
		"topic":     topic,
		"partition": fmt.Sprintf("%d", partition),
		"client":    kc.kafkaConfig.ClientID,
	}
	for {
		select {
		case msg, ok := <-partitionConsumer.Messages():
			if !ok {
				kc.kafkaConfig.messageErrors.With(promLabels).Add(1)
				Logger.Error(
					"Unable to process message from Kafka",
					zap.ByteString("key", msg.Key), zap.Int64("offset", msg.Offset),
					zap.Int32("partition", msg.Partition), zap.String("topic", msg.Topic),
					zap.Time("message_ts", msg.Timestamp))
				continue
			}
			timer := prometheus.NewTimer(kc.kafkaConfig.messageProcessingTime.With(promLabels))
			if err := handler.HandleMessage(ctx, msg, kc.messageUnmarshaler, caughtUpOffset); err != nil {
				Logger.Error(
					"Error handling message",
					zap.String("topic", topic),
					zap.Int32("partition", partition),
					zap.Int64("offset", msg.Offset),
					zap.ByteString("key", msg.Key),
					zap.String("message", string(msg.Value)),
					zap.Error(err))
			}
			timer.ObserveDuration()
			kc.kafkaConfig.messagesProcessed.With(promLabels).Add(1)
			if msg.Offset == caughtUpOffset {
				catchupWg.Done()
				Logger.Debug(
					"Successfully read to target Kafka offset",
					zap.String("topic", topic), zap.Int32("partition", partition),
					zap.Int64("offset", msg.Offset))
			}
		case err := <-partitionConsumer.Errors():
			kc.kafkaConfig.errorsProcessed.With(promLabels).Add(1)
			Logger.Error("Encountered an error from Kafka", zap.Error(err))
		case <-ctx.Done():
			wg.Done()
			Logger.Debug(
				"Kafka partition consumer closed", zap.String("topic", topic),
				zap.Int32("partition", partition))
			return
		}
	}
}

// Close Kafka producer and client
func (kp *KafkaProducer) Close() {
	kp.producer.Close()
	if !kp.client.Closed() {
		kp.client.Close()
	}
}

// RunProducer wraps the sarama AsyncProducer and adds metrics and logging
// to the producer
func (kp *KafkaProducer) RunProducer(
	ctx context.Context,
	messages <-chan *sarama.ProducerMessage,
	wg *sync.WaitGroup,
) {
	promLabels := prometheus.Labels{
		"client": kp.kafkaConfig.ClientID,
	}
	for {
		select {
		case message := <-messages:
			kp.producer.Input() <- message
		case err := <-kp.producer.Errors():
			key, _ := err.Msg.Key.Encode()
			Logger.Error(
				"Error producing Kafka message",
				zap.String("topic", err.Msg.Topic),
				zap.String("key", string(key)),
				zap.Int32("partition", err.Msg.Partition),
				zap.Int64("offset", err.Msg.Offset),
				zap.Error(err))
			promLabels["partition"] = fmt.Sprintf("%d", err.Msg.Partition)
			promLabels["topic"] = err.Msg.Topic
			kp.kafkaConfig.errorsProduced.With(promLabels).Add(1)
		case msg := <-kp.producer.Successes():
			promLabels["partition"] = fmt.Sprintf("%d", msg.Partition)
			promLabels["topic"] = msg.Topic
			kp.kafkaConfig.messagesProduced.With(promLabels).Add(1)
		case <-ctx.Done():
			wg.Done()
			Logger.Debug("Kafka producer closed")
			return
		}
	}
}
