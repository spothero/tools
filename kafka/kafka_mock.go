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
	"sync"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/mock"
)

// MockKafkaConsumer implements KafkaConsumerIface for testing purposes
type MockKafkaConsumer struct {
	mock.Mock
	sync.Mutex
	readResult chan PartitionOffsets
}

// ConsumeTopic mocks the Kafka consumer ConsumeTopic method
func (m *MockKafkaConsumer) ConsumeTopic(ctx context.Context, handler MessageHandler, topic string, offsets PartitionOffsets, readResult chan PartitionOffsets, catchupWg *sync.WaitGroup, exitAfterCaughtUp bool) error {
	catchupWg.Done()
	m.Lock()
	m.readResult = readResult
	m.Unlock()
	return m.Called(handler, topic, offsets, readResult, catchupWg, exitAfterCaughtUp).Error(0)
}

// ConsumeTopicFromBeginning mocks the Kafka consumer ConsumeTopicFromBeginning method
func (m *MockKafkaConsumer) ConsumeTopicFromBeginning(ctx context.Context, handler MessageHandler, topic string, readResult chan PartitionOffsets, catchupWg *sync.WaitGroup, exitAfterCaughtUp bool) error {
	catchupWg.Done()
	m.Lock()
	m.readResult = readResult
	m.Unlock()
	return m.Called(handler, topic, readResult, catchupWg, exitAfterCaughtUp).Error(0)
}

// ConsumeTopicFromLatest mocks the Kafka consumer ConsumeTopicFromLatest method
func (m *MockKafkaConsumer) ConsumeTopicFromLatest(ctx context.Context, handler MessageHandler, topic string, readResult chan PartitionOffsets) error {
	m.Lock()
	m.readResult = readResult
	m.Unlock()
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

// MockClient implements ClientIface for testingn purposes
type MockClient struct {
	mock.Mock
}

// NewConsumer creates a new mock consumer
func (m *MockClient) NewConsumer() (ConsumerIface, error) {
	args := m.Called()
	return args.Get(0).(ConsumerIface), args.Error(1)
}

// NewProducer creates a new mock producer
func (m *MockClient) NewProducer(returnMessages bool) (ProducerIface, error) {
	args := m.Called(returnMessages)
	return args.Get(0).(ProducerIface), args.Error(1)
}

type MockProducer struct {
	mock.Mock
	Messages chan *sarama.ProducerMessage
	Success  chan *sarama.ProducerMessage
	Errors   chan *sarama.ProducerError
}

func NewMockProducer(wg *sync.WaitGroup) *MockProducer {
	return &MockProducer{
		Messages: make(chan *sarama.ProducerMessage),
		Success:  make(chan *sarama.ProducerMessage),
		Errors:   make(chan *sarama.ProducerError),
	}
}

func (m *MockProducer) RunProducer(messages <-chan *sarama.ProducerMessage, done chan bool) {
	m.Called(messages, done)
}
