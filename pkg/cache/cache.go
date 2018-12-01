package core

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"sync"
	"time"

	"github.com/allegro/bigcache"
	"github.com/gomodule/redigo/redis"
	"github.com/mna/redisc"
	opentracing "github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
)

// Private shared Redis connection pool. We share one connection pool amongst all caches.
var sharedCluster = struct {
	cluster    *redisc.Cluster
	onceCreate sync.Once
	onceClose  sync.Once
}{
	nil,
	sync.Once{},
	sync.Once{},
}

// Cache defines the interface for interacting with caching utilities. All derived caches
// must implement this interface
type Cache interface {
	GetBytes(ctx context.Context, key string) ([]byte, error)
	Get(ctx context.Context, key string, target interface{}) error
	SetBytes(ctx context.Context, key string, value []byte) error
	Set(ctx context.Context, key string, value interface{}) error
	Delete(ctx context.Context, key string) error
	Purge(ctx context.Context) error
}

// LocalCache defines a remote-caching approach in which keys are stored remotely in a separate
// process.
type LocalCache struct {
	Cache          *bigcache.BigCache
	Encoder        CacheEncoder
	Metrics        CacheMetrics
	TracingEnabled bool
}

// LocalCacheConfig is the necessary configuration for instantiating a LocalCache struct
type LocalCacheConfig struct {
	Eviction       time.Duration
	TTL            time.Duration
	Shards         uint // Must be power of 2
	TracingEnabled bool
}

// RemoteCache defines a remote-caching approach in which keys are stored remotely in a separate
// process.
type RemoteCache struct {
	cluster        *redisc.Cluster
	Encoder        CacheEncoder
	Metrics        CacheMetrics
	TracingEnabled bool
}

// RemoteCacheConfig is the necessary configuration for instantiating a RemoteCache struct
type RemoteCacheConfig struct {
	URLs           []string
	AuthToken      string
	Timeout        time.Duration
	TracingEnabled bool
}

// TieredCache defines a combined local and remote-caching approach in which keys are stored
// remotely in a separate process as well as cached locally. Local cache is preferred.
type TieredCache struct {
	Remote         Cache
	Local          Cache
	Metrics        CacheMetrics
	TracingEnabled bool
}

// TieredCacheConfig is the necessary configuration for instantiating a TieredCache struct
type TieredCacheConfig struct {
	RemoteConfig   RemoteCacheConfig
	LocalConfig    LocalCacheConfig
	Encoder        CacheEncoder
	TracingEnabled bool
}

// TieredCacheCreator defines an interface to create and return a Tiered Cache
type TieredCacheCreator interface {
	NewCache(
		encoder CacheEncoder,
		metrics CacheMetrics,
		localMetrics CacheMetrics,
		remoteMetrics CacheMetrics,
	) (Cache, error)
}

// createPool creates and returns a Redis connection pool
func (rcc RemoteCacheConfig) createPool(addr string, opts ...redis.DialOption) (*redis.Pool, error) {
	return &redis.Pool{
		MaxIdle:     5,
		MaxActive:   10,
		IdleTimeout: time.Minute,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", addr, opts...)
			if err != nil {
				Logger.Error("Error dialing Redis!", zap.Error(err))
				return nil, err
			}
			return c, nil
		},
	}, nil
}

// NewCache constructs and returns a RemoteCache given configuration
func (rcc RemoteCacheConfig) NewCache(
	encoder CacheEncoder,
	metrics CacheMetrics,
) (RemoteCache, error) {
	sharedCluster.onceCreate.Do(func() {
		sharedCluster.cluster = &redisc.Cluster{
			StartupNodes: rcc.URLs,
			DialOptions:  []redis.DialOption{redis.DialConnectTimeout(rcc.Timeout)},
			CreatePool:   rcc.createPool,
		}
	})
	err := sharedCluster.cluster.Refresh()
	if err != nil {
		Logger.Error("Unable to Refresh Redis Cluster Pool!", zap.Error(err))
	} else {
		Logger.Info("Remote Redis Cache Connection Initialized")
	}
	if err == nil && rcc.AuthToken != "" {
		conn := sharedCluster.cluster.Get()
		defer conn.Close()
		if _, err = conn.Do("AUTH", rcc.AuthToken); err != nil {
			Logger.Error("Unable to authenticate with the Redis Cluster!", zap.Error(err))
		} else {
			Logger.Info("Redis Cluster Successfully Authenticated")
		}
	}
	return RemoteCache{
		cluster:        sharedCluster.cluster,
		Encoder:        encoder,
		Metrics:        metrics,
		TracingEnabled: rcc.TracingEnabled,
	}, err
}

// Close cleans up cache and removes any open connections
func (rc RemoteCache) Close() {
	sharedCluster.onceClose.Do(func() {
		sharedCluster.cluster.Close()
	})
}

// NewCache constructs and returns a LocalCache given configuration
func (lcc LocalCacheConfig) NewCache(
	encoder CacheEncoder,
	metrics CacheMetrics,
) (LocalCache, error) {
	cache := LocalCache{Encoder: encoder, TracingEnabled: lcc.TracingEnabled}
	if lcc.Shards != 0 && lcc.Shards%2 != 0 {
		err := fmt.Errorf("shards must be power of 2 - %v is invalid", lcc.Shards)
		Logger.Error("Shards must be power of 2", zap.Uint("shards", lcc.Shards), zap.Error(err))
		return cache, err
	}
	config := bigcache.DefaultConfig(lcc.Eviction)
	if lcc.TTL != 0 {
		config.LifeWindow = lcc.TTL
	}
	if lcc.Shards != 0 {
		config.Shards = int(lcc.Shards)
	}
	var err error
	cache.Cache, err = bigcache.NewBigCache(config)
	if err != nil {
		Logger.Error("Unable to allocate Local In-Memory Cache", zap.Error(err))
	} else {
		Logger.Info("Local In-Memory Cache Initialized")
	}
	if metrics != nil {
		cache.Metrics = metrics
	}
	return cache, err
}

// NewCache constructs and returns a TieredCache given configuration
func (tcc TieredCacheConfig) NewCache(
	encoder CacheEncoder,
	metrics CacheMetrics,
	localMetrics CacheMetrics,
	remoteMetrics CacheMetrics,
) (Cache, error) {
	remote, err := tcc.RemoteConfig.NewCache(encoder, remoteMetrics)
	if err != nil {
		return TieredCache{}, err
	}
	local, err := tcc.LocalConfig.NewCache(encoder, localMetrics)
	if err != nil {
		return TieredCache{}, err
	}
	return TieredCache{
		Remote:         remote,
		Local:          local,
		Metrics:        metrics,
		TracingEnabled: tcc.TracingEnabled,
	}, nil
}

// Close cleans up cache and removes any open connections
func (tc TieredCache) Close() {
	tc.Remote.(RemoteCache).Close()
}

// GetBytes gets the requested bytes from remote cache
func (rc RemoteCache) GetBytes(ctx context.Context, key string) ([]byte, error) {
	var span opentracing.Span
	if rc.TracingEnabled {
		span, _ = opentracing.StartSpanFromContext(ctx, "remote-cache-get-bytes")
		span.SetTag("command", "GET")
		span.SetTag("key", key)
	}
	conn := rc.cluster.Get()
	defer conn.Close()
	data, err := redis.Bytes(conn.Do("GET", key))
	if err != nil {
		Logger.Debug("Redis does not contain key", zap.String("key", key))
	}
	if rc.TracingEnabled {
		if err != nil {
			span.SetTag("result", "miss")
		} else {
			span.SetTag("result", "hit")
		}
		span.Finish()
	}
	return data, err
}

// Get retrieves the value from cache, decodes it, and sets the result in target. target must be a
// pointer.
func (rc RemoteCache) Get(ctx context.Context, key string, target interface{}) error {
	data, err := rc.GetBytes(ctx, key)
	if rc.Metrics != nil {
		if err != nil {
			rc.Metrics.Miss()
		} else {
			rc.Metrics.Hit()
		}
	}
	if err != nil {
		return err
	}
	return rc.Encoder.Decode(data, target)
}

// SetBytes sets the provided bytes in the remote cache on the provided key
func (rc RemoteCache) SetBytes(ctx context.Context, key string, value []byte) error {
	var span opentracing.Span
	if rc.TracingEnabled {
		span, _ = opentracing.StartSpanFromContext(ctx, "remote-cache-set-bytes")
		span.SetTag("command", "SET")
		span.SetTag("key", key)
	}
	conn := rc.cluster.Get()
	defer conn.Close()
	_, err := conn.Do("SET", key, value)
	if err != nil {
		Logger.Debug("Unable to set key on Redis", zap.String("key", key))
	}
	if rc.TracingEnabled {
		if err != nil {
			span.SetTag("result", "fail")
		} else {
			span.SetTag("result", "set")
		}
		span.Finish()
	}
	return err
}

// Set encodes the provided value and sets it in the remote cache
func (rc RemoteCache) Set(ctx context.Context, key string, value interface{}) error {
	encodedData, err := rc.Encoder.Encode(value)
	if rc.Metrics != nil {
		if err != nil {
			rc.Metrics.SetCollision()
		} else {
			rc.Metrics.Set()
		}
	}
	if err != nil {
		return err
	}
	return rc.SetBytes(ctx, key, encodedData)
}

// Delete removes the value from remote cache. Because Redis doesnt support Fuzzy matches for
// delete, this function first gets all matching keys, and then proceeds to pipeline deletion of
// those keys
func (rc RemoteCache) Delete(ctx context.Context, key string) error {
	var span opentracing.Span
	if rc.TracingEnabled {
		span, _ = opentracing.StartSpanFromContext(ctx, "remote-cache-delete")
		span.SetTag("command", "Pipeline:KEYS:MULTI:DEL:EXEC")
		span.SetTag("key", key)
	}
	conn := rc.cluster.Get()
	defer conn.Close()
	keysToDelete, err := redis.Strings(conn.Do("KEYS", key))
	if err != nil {
		Logger.Error("Unable to retrieve Keys to delete from Redis", zap.String("key", key))
	}

	// Execute a Redis Pipeline which bulk delets all matching keys
	err = conn.Send("MULTI")
	if err != nil {
		Logger.Error("Failed to start Redis deletion transaction", zap.String("key", key))
	}
	if rc.TracingEnabled {
		span.SetTag("num_keys", len(keysToDelete))
	}
	if rc.Metrics != nil {
		if err != nil || len(keysToDelete) <= 0 {
			rc.Metrics.DeleteMiss()
		} else {
			rc.Metrics.DeleteHit()
		}
	}
	for _, keyToDelete := range keysToDelete {
		err = conn.Send("DEL", keyToDelete)
		if err != nil {
			Logger.Debug("Failed to send Deletion command for key", zap.String("key", key))
		}
	}
	_, err = conn.Do("EXEC")
	if err != nil {
		Logger.Error("Unable to delete keys on Redis via pipeline", zap.String("key", key))
	}
	if rc.TracingEnabled {
		if err != nil {
			span.SetTag("result", "fail")
		} else {
			span.SetTag("result", "delete")
		}
		span.Finish()
	}
	return err
}

// Purge wipes out all items under control of this cache in Redis
func (rc RemoteCache) Purge(ctx context.Context) error {
	var span opentracing.Span
	if rc.TracingEnabled {
		span, _ = opentracing.StartSpanFromContext(ctx, "remote-cache-delete")
		span.SetTag("command", "FLUSHALL")
	}
	conn := rc.cluster.Get()
	defer conn.Close()
	_, err := conn.Do("FLUSHALL")
	if err != nil {
		Logger.Debug("Unable to purge keys in Redis")
	}
	if rc.Metrics != nil {
		if err != nil {
			rc.Metrics.PurgeMiss()
		} else {
			rc.Metrics.PurgeHit()
		}
	}
	if rc.TracingEnabled {
		if err != nil {
			span.SetTag("result", "fail")
		} else {
			span.SetTag("result", "purge")
		}
		span.Finish()
	}
	return err
}

// GetBytes gets the requested bytes from local cache
func (lc LocalCache) GetBytes(ctx context.Context, key string) ([]byte, error) {
	return lc.Cache.Get(key)
}

// Get retrieves the value from cache, decodes it, and sets the result in target. target must be a
// pointer.
func (lc LocalCache) Get(ctx context.Context, key string, target interface{}) error {
	data, err := lc.GetBytes(ctx, key)
	if lc.Metrics != nil {
		if err != nil {
			lc.Metrics.Miss()
		} else {
			lc.Metrics.Hit()
		}
	}
	if err != nil {
		return err
	}
	return lc.Encoder.Decode(data, target)
}

// SetBytes sets the provided bytes in the local cache on the provided key
func (lc LocalCache) SetBytes(ctx context.Context, key string, value []byte) error {
	return lc.Cache.Set(key, value)
}

// Set encodes the provided value and sets it in the local cache
func (lc LocalCache) Set(ctx context.Context, key string, value interface{}) error {
	encodedData, err := lc.Encoder.Encode(value)
	if lc.Metrics != nil {
		if err != nil {
			lc.Metrics.SetCollision()
		} else {
			lc.Metrics.Set()
		}
	}
	if err != nil {
		return err
	}
	return lc.SetBytes(ctx, key, encodedData)
}

// Delete removes the value from local cache
func (lc LocalCache) Delete(ctx context.Context, key string) error {
	err := lc.Cache.Delete(key)
	if lc.Metrics != nil {
		if err != nil {
			lc.Metrics.DeleteMiss()
		} else {
			lc.Metrics.DeleteHit()
		}
	}
	return err
}

// Purge wipes out all items in local cache
func (lc LocalCache) Purge(ctx context.Context) error {
	err := lc.Cache.Reset()
	if lc.Metrics != nil {
		if err != nil {
			lc.Metrics.PurgeMiss()
		} else {
			lc.Metrics.PurgeHit()
		}
	}
	return err
}

// GetBytes gets the requested bytes from from tiered cache. Local first, then remote.
func (tc TieredCache) GetBytes(ctx context.Context, key string) ([]byte, error) {
	data, err := tc.Local.GetBytes(ctx, key)
	if err != nil {
		data, err = tc.Remote.GetBytes(ctx, key)
	}
	return data, err
}

// Get retrieves the value from the tiered cache, cache, decodes it, and sets the result in target.
// Local cache first, then remote. target must be a pointer.
func (tc TieredCache) Get(ctx context.Context, key string, target interface{}) error {
	err := tc.Local.Get(ctx, key, target)
	if err != nil {
		err = tc.Remote.Get(ctx, key, target)
	}
	if tc.Metrics != nil {
		if err != nil {
			tc.Metrics.Miss()
		} else {
			tc.Metrics.Hit()
		}
	}
	return err
}

// SetBytes sets the provided bytes in the local and remote caches on the provided key
func (tc TieredCache) SetBytes(ctx context.Context, key string, value []byte) error {
	err := tc.Local.SetBytes(ctx, key, value)
	if err == nil {
		err = tc.Remote.SetBytes(ctx, key, value)
	}
	return err
}

// Set encodes the provided value and sets it in the local and remote cache
func (tc TieredCache) Set(ctx context.Context, key string, value interface{}) error {
	err := tc.Local.Set(ctx, key, value)
	if err == nil {
		err = tc.Remote.Set(ctx, key, value)
	}
	if tc.Metrics != nil {
		if err != nil {
			tc.Metrics.SetCollision()
		} else {
			tc.Metrics.Set()
		}
	}
	return err
}

// Delete removes the value from local cache and remote cache
func (tc TieredCache) Delete(ctx context.Context, key string) error {
	err := tc.Local.Delete(ctx, key)
	if err == nil {
		err = tc.Remote.Delete(ctx, key)
	}
	if tc.Metrics != nil {
		if err != nil {
			tc.Metrics.DeleteMiss()
		} else {
			tc.Metrics.DeleteHit()
		}
	}
	return err
}

// Purge wipes out all items locally, and all items under control of this cache in Redis
func (tc TieredCache) Purge(ctx context.Context) error {
	err := tc.Local.Purge(ctx)
	if err == nil {
		err = tc.Remote.Purge(ctx)
	}
	if tc.Metrics != nil {
		if err != nil {
			tc.Metrics.PurgeMiss()
		} else {
			tc.Metrics.PurgeHit()
		}
	}
	return err
}

// CacheEncoder defines an interface for encoding and decoding
// values stored in cache
type CacheEncoder interface {
	// value must be a pointer
	Encode(value interface{}) ([]byte, error)
	// target must be a pointer
	Decode(cachedValue []byte, target interface{}) error
}

// GobCacheEncoder uses encoding/gob to encode values for caching
type GobCacheEncoder struct{}

// Encode encodes the provided value using gob. value must be a pointer.
func (gb *GobCacheEncoder) Encode(value interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(value)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decode decodes the cached value using the gob encoding and sets the
// result in target. target must be a pointer
func (gb *GobCacheEncoder) Decode(cachedValue []byte, target interface{}) error {
	reader := bytes.NewReader(cachedValue)
	dec := gob.NewDecoder(reader)
	return dec.Decode(target)
}
