package tools

import (
	"sync"
	"time"
)

// locationsCache holds time.Location objects for re-use
type locationsCache struct {
	cache map[string]*time.Location
	mutex sync.RWMutex
}

var locCache = &locationsCache{
	cache: make(map[string]*time.Location),
}

// LoadLocation is a drop-in replacement for time.LoadLocation that caches all loaded locations so that subsequent
// loads do not require additional filesystem lookups.
func LoadLocation(name string) (*time.Location, error) {
	locCache.mutex.RLock()
	loc, ok := locCache.cache[name]
	locCache.mutex.RUnlock()
	if ok {
		return loc, nil
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, err
	}
	locCache.mutex.Lock()
	defer locCache.mutex.Unlock()
	locCache.cache[name] = loc
	return loc, nil
}
