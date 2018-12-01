package core

import (
	"encoding/gob"

	"github.com/stretchr/testify/mock"
)

// MockGeoLocationCache mocks the GeoLocationCache implementation for use in tests
type MockGeoLocationCache struct {
	mock.Mock
}

func init() {
	gob.Register(&MockGeoLocationCache{})
}

// ItemsWithinDistance is a mocked version of ItemsWithinDistance
func (m *MockGeoLocationCache) ItemsWithinDistance(
	latitude, longitude, distanceMeters float64, params SearchCoveringParameters,
) ([]int, SearchCoveringResult) {
	args := m.Called(latitude, longitude, distanceMeters)
	return args.Get(0).([]int), args.Get(1).(SearchCoveringResult)
}

// Set is a mocked version of Set
func (m *MockGeoLocationCache) Set(id int, latitude, longitude float64) {
	m.Called(id, latitude, longitude)
}

// Delete is a mocked version of Delete
func (m *MockGeoLocationCache) Delete(id int) {
	m.Called(id)
}
