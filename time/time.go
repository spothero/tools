// Copyright 2021 SpotHero
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

package time

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
