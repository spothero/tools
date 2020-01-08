package kafka

import (
	"fmt"
	"sync"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/Shopify/sarama/mocks"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAsyncProducer(t *testing.T) AsyncProducer {
	t.Helper()
	cfg := sarama.NewConfig()
	cfg.Producer.Return.Successes = true
	cfg.Producer.Return.Errors = true
	mockSaramaProducer := mocks.NewAsyncProducer(t, cfg)
	registry := prometheus.NewRegistry()
	metrics, err := NewProducerMetrics(registry)
	require.NoError(t, err)
	return AsyncProducer{
		AsyncProducer: mockSaramaProducer,
		successes:     make(chan *sarama.ProducerMessage),
		errors:        make(chan *sarama.ProducerError),
		asyncShutdown: make(chan bool),
		wg:            &sync.WaitGroup{},
		metrics:       metrics,
	}
}

func TestAsyncProducer_run(t *testing.T) {
	tests := []struct {
		name string
		fail bool
	}{
		{
			"messages are forwarded to the producer and returned when successful",
			false,
		}, {
			"messages are forwarded to the producer and return errors when failed",
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			producer := newAsyncProducer(t)
			if test.fail {
				producer.AsyncProducer.(*mocks.AsyncProducer).ExpectInputAndFail(fmt.Errorf("oh no"))
			} else {
				producer.AsyncProducer.(*mocks.AsyncProducer).ExpectInputAndSucceed()
			}
			producer.run()
			msg := &sarama.ProducerMessage{
				Value:     sarama.ByteEncoder([]byte("value")),
				Topic:     "topic",
				Partition: 0,
			}
			producer.Input() <- msg

			labels := prometheus.Labels{"topic": "topic", "partition": "0"}
			if test.fail {
				msgErr := <-producer.Errors()
				assert.Equal(t, msg, msgErr.Msg)
				errored, err := producer.metrics.errorsProduced.GetMetricWith(labels)
				require.NoError(t, err)
				erroredMetric := &dto.Metric{}
				require.NoError(t, errored.Write(erroredMetric))
				assert.Equal(t, float64(1), erroredMetric.Gauge.GetValue())
			} else {
				assert.Equal(t, msg, <-producer.Successes())
				produced, err := producer.metrics.messagesProduced.GetMetricWith(labels)
				require.NoError(t, err)
				producedMetric := &dto.Metric{}
				require.NoError(t, produced.Write(producedMetric))
				assert.Equal(t, float64(1), producedMetric.Gauge.GetValue())
			}
		})
	}
}

func TestAsyncProducer_Successes(t *testing.T) {
	assert.NotNil(t, AsyncProducer{successes: make(chan *sarama.ProducerMessage)}.Successes())
}

func TestAsyncProducer_Errors(t *testing.T) {
	assert.NotNil(t, AsyncProducer{errors: make(chan *sarama.ProducerError)}.Errors())
}

func TestAsyncProducer_AsyncClose(t *testing.T) {
	producer := newAsyncProducer(t)
	producer.run()

	// produce some messages to the underlying consumer and ensure it's drained when closed
	goodMsg := &sarama.ProducerMessage{}
	failMsg := &sarama.ProducerMessage{}
	producer.AsyncProducer.(*mocks.AsyncProducer).ExpectInputAndSucceed()
	producer.Input() <- goodMsg
	producer.AsyncProducer.(*mocks.AsyncProducer).ExpectInputAndFail(fmt.Errorf("error"))
	producer.Input() <- failMsg

	producer.AsyncClose()

	// ensure the wrapped channels are closed
	assert.NotNil(t, <-producer.Successes())
	assert.NotNil(t, <-producer.Errors())
	assert.Panics(t, func() {
		producer.successes <- &sarama.ProducerMessage{}
	})
	assert.Panics(t, func() {
		producer.errors <- &sarama.ProducerError{}
	})
}

func TestAsyncProducer_Close(t *testing.T) {
	producer := newAsyncProducer(t)
	producer.run()
	assert.NoError(t, producer.Close())
	assert.Panics(t, func() {
		producer.successes <- &sarama.ProducerMessage{}
	})
	assert.Panics(t, func() {
		producer.errors <- &sarama.ProducerError{}
	})
}

func TestNewProducerMetrics(t *testing.T) {
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
			"error registering messages produced returns an error",
			func(t *testing.T) prometheus.Registerer {
				r := prometheus.NewRegistry()
				r.MustRegister(
					prometheus.NewGaugeVec(
						prometheus.GaugeOpts{Name: "kafka_messages_produced"},
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
						prometheus.GaugeOpts{Name: "kafka_errors_produced"},
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
			metrics, err := NewProducerMetrics(r)
			if test.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, metrics.messagesProduced)
			assert.NotNil(t, metrics.errorsProduced)
		})
	}
}
