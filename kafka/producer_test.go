package kafka

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestRegisterProducerMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := RegisterProducerMetrics(registry)
	assert.NotNil(t, metrics.MessagesProduced)
	assert.NotNil(t, metrics.ErrorsProduced)
}
