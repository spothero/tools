package core

import (
	"context"
	"fmt"

	"github.com/stretchr/testify/mock"
)

// MockCache mocks the Cache implementation for use in test caches
type MockCache struct {
	Cache   map[string][]byte
	Encoder CacheEncoder
	Metrics CacheMetrics
}

// MockCacheMetrics provides a mock cache metrics implementation
type MockCacheMetrics struct {
	mock.Mock
}

// NewMockCache constructs a new cache for testing
func NewMockCache(encoder CacheEncoder) *MockCache {
	return &MockCache{
		Cache:   make(map[string][]byte),
		Encoder: encoder,
		Metrics: &MockCacheMetrics{},
	}
}

// GetBytes is a mock GetBytes implementation for cache
func (mc *MockCache) GetBytes(ctx context.Context, key string) ([]byte, error) {
	value, ok := mc.Cache[key]
	if !ok {
		return nil, fmt.Errorf("key not found")
	}
	return value, nil
}

// Get is a mock GetBytes implementation for cache
func (mc *MockCache) Get(ctx context.Context, key string, target interface{}) error {
	data, err := mc.GetBytes(ctx, key)
	if err != nil {
		return err
	}
	return mc.Encoder.Decode(data, target)
}

// SetBytes is a mock SetBytes implementation for cache
func (mc *MockCache) SetBytes(ctx context.Context, key string, value []byte) error {
	mc.Cache[key] = value
	return nil
}

// Set is a mock Set implementation for cache
func (mc *MockCache) Set(ctx context.Context, key string, value interface{}) error {
	cacheBytes, err := mc.Encoder.Encode(value)
	if err != nil {
		return err
	}
	return mc.SetBytes(ctx, key, cacheBytes)
}

// Delete is a mock Delete implementation for cache
func (mc *MockCache) Delete(ctx context.Context, key string) error {
	if _, ok := mc.Cache[key]; !ok {
		return fmt.Errorf("key not found for deletion")
	}
	delete(mc.Cache, key)
	return nil
}

// Purge is a mock Purge implementation for cache
func (mc *MockCache) Purge(ctx context.Context) error {
	mc.Cache = make(map[string][]byte)
	return nil
}

// MockCacheEncoder is a fake encoder for use in tests
type MockCacheEncoder struct{}

// Encode mock simply returns the value it was given
func (mce MockCacheEncoder) Encode(value interface{}) ([]byte, error) {
	return value.([]byte), nil
}

// Decode mock returns no error
func (mce MockCacheEncoder) Decode(cachedValue []byte, target interface{}) error {
	return nil
}

// Hit is a mock metrics Hit implementation
func (mcc *MockCacheMetrics) Hit() {
	mcc.Called()
}

// Miss is a mock metrics Miss implementation
func (mcc *MockCacheMetrics) Miss() {
	mcc.Called()
}

// Set is a mock metrics Set implementation
func (mcc *MockCacheMetrics) Set() {
	mcc.Called()
}

// SetCollision is a mock metrics SetCollision implementation
func (mcc *MockCacheMetrics) SetCollision() {
	mcc.Called()
}

// DeleteHit is a mock metrics DeleteHit implementation
func (mcc *MockCacheMetrics) DeleteHit() {
	mcc.Called()
}

// DeleteMiss is a mock metrics DeleteMiss implementation
func (mcc *MockCacheMetrics) DeleteMiss() {
	mcc.Called()
}

// PurgeHit is a mock metrics PurgeHit implementation
func (mcc *MockCacheMetrics) PurgeHit() {
	mcc.Called()
}

// PurgeMiss is a mock metrics PurgeMiss implementation
func (mcc *MockCacheMetrics) PurgeMiss() {
	mcc.Called()
}

// MockTieredCacheCreator provides a mock tiered cache config implementation
type MockTieredCacheCreator struct {
	mock.Mock
}

// NewCache returns a mocked tiered cache
func (m *MockTieredCacheCreator) NewCache(
	encoder CacheEncoder,
	metrics CacheMetrics,
	localMetrics CacheMetrics,
	remoteMetrics CacheMetrics,
) (Cache, error) {
	args := m.Called(encoder, metrics, localMetrics, remoteMetrics)
	return args.Get(0).(Cache), args.Error(1)
}
