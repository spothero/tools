package tools

import (
	"context"
	"sync"

	"github.com/stretchr/testify/mock"
)

// MockKafkaConsumer implements KafkaConsumerIface for testing purposes
type MockKafkaConsumer struct {
	mock.Mock
	readResult chan PartitionOffsets
}

// ConsumeTopic mocks the Kafka consumer ConsumeTopic method
func (m *MockKafkaConsumer) ConsumeTopic(ctx context.Context, handler KafkaMessageHandler, topic string, offsets PartitionOffsets, readResult chan PartitionOffsets, catchupWg *sync.WaitGroup, exitAfterCaughtUp bool) error {
	catchupWg.Done()
	m.readResult = readResult
	return m.Called(handler, topic, offsets, readResult, catchupWg, exitAfterCaughtUp).Error(0)
}

// ConsumeTopicFromBeginning mocks the Kafka consumer ConsumeTopicFromBeginning method
func (m *MockKafkaConsumer) ConsumeTopicFromBeginning(ctx context.Context, handler KafkaMessageHandler, topic string, readResult chan PartitionOffsets, catchupWg *sync.WaitGroup, exitAfterCaughtUp bool) error {
	catchupWg.Done()
	m.readResult = readResult
	return m.Called(handler, topic, readResult, catchupWg, exitAfterCaughtUp).Error(0)
}

// ConsumeTopicFromLatest mocks the Kafka consumer ConsumeTopicFromLatest method
func (m *MockKafkaConsumer) ConsumeTopicFromLatest(ctx context.Context, handler KafkaMessageHandler, topic string, readResult chan PartitionOffsets) error {
	m.readResult = readResult
	return m.Called(handler, topic, readResult).Error(0)
}

// Close mocks the Kafka consumer Close method
func (m *MockKafkaConsumer) Close() {
	m.Called()
}

// EmitReadResult allows tests to send values through the readResult channel passed into the mock consumer.
func (m *MockKafkaConsumer) EmitReadResult(offsets PartitionOffsets) {
	m.readResult <- offsets
}
