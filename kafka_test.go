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
) error {
	tc.Called(ctx, msg, unmarshaler)
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

func setupTestConsumer(t *testing.T) (*testHandler, *KafkaConsumer, *mocks.Consumer, context.Context, context.CancelFunc) {
	mockConsumer := mocks.NewConsumer(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	kafkaConfig := &KafkaConfig{ClientID: "test"}
	kafkaConfig.initKafkaMetrics(prometheus.NewRegistry())
	kc := &KafkaConsumer{consumer: mockConsumer, kafkaClient: kafkaClient{kafkaConfig: kafkaConfig}}
	return &testHandler{}, kc, mockConsumer, ctx, cancel
}

func setupTestClient(getOffsetReturn int, getOffsetError error, kc *KafkaConsumer) {
	mockClient := &mockSaramaClient{getOffsetReturn: int64(getOffsetReturn), getOffsetErr: getOffsetError}
	kc.client = mockClient
}

func TestConsumeTopic(t *testing.T) {
	// Simulate reading messages off of one topic with five partitions
	// Note that this is more of an integration test rather than a unit test
	handler, consumer, mockSaramaConsumer, ctx, cancel := setupTestConsumer(t)
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

	var catchupWg sync.WaitGroup
	catchupWg.Add(1)
	readStatus := make(chan PartitionOffsets)
	go consumer.ConsumeTopic(ctx, handler, "test-topic", sarama.OffsetOldest, readStatus, &catchupWg, false)

	// Yield a message from each partition
	for i := range partitionConsumers {
		message := &sarama.ConsumerMessage{
			Value:     []byte{0, 1, 2, 3, 4},
			Offset:    1,
			Partition: int32(i),
		}
		handler.On("HandleMessage", mock.Anything, message, nil)
		partitionConsumers[i].YieldMessage(message)
	}
	catchupWg.Wait()

	handler.AssertNumberOfCalls(t, "HandleMessage", numPartitions)
	cancel()
	status := <-readStatus
	assert.Equal(t, PartitionOffsets{0: 1, 1: 1, 2: 1, 3: 1, 4: 1}, status)
	for i := range partitionConsumers {
		partitionConsumers[i].ExpectMessagesDrainedOnClose()
	}
}

// Make sure there's a panic if we try to read a topic with no handler defined
func TestConsumeTopic_noTopics(t *testing.T) {
	handler, consumer, mockConsumer, _, _ := setupTestConsumer(t)
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
	consumer.ConsumeTopic(nil, handler, "test-topic", sarama.OffsetOldest, nil, nil, false)
}

// Make sure there's a panic trying to get the newest offset from a topic & partition
func TestConsumeTopic_errorGettingOffset(t *testing.T) {
	handler, consumer, mockSaramaConsumer, _, _ := setupTestConsumer(t)
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
	consumer.ConsumeTopic(nil, handler, "test-topic", sarama.OffsetOldest, nil, nil, false)
}

// Test that processing messages from a partition works
func TestConsumePartition(t *testing.T) {
	handler, consumer, mockSaramaConsumer, ctx, cancel := setupTestConsumer(t)
	defer mockSaramaConsumer.Close()
	partitionConsumer := *mockSaramaConsumer.ExpectConsumePartition("test-topic", 0, 0)
	message := &sarama.ConsumerMessage{
		Value:  []byte{0, 1, 2, 3, 4},
		Offset: 1,
	}
	message2 := &sarama.ConsumerMessage{
		Value:  []byte{0, 1, 2, 3, 4},
		Offset: 2,
	}
	handler.On("HandleMessage", mock.Anything, mock.Anything, mock.Anything)
	readStatus := make(chan consumerLastStatus)
	var catchupWg sync.WaitGroup
	catchupWg.Add(1)

	// Start partition consumer
	go consumer.consumePartition(ctx, handler, "test-topic", 0, 0, 1, readStatus, &catchupWg, false)
	// Send a message to the consumer
	partitionConsumer.YieldMessage(message)
	// Make sure read to offset 1 before being "caught up"
	catchupWg.Wait()
	handler.AssertNumberOfCalls(t, "HandleMessage", 1)

	// Send another message
	partitionConsumer.YieldMessage(message2)

	// Shutdown consumer
	cancel()
	status := <-readStatus
	assert.Equal(t, consumerLastStatus{offset: 2, partition: 0}, status)
	handler.AssertNumberOfCalls(t, "HandleMessage", 2)
	partitionConsumer.ExpectMessagesDrainedOnClose()
}

// Test that we're "caught up" if there aren't any messages to process
func TestConsumePartition_caughtUp(t *testing.T) {
	handler, consumer, mockSaramaConsumer, ctx, cancel := setupTestConsumer(t)
	defer mockSaramaConsumer.Close()
	partitionConsumer := *mockSaramaConsumer.ExpectConsumePartition("test-topic", 0, 0)
	message := &sarama.ConsumerMessage{
		Value: []byte{0, 1, 2, 3, 4},
	}
	readStatus := make(chan consumerLastStatus)
	var catchupWg sync.WaitGroup
	catchupWg.Add(1)

	// Starts at offset -1 which means there are no messages on the partition
	go consumer.consumePartition(ctx, handler, "test-topic", 0, 0, -1, readStatus, &catchupWg, false)
	catchupWg.Wait()
	cancel()
	<-readStatus
	handler.AssertNotCalled(t, "HandleMessage", message)
	partitionConsumer.ExpectMessagesDrainedOnClose()
}

// Test that the function exits after catching up if specified
func TestConsumePartition_exitAfterCaughtUp(t *testing.T) {
	handler, consumer, mockSaramaConsumer, ctx, _ := setupTestConsumer(t)
	defer mockSaramaConsumer.Close()
	partitionConsumer := *mockSaramaConsumer.ExpectConsumePartition("test-topic", 0, 0)
	message := &sarama.ConsumerMessage{
		Value:  []byte{0, 1, 2, 3, 4},
		Offset: 1,
	}
	handler.On("HandleMessage", mock.Anything, mock.Anything, mock.Anything)
	readStatus := make(chan consumerLastStatus)
	var catchupWg sync.WaitGroup
	catchupWg.Add(1)

	// Start partition consumer
	go consumer.consumePartition(ctx, handler, "test-topic", 0, 0, 1, readStatus, &catchupWg, true)
	// Send a message to the consumer
	partitionConsumer.YieldMessage(message)
	// Make sure read to offset 1 before being "caught up"
	catchupWg.Wait()
	handler.AssertNumberOfCalls(t, "HandleMessage", 1)

	defer func() {
		r := recover()
		assert.NotNil(t, r)
	}()

	// this should panic because the channel is closed if the consumer exited
	partitionConsumer.YieldMessage(message)
}

// Test that the consumer handles errors from Kafka
func TestConsumePartition_handleError(t *testing.T) {
	handler, consumer, mockSaramaConsumer, ctx, cancel := setupTestConsumer(t)
	defer mockSaramaConsumer.Close()
	partitionConsumer := *mockSaramaConsumer.ExpectConsumePartition("test-topic", 0, 0)
	readStatus := make(chan consumerLastStatus)
	var catchupWg sync.WaitGroup
	catchupWg.Add(1)

	// Start partition consumer
	go consumer.consumePartition(ctx, handler, "test-topic", 0, 0, -1, readStatus, &catchupWg, false)
	// Send an error to the consumer
	partitionConsumer.YieldError(&sarama.ConsumerError{})

	// Shutdown consumer
	cancel()

	<-readStatus
	handler.AssertNotCalled(t, "HandleMessage")
	partitionConsumer.ExpectErrorsDrainedOnClose()
}
