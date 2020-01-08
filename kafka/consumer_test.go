// Copyright 2020 SpotHero
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
	assert.NoError(t, pc.Close())
}

func TestPartitionConsumer_run(t *testing.T) {
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
	}
	pc.run("topic", 0)

	// send messages through the partition consumer
	saramaPartitionConsumer.(*mocks.PartitionConsumer).YieldMessage(&sarama.ConsumerMessage{})
	saramaPartitionConsumer.(*mocks.PartitionConsumer).YieldError(&sarama.ConsumerError{})
	<-pc.messages
	<-pc.errors

	// get the metrics out of the prometheus registry
	labels := prometheus.Labels{"topic": "topic", "partition": "0"}
	processed, err := metrics.messagesConsumed.GetMetricWith(labels)
	require.NoError(t, err)
	processedMetric := &dto.Metric{}
	require.NoError(t, processed.Write(processedMetric))
	errored, err := metrics.errorsConsumed.GetMetricWith(labels)
	require.NoError(t, err)
	erroredMetric := &dto.Metric{}
	require.NoError(t, errored.Write(erroredMetric))

	// ensure that the metrics have been updated
	assert.Equal(t, float64(1), processedMetric.Gauge.GetValue())
	assert.Equal(t, float64(1), erroredMetric.Gauge.GetValue())
}

func TestPartitionConsumer_Messages(t *testing.T) {
	assert.NotNil(t, PartitionConsumer{messages: make(chan *sarama.ConsumerMessage)}.Messages())
}

func TestPartitionConsumer_Errors(t *testing.T) {
	assert.NotNil(t, PartitionConsumer{errors: make(chan *sarama.ConsumerError)}.Errors())
}

func TestNewConsumerMetrics(t *testing.T) {
	tests := []struct {
		name       string
		registerer func(t *testing.T) prometheus.Registerer
		expectErr  bool
	}{
		{
			"new metrics are registered and returned",
			func(t *testing.T) prometheus.Registerer {
				return prometheus.NewRegistry()
			},
			false,
		}, {
			"error registering messages processed returns an error",
			func(t *testing.T) prometheus.Registerer {
				r := prometheus.NewRegistry()
				r.MustRegister(
					prometheus.NewGaugeVec(
						prometheus.GaugeOpts{Name: "kafka_messages_consumed"},
						[]string{"topic", "partition"},
					),
				)
				return r
			},
			true,
		}, {
			"error registering messages errored returns an error",
			func(t *testing.T) prometheus.Registerer {
				r := prometheus.NewRegistry()
				r.MustRegister(
					prometheus.NewGaugeVec(
						prometheus.GaugeOpts{Name: "kafka_errors_consumed"},
						[]string{"topic", "partition"},
					),
				)
				return r
			},
			true,
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
