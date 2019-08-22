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
	"fmt"
	"sync"

	"github.com/Shopify/sarama"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/pflag"
	"github.com/spothero/tools/log"
	"go.uber.org/zap"
)

// MessageUnmarshaler defines an interface for unmarshaling messages received from Kafka to Go types
type MessageUnmarshaler interface {
	UnmarshalMessage(ctx context.Context, msg *sarama.ConsumerMessage, target interface{}) error
}

// MessageHandler defines an interface for handling new messages received by the Kafka consumer
type MessageHandler interface {
	HandleMessage(ctx context.Context, msg *sarama.ConsumerMessage, unmarshaler MessageUnmarshaler) error
}

// ConsumerIface is an interface for consuming messages from a Kafka topic
type ConsumerIface interface {
	ConsumeTopic(ctx context.Context, handler MessageHandler, topic string, offsets PartitionOffsets, readResult chan PartitionOffsets, catchupWg *sync.WaitGroup, exitAfterCaughtUp bool) error
	ConsumeTopicFromBeginning(ctx context.Context, handler MessageHandler, topic string, readResult chan PartitionOffsets, catchupWg *sync.WaitGroup, exitAfterCaughtUp bool) error
	ConsumeTopicFromLatest(ctx context.Context, handler MessageHandler, topic string, readResult chan PartitionOffsets) error
	Close()
}

// ConsumerMetrics is a collection of Prometheus metrics for tracking a Kafka consumer's performance
type ConsumerMetrics struct {
	MessagesProcessed     *prometheus.GaugeVec
	MessageErrors         *prometheus.GaugeVec
	MessageProcessingTime *prometheus.SummaryVec
	ErrorsProcessed       *prometheus.GaugeVec
}

// ConsumerConfig contains consumer-specific configuration including whether or not to use JSON deserialization
// and schema registry configuration.
type ConsumerConfig struct {
	JSONEnabled    bool
	SchemaRegistry *SchemaRegistryConfig
}

// Registers high-level consumer flags with pflags
func (c *ConsumerConfig) RegisterFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&c.JSONEnabled, "enable-json", true, "When this flag is set, messages from Kafka will be consumed as JSON instead of Avro")
	c.SchemaRegistry.RegisterFlags(flags)
}

// Consumer is a high-level Kafka consumer that can consume off all partitions on a given topic, tracks metrics,
// provides optional logging, and unmarshals messages before passing them off to a handler.
type Consumer struct {
	metrics            ConsumerMetrics
	client             Client
	consumer           sarama.Consumer
	messageUnmarshaler MessageUnmarshaler
	logger             *zap.Logger
}

// NewConsumer sets up a high-level Kafka consumer that can be used for reading all partitions
// of a given topic a given offset. NewConsumer also sets up Prometheus metrics with the default
// registerer and configures schema registry if set in the configuration.
func (c Client) NewConsumer(config ConsumerConfig, logger *zap.Logger) (ConsumerIface, error) {
	consumer, err := sarama.NewConsumerFromClient(c.SaramaClient)
	if err != nil {
		return Consumer{}, err
	}
	kafkaConsumer := Consumer{
		client:   c,
		consumer: consumer,
		metrics:  RegisterConsumerMetrics(prometheus.DefaultRegisterer),
	}
	messageUnmarshaler := &messageDecoder{}
	if config.JSONEnabled {
		kafkaConsumer.messageUnmarshaler = &jsonMessageUnmarshaler{messageUnmarshaler: messageUnmarshaler}
	} else {
		config.SchemaRegistry.client = &schemaRegistryClient{}
		config.SchemaRegistry.messageUnmarshaler = messageUnmarshaler
		kafkaConsumer.messageUnmarshaler = config.SchemaRegistry
	}
	if logger != nil {
		kafkaConsumer.logger = logger
	} else {
		kafkaConsumer.logger = zap.NewNop()
	}
	return kafkaConsumer, nil
}

// RegisterConsumerMetrics registers Kafka consumer metrics with the provided registerer and returns
// a struct containing consumer Prometheus metrics.
func RegisterConsumerMetrics(registerer prometheus.Registerer) ConsumerMetrics {
	promLabels := []string{"topic", "partition", "client"}
	c := ConsumerMetrics{
		MessageProcessingTime: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name: "kafka_message_processing_time_seconds",
				Help: "Kafka Message processing duration in seconds",
			},
			promLabels,
		),
		MessagesProcessed: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "kafka_messages_processed",
				Help: "Number of Kafka messages processed",
			},
			promLabels,
		),
		MessageErrors: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "kafka_message_errors",
				Help: "Number of Kafka messages that couldn't be processed due to an error",
			},
			promLabels,
		),
		ErrorsProcessed: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "kafka_errors_processed",
				Help: "Number of errors received from the Kafka broker",
			},
			promLabels,
		),
	}
	registerer.MustRegister(
		c.MessageProcessingTime,
		c.MessagesProcessed,
		c.MessageErrors,
		c.ErrorsProcessed,
	)
	return c
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
// through the readResult channel.
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
		newestOffset, err := c.client.SaramaClient.GetOffset(topic, partition, sarama.OffsetNewest)
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
		"client":    c.client.ClientID,
	}
	for {
		select {
		case msg, ok := <-partitionConsumer.Messages():
			curOffset = msg.Offset
			if !ok {
				c.metrics.MessageErrors.With(promLabels).Add(1)
				c.logger.Error(
					"Unable to process message from Kafka",
					zap.ByteString("key", msg.Key), zap.Int64("offset", msg.Offset),
					zap.Int32("partition", msg.Partition), zap.String("topic", msg.Topic),
					zap.Time("message_ts", msg.Timestamp))
				continue
			}
			timer := prometheus.NewTimer(c.metrics.MessageProcessingTime.With(promLabels))
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
			c.metrics.MessagesProcessed.With(promLabels).Add(1)
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
			c.metrics.ErrorsProcessed.With(promLabels).Add(1)
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
