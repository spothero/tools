package tools

import "time"

var locationsCache = make(map[string]*time.Location)

// LoadLocation is a drop-in replacement for time.LoadLocation that caches all loaded locations so that subsequent
// loads do not require additional filesystem lookups.
func LoadLocation(name string) (*time.Location, error) {
	if _, ok := locationsCache[name]; ok {
		return locationsCache[name], nil
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, err
	}
	locationsCache[name] = loc
	return loc, nil
}
