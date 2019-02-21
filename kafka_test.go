// Copyright 2018 SpotHero
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

package tools

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/Shopify/sarama/mocks"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

func setupTestConsumer(t *testing.T, clientGetOffsetReturn int, clientGetOffsetError error) (*testHandler, KafkaConsumer, *mocks.Consumer, context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	mockConsumer := mocks.NewConsumer(t, nil)
	config := KafkaConfig{ClientID: "test"}
	config.initKafkaMetrics(prometheus.NewRegistry())
	mockClient := &mockSaramaClient{getOffsetReturn: int64(clientGetOffsetReturn), getOffsetErr: clientGetOffsetError}
	consumer := KafkaConsumer{consumer: mockConsumer, KafkaClient: KafkaClient{KafkaConfig: config, client: mockClient}}
	return &testHandler{}, consumer, consumer.consumer.(*mocks.Consumer), ctx, cancel
}

func TestConsumeTopic(t *testing.T) {
	// Simulate reading messages off of one topic with five partitions
	// Note that this is more of an integration test rather than a unit test
	handler, consumer, mockSaramaConsumer, ctx, cancel := setupTestConsumer(t, 2, nil)
	defer mockSaramaConsumer.Close()

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
	offsets := PartitionOffsets{
		0: sarama.OffsetOldest, 1: sarama.OffsetOldest, 2: sarama.OffsetOldest,
		3: sarama.OffsetOldest, 4: sarama.OffsetOldest}
	err := consumer.ConsumeTopic(ctx, handler, "test-topic", offsets, readStatus, &catchupWg, false)
	require.NoError(t, err)

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

func TestConsumeTopic_noTopics(t *testing.T) {
	handler, consumer, mockConsumer, _, _ := setupTestConsumer(t, 2, nil)
	defer mockConsumer.Close()
	mockConsumer.SetTopicMetadata(map[string][]int32{
		"test-topic": nil,
	})
	err := consumer.ConsumeTopic(nil, handler, "test-topic", PartitionOffsets{0: sarama.OffsetOldest}, nil, nil, false)
	assert.Error(t, err)
}

// Make sure there's a panic trying to get the newest offset from a topic & partition
func TestConsumeTopic_errorGettingOffset(t *testing.T) {
	handler, consumer, mockSaramaConsumer, _, _ := setupTestConsumer(t, 2, fmt.Errorf("some kafka error"))
	defer mockSaramaConsumer.Close()

	mockSaramaConsumer.SetTopicMetadata(map[string][]int32{
		"test-topic": {0},
	})
	err := consumer.ConsumeTopic(nil, handler, "test-topic", PartitionOffsets{0: sarama.OffsetOldest}, nil, nil, false)
	assert.Error(t, err)
}

func TestConsumeTopicFromBeginning(t *testing.T) {
	handler, consumer, mockSaramaConsumer, ctx, cancel := setupTestConsumer(t, 2, nil)
	defer mockSaramaConsumer.Close()

	numPartitions := 2
	mockSaramaConsumer.SetTopicMetadata(map[string][]int32{
		"test-topic": {0, 1},
	})
	partitionConsumers := make([]mocks.PartitionConsumer, numPartitions)
	for i := 0; i < numPartitions; i++ {
		partitionConsumers[i] = *mockSaramaConsumer.ExpectConsumePartition("test-topic", int32(i), sarama.OffsetOldest)
	}

	var catchupWg sync.WaitGroup
	catchupWg.Add(1)
	readStatus := make(chan PartitionOffsets)
	err := consumer.ConsumeTopicFromBeginning(ctx, handler, "test-topic", readStatus, &catchupWg, false)
	require.NoError(t, err)

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
	assert.Equal(t, PartitionOffsets{0: 1, 1: 1}, status)
	for i := range partitionConsumers {
		partitionConsumers[i].ExpectMessagesDrainedOnClose()
	}
}

func TestConsumeTopicFromLatest(t *testing.T) {
	handler, consumer, mockSaramaConsumer, ctx, cancel := setupTestConsumer(t, 0, nil)
	defer mockSaramaConsumer.Close()
	mockSaramaConsumer.SetTopicMetadata(map[string][]int32{
		"test-topic": {0},
	})
	pc := mockSaramaConsumer.ExpectConsumePartition("test-topic", int32(0), sarama.OffsetNewest)
	message := &sarama.ConsumerMessage{
		Value:     []byte{1, 2, 3, 4},
		Offset:    1,
		Partition: int32(0),
	}
	handler.On("HandleMessage", mock.Anything, message, nil)
	pc.YieldMessage(message)
	readStatus := make(chan PartitionOffsets)
	err := consumer.ConsumeTopicFromLatest(ctx, handler, "test-topic", readStatus)
	require.NoError(t, err)
	time.Sleep(1 * time.Millisecond)
	cancel()
	status := <-readStatus
	assert.Equal(t, PartitionOffsets{0: 1}, status)
}

// Test that processing messages from a partition works
func TestConsumePartition(t *testing.T) {
	handler, consumer, mockSaramaConsumer, ctx, cancel := setupTestConsumer(t, 0, nil)
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
	handler, consumer, mockSaramaConsumer, ctx, cancel := setupTestConsumer(t, 1, nil)
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
	handler, consumer, mockSaramaConsumer, ctx, _ := setupTestConsumer(t, 1, nil)
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
	go consumer.consumePartition(ctx, handler, "test-topic", 0, 0, 1, readStatus, &catchupWg, true)
	partitionConsumer.YieldMessage(message)
	<-readStatus
	handler.AssertNumberOfCalls(t, "HandleMessage", 1)
}

// Test that the consumer handles errors from Kafka
func TestConsumePartition_handleError(t *testing.T) {
	handler, consumer, mockSaramaConsumer, ctx, cancel := setupTestConsumer(t, 0, nil)
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
