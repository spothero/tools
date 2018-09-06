package core

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/Shopify/sarama/mocks"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Create a mock message handler
type testHandler struct {
	mock.Mock
}

func (tc *testHandler) HandleMessage(
	ctx context.Context,
	msg *sarama.ConsumerMessage,
	unmarshaler KafkaMessageUnmarshaler,
	caughtUpOffset int64,
) error {
	tc.Called(msg, caughtUpOffset, ctx)
	return nil
}

// Mock Sarama client
type mockSaramaClient struct {
	sarama.Client
	getOffsetReturn int64
	getOffsetErr    error
}

// Mock GetOffset on the Sarama client
func (msc *mockSaramaClient) GetOffset(topic string, partitionID int32, time int64) (int64, error) {
	return msc.getOffsetReturn, msc.getOffsetErr
}

func setupTestConsumer(t *testing.T) (*testHandler, map[string]KafkaMessageHandler, *KafkaConsumer, *mocks.Consumer, context.Context, context.CancelFunc) {
	handler := &testHandler{}
	handlers := make(map[string]KafkaMessageHandler)
	handlers["test-topic"] = handler
	mockConsumer := mocks.NewConsumer(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	kafkaConfig := &KafkaConfig{ClientID: "test"}
	kafkaConfig.initKafkaMetrics(prometheus.NewRegistry())
	kc := &KafkaConsumer{consumer: mockConsumer, kafkaClient: kafkaClient{kafkaConfig: kafkaConfig}}
	return handler, handlers, kc, mockConsumer, ctx, cancel
}

func setupTestClient(getOffsetReturn int, getOffsetError error, kc *KafkaConsumer) {
	mockClient := &mockSaramaClient{getOffsetReturn: int64(getOffsetReturn), getOffsetErr: getOffsetError}
	kc.client = mockClient
}

func TestConsumeTopic(t *testing.T) {
	// Simulate reading messages off of one topic with five partitions
	// Note that this is more of an integration test rather than a unit test
	handler, handlers, consumer, mockSaramaConsumer, ctx, cancel := setupTestConsumer(t)
	defer mockSaramaConsumer.Close()
	setupTestClient(2, nil, consumer)

	// Setup 5 partitions on the topic
	numPartitions := 5
	mockSaramaConsumer.SetTopicMetadata(map[string][]int32{
		"test-topic": {0, 1, 2, 3, 4},
	})
	partitionConsumers := make([]mocks.PartitionConsumer, numPartitions)
	for i := 0; i < numPartitions; i++ {
		partitionConsumers[i] = *mockSaramaConsumer.ExpectConsumePartition("test-topic", int32(i), sarama.OffsetOldest)
	}

	var wg sync.WaitGroup
	var catchupWg sync.WaitGroup
	wg.Add(1)
	catchupWg.Add(1)
	go consumer.ConsumeTopic(ctx, handlers, "test-topic", sarama.OffsetOldest, &wg, &catchupWg)

	// Yield a message from each partition
	for i := range partitionConsumers {
		message := &sarama.ConsumerMessage{
			Value:     []byte{0, 1, 2, 3, 4},
			Offset:    1,
			Partition: int32(i),
		}
		handler.On("HandleMessage", message, message.Offset, ctx)
		partitionConsumers[i].YieldMessage(message)
	}
	catchupWg.Wait()

	handler.AssertNumberOfCalls(t, "HandleMessage", numPartitions)
	cancel()
	wg.Wait()
	for i := range partitionConsumers {
		partitionConsumers[i].ExpectMessagesDrainedOnClose()
	}
}

// Make sure there's a panic if we try to read a topic with no handler defined
func TestConsumeTopic_NoTopics(t *testing.T) {
	_, handlers, consumer, mockConsumer, _, _ := setupTestConsumer(t)
	defer mockConsumer.Close()
	setupTestClient(2, nil, consumer)
	defer func() {
		if r := recover(); r != nil {
			assert.Contains(t, "Couldn't get Kafka partitions", r)
		}
	}()
	mockConsumer.SetTopicMetadata(map[string][]int32{
		"test-topic": nil,
	})
	consumer.ConsumeTopic(nil, handlers, "test-topic", sarama.OffsetOldest, nil, nil)
}

// Make sure there's a panic if there's an error listing partitions on a topic
func TestConsumeTopic_KafkaErrorListingTopics(t *testing.T) {
	handler, handlers, consumer, mockSaramaConsumer, _, _ := setupTestConsumer(t)
	defer mockSaramaConsumer.Close()
	setupTestClient(2, nil, consumer)
	defer func() {
		if r := recover(); r != nil {
			assert.Contains(t, "No handler defined for topic, messages cannot be processed!", r)
			handler.AssertNumberOfCalls(t, "HandleMessage", 0)
		}
	}()
	consumer.ConsumeTopic(nil, handlers, "bad-topic", sarama.OffsetOldest, nil, nil)
}

// Make sure there's a panic trying to get the newest offset from a topic & partition
func TestConsumeTopic_ErrorGettingOffset(t *testing.T) {
	handler, handlers, consumer, mockSaramaConsumer, _, _ := setupTestConsumer(t)
	defer mockSaramaConsumer.Close()
	setupTestClient(2, fmt.Errorf("some kafka error"), consumer)

	defer func() {
		if r := recover(); r != nil {
			assert.Contains(t, "Failed to get the newest offset from Kafka", r)
			handler.AssertNumberOfCalls(t, "HandleMessage", 0)
		}
	}()
	mockSaramaConsumer.SetTopicMetadata(map[string][]int32{
		"test-topic": {0},
	})
	var wg sync.WaitGroup
	consumer.ConsumeTopic(nil, handlers, "test-topic", sarama.OffsetOldest, &wg, &wg)
}

// Test that processing messages from a partition works
func TestConsumePartition(t *testing.T) {
	handler, _, consumer, mockSaramaConsumer, ctx, cancel := setupTestConsumer(t)
	defer mockSaramaConsumer.Close()
	partitionConsumer := *mockSaramaConsumer.ExpectConsumePartition("test-topic", 0, 0)
	defer mockSaramaConsumer.Close()
	message := &sarama.ConsumerMessage{
		Value:  []byte{0, 1, 2, 3, 4},
		Offset: 1,
	}
	message2 := &sarama.ConsumerMessage{
		Value:  []byte{0, 1, 2, 3, 4},
		Offset: 2,
	}
	handler.On("HandleMessage", mock.Anything, mock.Anything, mock.Anything)
	var wg sync.WaitGroup
	var catchupWg sync.WaitGroup
	catchupWg.Add(1)
	wg.Add(1)

	// Start partition consumer
	go consumer.consumePartition(ctx, handler, "test-topic", 0, 0, 1, &wg, &catchupWg)
	// Send a message to the consumer
	partitionConsumer.YieldMessage(message)
	// Make sure read to offset 1 before being "caught up"
	catchupWg.Wait()
	handler.AssertNumberOfCalls(t, "HandleMessage", 1)

	// Send another message
	partitionConsumer.YieldMessage(message2)

	// Shutdown consumer
	cancel()
	wg.Wait()
	handler.AssertNumberOfCalls(t, "HandleMessage", 2)
	partitionConsumer.ExpectMessagesDrainedOnClose()
}

// Test that we're "caught up" if there aren't any messages to process
func TestConsumePartition_CaughtUp(t *testing.T) {
	handler, _, consumer, mockSaramaConsumer, ctx, cancel := setupTestConsumer(t)
	defer mockSaramaConsumer.Close()
	partitionConsumer := *mockSaramaConsumer.ExpectConsumePartition("test-topic", 0, 0)
	message := &sarama.ConsumerMessage{
		Value: []byte{0, 1, 2, 3, 4},
	}
	var wg sync.WaitGroup
	var catchupWg sync.WaitGroup
	catchupWg.Add(1)
	wg.Add(1)

	// Starts at offset -1 which means there are no messages on the partition
	go consumer.consumePartition(ctx, handler, "test-topic", 0, 0, -1, &wg, &catchupWg)
	catchupWg.Wait()
	cancel()
	wg.Wait()
	handler.AssertNotCalled(t, "HandleMessage", message)
	partitionConsumer.ExpectMessagesDrainedOnClose()
}

// Test that the consumer handles errors from Kafka
func TestConsumePartition_HandleError(t *testing.T) {
	handler, _, consumer, mockSaramaConsumer, ctx, cancel := setupTestConsumer(t)
	defer mockSaramaConsumer.Close()
	partitionConsumer := *mockSaramaConsumer.ExpectConsumePartition("test-topic", 0, 0)
	var wg sync.WaitGroup
	var catchupWg sync.WaitGroup
	wg.Add(1)
	catchupWg.Add(1)

	// Start partition consumer
	go consumer.consumePartition(ctx, handler, "test-topic", 0, 0, -1, &wg, &catchupWg)
	// Send an error to the consumer
	partitionConsumer.YieldError(&sarama.ConsumerError{})

	// Shutdown consumer
	cancel()

	wg.Wait()
	handler.AssertNotCalled(t, "HandleMessage")
	partitionConsumer.ExpectErrorsDrainedOnClose()
}
