package core

import (
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis"
	"github.com/mna/redisc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockCacheEncoder mocks the cache encoder for use in tests
type MockedCacheEncoder struct {
	mock.Mock
}

// Encode mocks the cache encode implementation
func (mce *MockedCacheEncoder) Encode(value interface{}) ([]byte, error) {
	args := mce.Called(value)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

// Decode mocks the cache decode implementation
func (mce *MockedCacheEncoder) Decode(cachedValue []byte, target interface{}) error {
	args := mce.Called(cachedValue, target)
	return args.Error(0)
}

// Remote Cache Testing

func TestRemoteSetBytes(t *testing.T) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()
	s.HSet("hash", "test-key", "test-value")
	mockCluster := &redisc.Cluster{StartupNodes: []string{s.Addr()}}

	rc := RemoteCache{cluster: mockCluster, HashKey: "hash"}
	err = rc.SetBytes("test-key", []byte("test-value"))
	assert.NoError(t, err)
}

func TestRemoteSet(t *testing.T) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()
	s.HSet("hash", "test-key", "test-value")
	mockCluster := &redisc.Cluster{StartupNodes: []string{s.Addr()}}

	encoder := &MockedCacheEncoder{}
	value := "don't care"
	encoder.On("Encode", value).Return([]byte("test-value"), nil)
	mcm := &MockCacheMetrics{}
	mcm.On("Set")
	rc := RemoteCache{cluster: mockCluster, Encoder: encoder, HashKey: "hash", Metrics: mcm}
	err = rc.Set("test-key", value)
	assert.NoError(t, err)
	mcm.AssertCalled(t, "Set")
}

func TestRemoteSetError(t *testing.T) {
	encoder := &MockedCacheEncoder{}
	value := "don't care"
	encoder.On("Encode", value).Return(nil, fmt.Errorf("error"))
	mcm := &MockCacheMetrics{}
	mcm.On("SetCollision")
	rc := RemoteCache{cluster: &redisc.Cluster{}, Encoder: encoder, Metrics: mcm}
	err := rc.Set("test-key", value)
	assert.Error(t, err)
	mcm.AssertCalled(t, "SetCollision")
}

func TestRemoteSetBytesError(t *testing.T) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()
	// Use a normal Set for the hashed key to force a conflict which in turn causes a SET failure
	s.Set("hash", "test-value")
	mockCluster := &redisc.Cluster{StartupNodes: []string{s.Addr()}}

	rc := RemoteCache{cluster: mockCluster, HashKey: "hash"}
	err = rc.SetBytes("test-key", []byte("test-value"))
	assert.Error(t, err)
}

func TestRemoteGetBytes(t *testing.T) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()
	s.HSet("hash", "test-key", "test-value")
	mockCluster := &redisc.Cluster{StartupNodes: []string{s.Addr()}}

	rc := RemoteCache{cluster: mockCluster, HashKey: "hash"}
	value, err := rc.GetBytes("test-key")
	assert.NoError(t, err)
	assert.Equal(t, value, []byte("test-value"))
}

func TestRemoteGet(t *testing.T) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()
	s.HSet("hash", "test-key", "test-value")
	mockCluster := &redisc.Cluster{StartupNodes: []string{s.Addr()}}

	encoder := &MockedCacheEncoder{}
	target := struct{}{}
	encoder.On("Decode", []byte("test-value"), target).Return(nil)
	mcm := &MockCacheMetrics{}
	mcm.On("Hit")
	rc := RemoteCache{cluster: mockCluster, Encoder: encoder, HashKey: "hash", Metrics: mcm}
	err = rc.Get("test-key", target)
	assert.NoError(t, err)
	mcm.AssertCalled(t, "Hit")
}

func TestRemoteGetError(t *testing.T) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()
	// Use a normal Set for the hashed key to force a conflict which in turn causes a SET failure
	s.Set("hash", "test-value")
	mockCluster := &redisc.Cluster{StartupNodes: []string{s.Addr()}}

	target := struct{}{}
	mcm := &MockCacheMetrics{}
	mcm.On("Miss")
	rc := RemoteCache{cluster: mockCluster, HashKey: "hash", Metrics: mcm}
	err = rc.Get("test-key", target)
	assert.Error(t, err)
	mcm.AssertCalled(t, "Miss")
}

func TestRemoteGetBytesError(t *testing.T) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()
	// Use a normal Set for the hashed key to force a conflict which in turn causes a SET failure
	s.Set("hash", "test-value")
	mockCluster := &redisc.Cluster{StartupNodes: []string{s.Addr()}}

	rc := RemoteCache{cluster: mockCluster, HashKey: "hash"}
	value, err := rc.GetBytes("test-key")
	assert.Error(t, err)
	assert.Nil(t, value)
}

func TestRemoteDelete(t *testing.T) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()
	s.HSet("hash", "test-key", "test-value")
	mockCluster := &redisc.Cluster{StartupNodes: []string{s.Addr()}}

	mcm := &MockCacheMetrics{}
	mcm.On("DeleteHit")
	rc := RemoteCache{cluster: mockCluster, HashKey: "hash", Metrics: mcm}
	err = rc.Delete("test-key")
	assert.NoError(t, err)
	mcm.AssertCalled(t, "DeleteHit")
}

func TestRemoteDeleteError(t *testing.T) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()
	s.Set("hash", "test-key")
	mockCluster := &redisc.Cluster{StartupNodes: []string{s.Addr()}}

	mcm := &MockCacheMetrics{}
	mcm.On("DeleteMiss")
	rc := RemoteCache{cluster: mockCluster, HashKey: "hash", Metrics: mcm}
	err = rc.Delete("test-key")
	assert.Error(t, err)
	mcm.AssertCalled(t, "DeleteMiss")
}

func TestRemotePurge(t *testing.T) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()
	mockCluster := &redisc.Cluster{StartupNodes: []string{s.Addr()}}

	mcm := &MockCacheMetrics{}
	mcm.On("PurgeHit")
	rc := RemoteCache{cluster: mockCluster, HashKey: "hash", Metrics: mcm}
	err = rc.Purge()
	assert.NoError(t, err)
	mcm.AssertCalled(t, "PurgeHit")
}

func TestRemotePurgeError(t *testing.T) {
	// Here we are just lazy -- the only way to get DEL to fail is to break the connection.
	// Given that, do not create a miniredis to connect to.
	mockCluster := &redisc.Cluster{StartupNodes: []string{""}}

	mcm := &MockCacheMetrics{}
	mcm.On("PurgeMiss")
	rc := RemoteCache{cluster: mockCluster, HashKey: "hash", Metrics: mcm}
	err := rc.Purge()
	assert.Error(t, err)
	mcm.AssertCalled(t, "PurgeMiss")
}

// Local Cache Testing

func newLocalCache(t *testing.T, ttl time.Duration, eviction time.Duration) LocalCache {
	if ttl == 0 {
		ttl = time.Duration(time.Second)
	}
	if eviction == 0 {
		eviction = time.Duration(time.Second)
	}
	lcc := LocalCacheConfig{TTL: ttl, Eviction: eviction}
	cache, err := lcc.NewCache(&MockedCacheEncoder{}, nil)
	cache.Metrics = &MockCacheMetrics{}
	assert.Nil(t, err)
	return cache
}

func TestLocalInvalidShards(t *testing.T) {
	lcc := LocalCacheConfig{TTL: time.Duration(time.Second * 1), Shards: 3}
	_, err := lcc.NewCache(&MockedCacheEncoder{}, nil)
	assert.NotNil(t, err)
}

func TestLocalSetBytes(t *testing.T) {
	lc := newLocalCache(t, 0, 0)
	err := lc.SetBytes("test-key", []byte("test-value"))
	assert.Nil(t, err)
}

func TestLocalSet(t *testing.T) {
	lc := newLocalCache(t, 0, 0)
	lc.Metrics.(*MockCacheMetrics).On("Set")
	value := "don't care"
	lc.Encoder.(*MockedCacheEncoder).On("Encode", value).Return([]byte("test-value"), nil)
	err := lc.Set("test-key", value)
	assert.Nil(t, err)
	lc.Metrics.(*MockCacheMetrics).AssertCalled(t, "Set")
}

func TestLocalSetError(t *testing.T) {
	lc := newLocalCache(t, 0, 0)
	lc.Metrics.(*MockCacheMetrics).On("SetCollision")
	value := "don't care"
	lc.Encoder.(*MockedCacheEncoder).On("Encode", value).Return(nil, fmt.Errorf("error"))
	err := lc.Set("test-key", value)
	assert.Error(t, err)
	lc.Metrics.(*MockCacheMetrics).AssertCalled(t, "SetCollision")
}

func TestLocalGetBytes(t *testing.T) {
	lc := newLocalCache(t, 0, 0)

	// Use underlying cache to avoid testing two functions in one test
	err := lc.Cache.Set("test-key", []byte("test-value"))
	require.Nil(t, err)
	value, err := lc.GetBytes("test-key")
	assert.Nil(t, err)
	assert.Equal(t, value, []byte("test-value"))
}

func TestLocalGet(t *testing.T) {
	lc := newLocalCache(t, 0, 0)
	lc.Metrics.(*MockCacheMetrics).On("Hit")

	// Use underlying cache to avoid testing two functions in one test
	err := lc.Cache.Set("test-key", []byte("test-value"))
	require.Nil(t, err)
	target := struct{}{}
	lc.Encoder.(*MockedCacheEncoder).On("Decode", []byte("test-value"), target).Return(nil)
	err = lc.Get("test-key", target)
	assert.Nil(t, err)
	lc.Metrics.(*MockCacheMetrics).AssertCalled(t, "Hit")
}

func TestLocalGetError(t *testing.T) {
	lc := newLocalCache(t, 0, 0)
	lc.Metrics.(*MockCacheMetrics).On("Miss")

	target := struct{}{}
	lc.Encoder.(*MockedCacheEncoder).On("Decode", []byte("test-value"), target).Return(fmt.Errorf("error"))
	err := lc.Get("test-key", target)
	assert.Error(t, err)
	lc.Metrics.(*MockCacheMetrics).AssertCalled(t, "Miss")
}

func TestLocalGetBytesError(t *testing.T) {
	lc := newLocalCache(t, 0, 0)
	value, err := lc.GetBytes("test-key")
	assert.Error(t, err)
	assert.Nil(t, value)
}

func TestLocalDelete(t *testing.T) {
	lc := newLocalCache(t, 0, 0)
	lc.Metrics.(*MockCacheMetrics).On("DeleteHit")

	// Use underlying cache to avoid testing two functions in one test
	err := lc.Cache.Set("test-key", []byte("test-value"))
	assert.Nil(t, err)
	err = lc.Delete("test-key")
	assert.Nil(t, err)
	lc.Metrics.(*MockCacheMetrics).AssertCalled(t, "DeleteHit")
}

func TestLocalDeleteError(t *testing.T) {
	lc := newLocalCache(t, 0, 0)
	lc.Metrics.(*MockCacheMetrics).On("DeleteMiss")
	err := lc.Delete("test-key")
	assert.Error(t, err)
	lc.Metrics.(*MockCacheMetrics).AssertCalled(t, "DeleteMiss")
}

func TestLocalPurge(t *testing.T) {
	lc := newLocalCache(t, 0, 0)
	lc.Metrics.(*MockCacheMetrics).On("PurgeHit")
	err := lc.Purge()
	assert.Nil(t, err)
	lc.Metrics.(*MockCacheMetrics).AssertCalled(t, "PurgeHit")
}

// Tiered Cache Testing

func TestTieredSetBytes(t *testing.T) {
	mtc := TieredCache{
		Local:  NewMockCache(nil),
		Remote: NewMockCache(nil),
	}
	err := mtc.SetBytes("test-key", []byte("test-value"))
	assert.Nil(t, err)
	localValue, ok := mtc.Local.(*MockCache).Cache["test-key"]
	assert.True(t, ok)
	assert.Equal(t, "test-value", string(localValue))
	remoteValue, ok := mtc.Remote.(*MockCache).Cache["test-key"]
	assert.True(t, ok)
	assert.Equal(t, "test-value", string(remoteValue))
}

func TestTieredSet(t *testing.T) {
	encoder := &MockedCacheEncoder{}
	value := "don't care"
	encoder.On("Encode", value).Return([]byte("test-value"), nil)
	mcm := &MockCacheMetrics{}
	mcm.On("Set")
	mtc := TieredCache{
		Local:   NewMockCache(encoder),
		Remote:  NewMockCache(encoder),
		Metrics: mcm,
	}
	err := mtc.Set("test-key", value)
	assert.Nil(t, err)
	localValue, ok := mtc.Local.(*MockCache).Cache["test-key"]
	assert.True(t, ok)
	assert.Equal(t, "test-value", string(localValue))
	remoteValue, ok := mtc.Remote.(*MockCache).Cache["test-key"]
	assert.True(t, ok)
	assert.Equal(t, "test-value", string(remoteValue))
	mcm.AssertCalled(t, "Set")
}

func TestTieredGetBytesLocal(t *testing.T) {
	mtc := TieredCache{
		Local:  NewMockCache(nil),
		Remote: NewMockCache(nil),
	}
	mtc.Local.(*MockCache).Cache["test-key"] = []byte("test-value")
	value, err := mtc.GetBytes("test-key")
	assert.Nil(t, err)
	assert.Equal(t, "test-value", string(value))
}

func TestTieredGetLocal(t *testing.T) {
	encoder := &MockedCacheEncoder{}
	target := struct{}{}
	encoder.On("Decode", []byte("test-value"), target).Return(nil)
	mcm := &MockCacheMetrics{}
	mcm.On("Hit")
	mtc := TieredCache{
		Local:   NewMockCache(encoder),
		Remote:  NewMockCache(encoder),
		Metrics: mcm,
	}
	mtc.Local.(*MockCache).Cache["test-key"] = []byte("test-value")
	err := mtc.Get("test-key", target)
	assert.Nil(t, err)
	mcm.AssertCalled(t, "Hit")
}

func TestTieredGetBytesRemote(t *testing.T) {
	mtc := TieredCache{
		Local:  NewMockCache(nil),
		Remote: NewMockCache(nil),
	}
	mtc.Remote.(*MockCache).Cache["test-key"] = []byte("test-value")
	value, err := mtc.GetBytes("test-key")
	assert.Nil(t, err)
	assert.Equal(t, "test-value", string(value))
}

func TestTieredGetRemote(t *testing.T) {
	encoder := &MockedCacheEncoder{}
	target := struct{}{}
	encoder.On("Decode", []byte("test-value"), target).Return(nil)
	mcm := &MockCacheMetrics{}
	mcm.On("Hit")
	mtc := TieredCache{
		Local:   NewMockCache(encoder),
		Remote:  NewMockCache(encoder),
		Metrics: mcm,
	}
	mtc.Remote.(*MockCache).Cache["test-key"] = []byte("test-value")
	err := mtc.Get("test-key", target)
	assert.Nil(t, err)
	mcm.AssertCalled(t, "Hit")
}

// TestTieredGetError tests a fall-through on both local and remote
func TestTieredGetError(t *testing.T) {
	encoder := &MockedCacheEncoder{}
	target := struct{}{}
	encoder.On("Decode", []byte("test-value"), target).Return(nil)
	mcm := &MockCacheMetrics{}
	mcm.On("Miss")
	mtc := TieredCache{
		Local:   NewMockCache(encoder),
		Remote:  NewMockCache(encoder),
		Metrics: mcm,
	}
	err := mtc.Get("test-key", target)
	assert.Error(t, err)
	mcm.AssertCalled(t, "Miss")
}

func TestTieredGetBytesError(t *testing.T) {
	mtc := TieredCache{
		Local:  NewMockCache(nil),
		Remote: NewMockCache(nil),
	}
	value, err := mtc.GetBytes("test-key")
	assert.Error(t, err)
	assert.Nil(t, value)
}

func TestTieredDelete(t *testing.T) {
	mcm := &MockCacheMetrics{}
	mcm.On("DeleteHit")
	mtc := TieredCache{
		Local:   NewMockCache(nil),
		Remote:  NewMockCache(nil),
		Metrics: mcm,
	}
	mtc.Local.(*MockCache).Cache["test-key"] = []byte("test-value")
	mtc.Remote.(*MockCache).Cache["test-key"] = []byte("test-value")
	err := mtc.Delete("test-key")
	assert.Nil(t, err)
	localValue, localOk := mtc.Local.(*MockCache).Cache["test-key"]
	assert.False(t, localOk)
	assert.Nil(t, localValue)
	remoteValue, remoteOk := mtc.Remote.(*MockCache).Cache["test-key"]
	assert.False(t, remoteOk)
	assert.Nil(t, remoteValue)
	mcm.AssertCalled(t, "DeleteHit")
}

func TestTieredDeleteError(t *testing.T) {
	mcm := &MockCacheMetrics{}
	mcm.On("DeleteMiss")
	mtc := TieredCache{
		Local:   NewMockCache(nil),
		Remote:  NewMockCache(nil),
		Metrics: mcm,
	}
	err := mtc.Delete("test-key")
	assert.NotNil(t, err)
	mcm.AssertCalled(t, "DeleteMiss")
}

func TestTieredPurge(t *testing.T) {
	mcm := &MockCacheMetrics{}
	mcm.On("PurgeHit")
	mtc := TieredCache{
		Local:   NewMockCache(nil),
		Remote:  NewMockCache(nil),
		Metrics: mcm,
	}
	mtc.Local.(*MockCache).Cache["test-key"] = []byte("test-value")
	mtc.Remote.(*MockCache).Cache["test-key"] = []byte("test-value")
	err := mtc.Purge()
	assert.Nil(t, err)
	localValue, localOk := mtc.Local.(*MockCache).Cache["test-key"]
	assert.False(t, localOk)
	assert.Nil(t, localValue)
	remoteValue, remoteOk := mtc.Remote.(*MockCache).Cache["test-key"]
	assert.False(t, remoteOk)
	assert.Nil(t, remoteValue)
	mcm.AssertCalled(t, "PurgeHit")
}

func TestTieredPurgeError(t *testing.T) {
	// Here we are just lazy -- the only way to get DEL to fail is to break the connection.
	// Given that, do not create a miniredis to connect to.
	mockCluster := &redisc.Cluster{StartupNodes: []string{""}}

	mcm := &MockCacheMetrics{}
	mcm.On("PurgeMiss")
	mtc := TieredCache{
		Local:   NewMockCache(nil),
		Remote:  RemoteCache{cluster: mockCluster, HashKey: "hash"},
		Metrics: mcm,
	}
	err := mtc.Purge()
	assert.Error(t, err)
	mcm.AssertCalled(t, "PurgeMiss")
}

type testEncodable struct {
	A int
	B map[int]int
	C *testNestedEncodable
}

type testNestedEncodable struct {
	D string
}

// Test encoding and then decoding a search result from gob
func TestGobCacheEncoder_EncodeDecode(t *testing.T) {
	result := testEncodable{
		A: 5,
		B: map[int]int{0: 5, 1: 6, 2: 7},
		C: &testNestedEncodable{"thank"},
	}
	enc := GobCacheEncoder{}
	bytes, encodeErr := enc.Encode(result)
	require.Nil(t, encodeErr)
	decodedResult := testEncodable{}
	decodeErr := enc.Decode(bytes, &decodedResult)
	require.Nil(t, decodeErr)
	assert.Equal(t, result, decodedResult)
}

// Test that an error is returned if an error occurs encoding a search result
func TestGobCacheEncoder_Error(t *testing.T) {
	// nil pointer should cause gob to error
	enc := GobCacheEncoder{}
	_, err := enc.Encode(nil)
	assert.Error(t, err)
}
