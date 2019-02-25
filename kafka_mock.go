package tools

import (
	"context"
	"github.com/stretchr/testify/mock"
	"sync"
)

// MockKafkaConsumer implements KafkaConsumerIface for testing purposes
type MockKafkaConsumer struct {
	mock.Mock
}

// ConsumeTopic mocks the Kafka consumer ConsumeTopic method
func (m MockKafkaConsumer) ConsumeTopic(ctx context.Context, handler KafkaMessageHandler, topic string, offsets PartitionOffsets, readResult chan PartitionOffsets, catchupWg *sync.WaitGroup, exitAfterCaughtUp bool) error {
	return m.Called(handler, topic, offsets, readResult, catchupWg, exitAfterCaughtUp).Error(0)
}

// ConsumeTopicFromBeginning mocks the Kafka consumer ConsumeTopicFromBeginning method
func (m MockKafkaConsumer) ConsumeTopicFromBeginning(ctx context.Context, handler KafkaMessageHandler, topic string, readResult chan PartitionOffsets, catchupWg *sync.WaitGroup, exitAfterCaughtUp bool) error {
	return m.Called(handler, topic, topic, readResult, catchupWg, exitAfterCaughtUp).Error(0)
}

// ConsumeTopicFromLatest mocks the Kafka consumer ConsumeTopicFromLatest method
func (m MockKafkaConsumer) ConsumeTopicFromLatest(ctx context.Context, handler KafkaMessageHandler, topic string, readResult chan PartitionOffsets) error {
	return m.Called(handler, topic, readResult).Error(0)
}

// Close mocks the Kafka consumer Close method
func (m MockKafkaConsumer) Close() {
	m.Called()
}

