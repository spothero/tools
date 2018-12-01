package core

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func getCounter(t *testing.T, counterVec *prometheus.CounterVec) int {
	counter, err := counterVec.GetMetricWith(prometheus.Labels{"client": "c", "cache_name": "n"})
	assert.NoError(t, err)
	pb := &dto.Metric{}
	counter.Write(pb)
	return int(pb.Counter.GetValue())
}

func deregister(pcm *PrometheusCacheMetrics) {
	prometheus.Unregister(pcm.hits)
	prometheus.Unregister(pcm.misses)
	prometheus.Unregister(pcm.sets)
	prometheus.Unregister(pcm.setsCollisions)
	prometheus.Unregister(pcm.deletesHits)
	prometheus.Unregister(pcm.deletesMisses)
	prometheus.Unregister(pcm.purgesHits)
	prometheus.Unregister(pcm.purgesMisses)
}

func TestPrometheusCacheHit(t *testing.T) {
	pcm := NewPrometheusCacheMetrics("c", "n")
	assert.Equal(t, 0, getCounter(t, pcm.hits))
	pcm.Hit()
	assert.Equal(t, 1, getCounter(t, pcm.hits))
	deregister(pcm)
}

func TestPrometheusCacheMiss(t *testing.T) {
	pcm := NewPrometheusCacheMetrics("c", "n")
	assert.Equal(t, 0, getCounter(t, pcm.misses))
	pcm.Miss()
	assert.Equal(t, 1, getCounter(t, pcm.misses))
	deregister(pcm)
}

func TestPrometheusCacheSet(t *testing.T) {
	pcm := NewPrometheusCacheMetrics("c", "n")
	assert.Equal(t, 0, getCounter(t, pcm.sets))
	pcm.Set()
	assert.Equal(t, 1, getCounter(t, pcm.sets))
	deregister(pcm)
}

func TestPrometheusCacheSetCollision(t *testing.T) {
	pcm := NewPrometheusCacheMetrics("c", "n")
	assert.Equal(t, 0, getCounter(t, pcm.setsCollisions))
	pcm.SetCollision()
	assert.Equal(t, 1, getCounter(t, pcm.setsCollisions))
	deregister(pcm)
}

func TestPrometheusCacheDeleteHit(t *testing.T) {
	pcm := NewPrometheusCacheMetrics("c", "n")
	assert.Equal(t, 0, getCounter(t, pcm.deletesHits))
	pcm.DeleteHit()
	assert.Equal(t, 1, getCounter(t, pcm.deletesHits))
	deregister(pcm)
}

func TestPrometheusCacheDeleteMiss(t *testing.T) {
	pcm := NewPrometheusCacheMetrics("c", "n")
	assert.Equal(t, 0, getCounter(t, pcm.deletesMisses))
	pcm.DeleteMiss()
	assert.Equal(t, 1, getCounter(t, pcm.deletesMisses))
	deregister(pcm)
}

func TestPrometheusCachePurgeHit(t *testing.T) {
	pcm := NewPrometheusCacheMetrics("c", "n")
	assert.Equal(t, 0, getCounter(t, pcm.purgesHits))
	pcm.PurgeHit()
	assert.Equal(t, 1, getCounter(t, pcm.purgesHits))
	deregister(pcm)
}

func TestPrometheusCachePurgeMiss(t *testing.T) {
	pcm := NewPrometheusCacheMetrics("c", "n")
	assert.Equal(t, 0, getCounter(t, pcm.purgesMisses))
	pcm.PurgeMiss()
	assert.Equal(t, 1, getCounter(t, pcm.purgesMisses))
	deregister(pcm)
}
