package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"sync"
	"time"

	"github.com/allegro/bigcache"
	"github.com/gomodule/redigo/redis"
	"github.com/mna/redisc"
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
	GetBytes(key string) ([]byte, error)
	Get(key string, target interface{}) error
	SetBytes(key string, value []byte) error
	Set(key string, value interface{}) error
	Delete(key string) error
	Purge() error
}

// LocalCache defines a remote-caching approach in which keys are stored remotely in a separate
// process.
type LocalCache struct {
	Cache   *bigcache.BigCache
	Encoder CacheEncoder
	Metrics CacheMetrics
}

// LocalCacheConfig is the necessary configuration for instantiating a LocalCache struct
type LocalCacheConfig struct {
	Eviction time.Duration
	TTL      time.Duration
	Shards   uint // Must be power of 2
}

// RemoteCache defines a remote-caching approach in which keys are stored remotely in a separate
// process. RemoteCache utilizes Redis hash to group items together.
type RemoteCache struct {
	cluster *redisc.Cluster
	Encoder CacheEncoder
	HashKey string
	Metrics CacheMetrics
}

// RemoteCacheConfig is the necessary configuration for instantiating a RemoteCache struct
type RemoteCacheConfig struct {
	URLs      []string
	AuthToken string
	Timeout   time.Duration
}

// TieredCache defines a combined local and remote-caching approach in which keys are stored
// remotely in a separate process as well as cached locally. Local cache is preferred.
type TieredCache struct {
	Remote  Cache
	Local   Cache
	Metrics CacheMetrics
}

// TieredCacheConfig is the necessary configuration for instantiating a TieredCache struct
type TieredCacheConfig struct {
	RemoteConfig RemoteCacheConfig
	LocalConfig  LocalCacheConfig
	Encoder      CacheEncoder
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
func (rcc RemoteCacheConfig) NewCache(encoder CacheEncoder, hashKey string, metrics CacheMetrics) (RemoteCache, error) {
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
		cluster: sharedCluster.cluster,
		Encoder: encoder,
		HashKey: hashKey,
		Metrics: metrics,
	}, err
}

// Close cleans up cache and removes any open connections
func (rc RemoteCache) Close() {
	sharedCluster.onceClose.Do(func() {
		sharedCluster.cluster.Close()
	})
}

// NewCache constructs and returns a LocalCache given configuration
func (lcc LocalCacheConfig) NewCache(encoder CacheEncoder, metrics CacheMetrics) (LocalCache, error) {
	cache := LocalCache{Encoder: encoder}
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
	remoteHashKey string,
	metrics CacheMetrics,
	localMetrics CacheMetrics,
	remoteMetrics CacheMetrics,
) (TieredCache, error) {
	remote, err := tcc.RemoteConfig.NewCache(encoder, remoteHashKey, remoteMetrics)
	if err != nil {
		return TieredCache{}, err
	}
	local, err := tcc.LocalConfig.NewCache(encoder, localMetrics)
	if err != nil {
		return TieredCache{}, err
	}
	return TieredCache{Remote: remote, Local: local, Metrics: metrics}, nil
}

// Close cleans up cache and removes any open connections
func (tc TieredCache) Close() {
	tc.Remote.(RemoteCache).Close()
}

// GetBytes gets the requested bytes from remote cache
func (rc RemoteCache) GetBytes(key string) ([]byte, error) {
	conn := rc.cluster.Get()
	defer conn.Close()
	data, err := redis.Bytes(conn.Do("HGET", rc.HashKey, key))
	if err != nil {
		Logger.Debug("Redis does not contain key", zap.String("key", key))
	}
	return data, err
}

// Get retrieves the value from cache, decodes it, and sets the result in target. target must be a
// pointer.
func (rc RemoteCache) Get(key string, target interface{}) error {
	data, err := rc.GetBytes(key)
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
func (rc RemoteCache) SetBytes(key string, value []byte) error {
	conn := rc.cluster.Get()
	defer conn.Close()
	_, err := conn.Do("HSET", rc.HashKey, key, value)
	if err != nil {
		Logger.Debug("Unable to set key on Redis", zap.String("key", key), zap.String("hash_key", rc.HashKey))
	}
	return err
}

// Set encodes the provided value and sets it in the remote cache
func (rc RemoteCache) Set(key string, value interface{}) error {
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
	return rc.SetBytes(key, encodedData)
}

// Delete removes the value from remote cache
func (rc RemoteCache) Delete(key string) error {
	conn := rc.cluster.Get()
	defer conn.Close()
	_, err := conn.Do("HDEL", rc.HashKey, key)
	if err != nil {
		Logger.Debug("Unable to delete key on Redis", zap.String("key", key), zap.String("hash_key", rc.HashKey))
	}
	if rc.Metrics != nil {
		if err != nil {
			rc.Metrics.DeleteMiss()
		} else {
			rc.Metrics.DeleteHit()
		}
	}
	return err
}

// Purge wipes out all items under control of this cache in Redis
func (rc RemoteCache) Purge() error {
	conn := rc.cluster.Get()
	defer conn.Close()
	_, err := conn.Do("DEL", rc.HashKey)
	if err != nil {
		Logger.Debug("Unable to purge keys in Redis", zap.String("hash_key", rc.HashKey))
	}
	if rc.Metrics != nil {
		if err != nil {
			rc.Metrics.PurgeMiss()
		} else {
			rc.Metrics.PurgeHit()
		}
	}
	return err
}

// GetBytes gets the requested bytes from local cache
func (lc LocalCache) GetBytes(key string) ([]byte, error) {
	return lc.Cache.Get(key)
}

// Get retrieves the value from cache, decodes it, and sets the result in target. target must be a
// pointer.
func (lc LocalCache) Get(key string, target interface{}) error {
	data, err := lc.GetBytes(key)
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
func (lc LocalCache) SetBytes(key string, value []byte) error {
	return lc.Cache.Set(key, value)
}

// Set encodes the provided value and sets it in the local cache
func (lc LocalCache) Set(key string, value interface{}) error {
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
	return lc.SetBytes(key, encodedData)
}

// Delete removes the value from local cache
func (lc LocalCache) Delete(key string) error {
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
func (lc LocalCache) Purge() error {
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
func (tc TieredCache) GetBytes(key string) ([]byte, error) {
	data, err := tc.Local.GetBytes(key)
	if err != nil {
		data, err = tc.Remote.GetBytes(key)
	}
	return data, err
}

// Get retrieves the value from the tiered cache, cache, decodes it, and sets the result in target.
// Local cache first, then remote. target must be a pointer.
func (tc TieredCache) Get(key string, target interface{}) error {
	err := tc.Local.Get(key, target)
	if err != nil {
		err = tc.Remote.Get(key, target)
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
func (tc TieredCache) SetBytes(key string, value []byte) error {
	err := tc.Local.SetBytes(key, value)
	if err == nil {
		err = tc.Remote.SetBytes(key, value)
	}
	return err
}

// Set encodes the provided value and sets it in the local and remote cache
func (tc TieredCache) Set(key string, value interface{}) error {
	err := tc.Local.Set(key, value)
	if err == nil {
		err = tc.Remote.Set(key, value)
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
func (tc TieredCache) Delete(key string) error {
	err := tc.Local.Delete(key)
	if err == nil {
		err = tc.Remote.Delete(key)
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
func (tc TieredCache) Purge() error {
	err := tc.Local.Purge()
	if err == nil {
		err = tc.Remote.Purge()
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
