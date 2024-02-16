// Copyright 2023 SpotHero
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
	"sync"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/Shopify/sarama/mocks"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsumer_ConsumePartition(t *testing.T) {
	mockSaramaConsumer := mocks.NewConsumer(t, nil)
	consumer := Consumer{Consumer: mockSaramaConsumer}
	mockSaramaConsumer.ExpectConsumePartition("topic", 0, 1)
	pc, err := consumer.ConsumePartition("topic", 0, 1)
	assert.NoError(t, err)
	assert.NotNil(t, pc.(PartitionConsumer).PartitionConsumer)
	assert.NotNil(t, pc.(PartitionConsumer).messages)
	assert.NotNil(t, pc.(PartitionConsumer).errors)
	assert.NotNil(t, pc.(PartitionConsumer).closeWg)
	assert.NotNil(t, pc.(PartitionConsumer).metrics)
	assert.NoError(t, pc.Close())
}

func newPartitionConsumer(t *testing.T) PartitionConsumer {
	t.Helper()
	mockSaramaConsumer := mocks.NewConsumer(t, nil)
	mockSaramaConsumer.ExpectConsumePartition("topic", 0, 0)
	saramaPartitionConsumer, err := mockSaramaConsumer.ConsumePartition("topic", 0, 0)
	require.NoError(t, err)
	registry := prometheus.NewRegistry()
	metrics, err := NewConsumerMetrics(registry)
	require.NoError(t, err)
	pc := PartitionConsumer{
		PartitionConsumer: saramaPartitionConsumer,
		metrics:           metrics,
		messages:          make(chan *sarama.ConsumerMessage, 1),
		errors:            make(chan *sarama.ConsumerError, 1),
		closeWg:           &sync.WaitGroup{},
	}
	return pc
}

func TestPartitionConsumer_run(t *testing.T) {
	pc := newPartitionConsumer(t)
	pc.run("topic", 0)

	// send messages through the partition consumer
	pc.PartitionConsumer.(*mocks.PartitionConsumer).YieldMessage(&sarama.ConsumerMessage{})
	pc.PartitionConsumer.(*mocks.PartitionConsumer).YieldError(&sarama.ConsumerError{})
	<-pc.messages
	<-pc.errors

	// get the metrics out of the prometheus registry
	labels := prometheus.Labels{"topic": "topic", "partition": "0"}
	processed, err := pc.metrics.messagesConsumed.GetMetricWith(labels)
	require.NoError(t, err)
	processedMetric := &dto.Metric{}
	require.NoError(t, processed.Write(processedMetric))
	errored, err := pc.metrics.errorsConsumed.GetMetricWith(labels)
	require.NoError(t, err)
	erroredMetric := &dto.Metric{}
	require.NoError(t, errored.Write(erroredMetric))

	// ensure that the metrics have been updated
	assert.Equal(t, float64(1), processedMetric.Counter.GetValue())
	assert.Equal(t, float64(1), erroredMetric.Counter.GetValue())
}

func TestPartitionConsumer_Messages(t *testing.T) {
	assert.NotNil(t, PartitionConsumer{messages: make(chan *sarama.ConsumerMessage)}.Messages())
}

func TestPartitionConsumer_Errors(t *testing.T) {
	assert.NotNil(t, PartitionConsumer{errors: make(chan *sarama.ConsumerError)}.Errors())
}

func TestPartitionConsumer_AsyncClose(t *testing.T) {
	pc := newPartitionConsumer(t)
	pc.PartitionConsumer.(*mocks.PartitionConsumer).ExpectMessagesDrainedOnClose()
	pc.PartitionConsumer.(*mocks.PartitionConsumer).ExpectErrorsDrainedOnClose()
	pc.PartitionConsumer.(*mocks.PartitionConsumer).YieldMessage(&sarama.ConsumerMessage{})
	pc.PartitionConsumer.(*mocks.PartitionConsumer).YieldError(&sarama.ConsumerError{})
	pc.AsyncClose()
	<-pc.messages
	<-pc.errors

	// make sure the channels were closed after they were drained by attempting to send a message through
	assert.Panics(t, func() {
		pc.messages <- &sarama.ConsumerMessage{}
	})
	assert.Panics(t, func() {
		pc.errors <- &sarama.ConsumerError{}
	})
}

func TestPartitionConsumer_Close(t *testing.T) {
	pc := newPartitionConsumer(t)
	pc.PartitionConsumer.(*mocks.PartitionConsumer).YieldMessage(&sarama.ConsumerMessage{})
	pc.PartitionConsumer.(*mocks.PartitionConsumer).YieldError(&sarama.ConsumerError{})
	err := pc.Close()
	assert.NotNil(t, err)

	// make sure the channels were closed after they were drained by attempting to send a message through
	assert.Panics(t, func() {
		pc.messages <- &sarama.ConsumerMessage{}
	})
	assert.Panics(t, func() {
		pc.errors <- &sarama.ConsumerError{}
	})
}

func TestNewConsumerMetrics(t *testing.T) {
	tests := []struct {
		registerer func(t *testing.T) prometheus.Registerer
		name       string
		expectErr  bool
	}{
		{
			name: "new metrics are registered and returned",
			registerer: func(_ *testing.T) prometheus.Registerer {
				return prometheus.NewRegistry()
			},
		}, {
			name: "error registering messages processed returns an error",
			registerer: func(_ *testing.T) prometheus.Registerer {
				r := prometheus.NewRegistry()
				r.MustRegister(
					prometheus.NewGaugeVec(
						prometheus.GaugeOpts{Name: "kafka_messages_consumed"},
						[]string{"topic", "partition"},
					),
				)
				return r
			},
			expectErr: true,
		}, {
			name: "error registering messages errored returns an error",
			registerer: func(_ *testing.T) prometheus.Registerer {
				r := prometheus.NewRegistry()
				r.MustRegister(
					prometheus.NewGaugeVec(
						prometheus.GaugeOpts{Name: "kafka_errors_consumed"},
						[]string{"topic", "partition"},
					),
				)
				return r
			},
			expectErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := test.registerer(t)
			metrics, err := NewConsumerMetrics(r)
			if test.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, metrics.messagesConsumed)
			assert.NotNil(t, metrics.errorsConsumed)
		})
	}
}
