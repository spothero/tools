package core

import (
	"github.com/prometheus/client_golang/prometheus"
)

// CacheMetrics defines an interface for recording cache metrics
type CacheMetrics interface {
	Hit()
	Miss()
	Set()
	SetCollision()
	DeleteHit()
	DeleteMiss()
	PurgeHit()
	PurgeMiss()
}

var (
	hits           *prometheus.CounterVec
	misses         *prometheus.CounterVec
	sets           *prometheus.CounterVec
	setsCollisions *prometheus.CounterVec
	deletesHits    *prometheus.CounterVec
	deletesMisses  *prometheus.CounterVec
	purgesHits     *prometheus.CounterVec
	purgesMisses   *prometheus.CounterVec
)

// PrometheusCacheMetrics surfaces cache metrics for usage with Prometheus
type PrometheusCacheMetrics struct {
	client         string
	name           string
	hits           *prometheus.CounterVec
	misses         *prometheus.CounterVec
	sets           *prometheus.CounterVec
	setsCollisions *prometheus.CounterVec
	deletesHits    *prometheus.CounterVec
	deletesMisses  *prometheus.CounterVec
	purgesHits     *prometheus.CounterVec
	purgesMisses   *prometheus.CounterVec
}

// NewPrometheusCacheMetrics creates and returns a Prometheus cache metrics recorder
func NewPrometheusCacheMetrics(client, cacheName string) *PrometheusCacheMetrics {
	labels := []string{"client", "cache_name"}
	if hits == nil {
		hits = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_hits",
				Help: "Total number of cache hits",
			},
			labels,
		)
		prometheus.MustRegister(hits)
	}
	if misses == nil {
		misses = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_misses",
				Help: "Total number of cache misses",
			},
			labels,
		)
		prometheus.MustRegister(misses)
	}
	if sets == nil {
		sets = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_sets",
				Help: "Total number of cache sets",
			},
			labels,
		)
		prometheus.MustRegister(sets)
	}
	if setsCollisions == nil {
		setsCollisions = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_sets_collisions",
				Help: "Total number of cache sets collisions",
			},
			labels,
		)
		prometheus.MustRegister(setsCollisions)
	}
	if deletesHits == nil {
		deletesHits = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_deletes_hits",
				Help: "Total number of cache deletes hits",
			},
			labels,
		)
		prometheus.MustRegister(deletesHits)
	}
	if deletesMisses == nil {
		deletesMisses = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_deletes_misses",
				Help: "Total number of cache deletes misses",
			},
			labels,
		)
		prometheus.MustRegister(deletesMisses)
	}
	if purgesHits == nil {
		purgesHits = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_purges_hits",
				Help: "Total number of cache purges hits",
			},
			labels,
		)
		prometheus.MustRegister(purgesHits)
	}
	if purgesMisses == nil {
		purgesMisses = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_purges_misses",
				Help: "Total number of cache purges misses",
			},
			labels,
		)
		prometheus.MustRegister(purgesMisses)
	}
	return &PrometheusCacheMetrics{
		client:         client,
		name:           cacheName,
		hits:           hits,
		misses:         misses,
		sets:           sets,
		setsCollisions: setsCollisions,
		deletesHits:    deletesHits,
		deletesMisses:  deletesMisses,
		purgesHits:     purgesHits,
		purgesMisses:   purgesMisses,
	}
}

// Hit defines a cache hit
func (pcm *PrometheusCacheMetrics) Hit() {
	pcm.hits.WithLabelValues(pcm.client, pcm.name).Inc()
}

// Miss defines a cache miss
func (pcm *PrometheusCacheMetrics) Miss() {
	pcm.misses.WithLabelValues(pcm.client, pcm.name).Inc()
}

// Set defines a cache set
func (pcm *PrometheusCacheMetrics) Set() {
	pcm.sets.WithLabelValues(pcm.client, pcm.name).Inc()
}

// SetCollision defines a cache set collision
func (pcm *PrometheusCacheMetrics) SetCollision() {
	pcm.setsCollisions.WithLabelValues(pcm.client, pcm.name).Inc()
}

// DeleteHit defines a deletion hit from cache
func (pcm *PrometheusCacheMetrics) DeleteHit() {
	pcm.deletesHits.WithLabelValues(pcm.client, pcm.name).Inc()
}

// DeleteMiss defines a deletion miss from cache
func (pcm *PrometheusCacheMetrics) DeleteMiss() {
	pcm.deletesMisses.WithLabelValues(pcm.client, pcm.name).Inc()
}

// PurgeHit defines a purge hit of cache
func (pcm *PrometheusCacheMetrics) PurgeHit() {
	pcm.purgesHits.WithLabelValues(pcm.client, pcm.name).Inc()
}

// PurgeMiss defines a purge miss of cache
func (pcm *PrometheusCacheMetrics) PurgeMiss() {
	pcm.purgesMisses.WithLabelValues(pcm.client, pcm.name).Inc()
}
