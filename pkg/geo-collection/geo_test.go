package core

import (
	"bytes"
	"encoding/gob"
	"testing"

	"github.com/golang/geo/s2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testCell = struct {
	cellID   s2.CellID
	lat, lon float64
}

// cell IDs were generated using the Sidewalk Labs s2 demo
// https://s2.sidewalklabs.com/regioncoverer/
var (
	// downtown Chicago
	cell1 = testCell{
		cellID: s2.CellIDFromToken("880e2cbc904a0c29"),
		lat:    41.87963549397698,
		lon:    -87.63028184499035,
	}
	// midtown Manhattan
	cell2 = testCell{
		cellID: s2.CellIDFromToken("89c25900437"),
		lat:    40.75306726395187,
		lon:    -73.98119781456353,
	}
)

func TestGeoLocationCache_Set(t *testing.T) {
	type setParams struct {
		id       int
		lat, lon float64
	}
	type cellContains struct {
		cellID s2.CellID
		itemID int
	}
	tests := []struct {
		name                   string
		params                 []setParams
		expectedCellIDContains []cellContains
	}{
		{
			name:                   "Should set an item in cache",
			params:                 []setParams{{0, cell1.lat, cell1.lon}},
			expectedCellIDContains: []cellContains{{cell1.cellID, 0}},
		}, {
			name:                   "Should set multiple items in cache",
			params:                 []setParams{{0, cell1.lat, cell1.lon}, {1, cell2.lat, cell2.lon}},
			expectedCellIDContains: []cellContains{{cell1.cellID, 0}, {cell2.cellID, 1}},
		}, {
			name:                   "Should replace an item in cache",
			params:                 []setParams{{0, cell1.lat, cell1.lon}, {0, cell2.lat, cell2.lon}},
			expectedCellIDContains: []cellContains{{cell2.cellID, 0}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			glc := NewGeoLocationCache()
			for _, p := range test.params {
				glc.Set(p.id, p.lat, p.lon)
			}
			assert.Len(t, glc.items, len(test.expectedCellIDContains))
			// assert that the location's cell has been cached at every cell level (31 of them)
			assert.Len(t, glc.cells, 31)
			for _, expectedContains := range test.expectedCellIDContains {
				expectedCellID := expectedContains.cellID
				assert.Contains(t, glc.items, expectedContains.itemID)
				assert.Contains(t, glc.cells[expectedCellID.Level()][expectedCellID.Pos()], expectedContains.itemID)
				require.Contains(t, glc.cells[expectedCellID.Level()], expectedCellID.Pos())
			}
		})
	}
}

func TestGeoLocationCache_Delete(t *testing.T) {
	itemID := 0
	populateTestCache := func() *GeoLocationCache {
		glc := NewGeoLocationCache()
		// populate the cache
		for level := maxCellLevel; level >= 0; level-- {
			glc.cells[level] = map[uint64]map[int]bool{
				cell1.cellID.Pos(): {
					itemID: true,
				},
			}
			glc.items[itemID] = append(glc.items[itemID], itemIndex{cellPosition: cell1.cellID.Pos(), cellLevel: level})
		}
		return glc
	}
	tests := []struct {
		name                 string
		deleteID             int
		expectedRemainingIds []int
	}{
		{
			name:                 "Should delete an item in cache",
			deleteID:             itemID,
			expectedRemainingIds: []int{},
		}, {
			name:                 "Deleting an item in cache that does not exist should be ok",
			deleteID:             1,
			expectedRemainingIds: []int{itemID},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			glc := populateTestCache()
			glc.Delete(test.deleteID)
			assert.NotContains(t, glc.items, test.deleteID)
			for level := maxCellLevel; level >= 0; level-- {
				assert.NotContains(t, glc.cells[level][cell1.cellID.Pos()], test.deleteID)
				for remainingID := range test.expectedRemainingIds {
					assert.Contains(t, glc.cells[level][cell1.cellID.Pos()], remainingID)
				}
			}
			for remainingID := range test.expectedRemainingIds {
				assert.Contains(t, glc.items, remainingID)
			}
		})
	}
}

func TestGeoLocationCache_ItemsWithinDistance(t *testing.T) {
	item1ID := 0
	item2ID := 1
	populateTestCache := func() *GeoLocationCache {
		glc := NewGeoLocationCache()
		// populate the cache by placing 2 items at the location of the test cells
		for level := maxCellLevel; level >= 0; level-- {
			glc.cells[level] = map[uint64]map[int]bool{
				cell1.cellID.Parent(level).Pos(): {
					item1ID: true,
				},
				cell2.cellID.Parent(level).Pos(): {
					item2ID: true,
				},
			}
		}
		return glc
	}
	tests := []struct {
		name                                 string
		searchLat, searchLon, distanceMeters float64
		useFastAlgorithm                     bool
		expectedIDs                          []int
	}{
		{
			name:             "Search should return relevant results",
			searchLat:        cell1.lat - 0.001,
			searchLon:        cell1.lon - 0.001,
			distanceMeters:   1000,
			useFastAlgorithm: false,
			expectedIDs:      []int{item1ID},
		},
		{
			name:             "Search should return relevant with the fast cover algorithm",
			searchLat:        cell1.lat - 0.001,
			searchLon:        cell1.lon - 0.001,
			distanceMeters:   1000,
			useFastAlgorithm: true,
			expectedIDs:      []int{item1ID},
		}, {
			name:             "Search should return multiple relevant results",
			searchLat:        cell1.lat,
			searchLon:        cell1.lon,
			distanceMeters:   4000000,
			useFastAlgorithm: false,
			expectedIDs:      []int{item1ID, item2ID},
		}, {
			name:             "Search should return no results when no items are close by",
			searchLat:        0,
			searchLon:        0,
			distanceMeters:   1,
			useFastAlgorithm: false,
			expectedIDs:      []int{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			glc := populateTestCache()
			results, _ := glc.ItemsWithinDistance(
				test.searchLat, test.searchLon, test.distanceMeters, SearchCoveringParameters{
					MaxLevel: 5, MinLevel: 5, LevelMod: 1, MaxCells: 5, UseFastCovering: test.useFastAlgorithm})
			assert.Len(t, results, len(test.expectedIDs))
			for expectedID := range test.expectedIDs {
				assert.Contains(t, results, expectedID)
			}
		})
	}
}

func TestGobEncodeDecode_geoLocationCache(t *testing.T) {
	glc := NewGeoLocationCache()
	glc.cells[0] = map[uint64]map[int]bool{1: {1: true}}
	glc.items[0] = []itemIndex{{cellPosition: 1, cellLevel: 1}}
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(glc)
	require.NoError(t, err)
	dec := gob.NewDecoder(bytes.NewBuffer(buf.Bytes()))
	decodedGLC := &GeoLocationCache{}
	err = dec.Decode(decodedGLC)
	require.NoError(t, err)
	assert.Equal(t, decodedGLC.items, glc.items)
	assert.Equal(t, decodedGLC.cells, glc.cells)
}

func TestEarthDistanceMeters(t *testing.T) {
	// pick 2 points off a map that are roughly 105 meters of each other
	p1 := NewPointFromLatLng(41.883170, -87.632278)
	p2 := NewPointFromLatLng(41.883178, -87.630916)
	assert.InDelta(t, 105, EarthDistanceMeters(p1, p2), 10)
}

func TestGobEncodeDecode_itemIndex(t *testing.T) {
	ii := itemIndex{cellPosition: 1234, cellLevel: 9876}
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(&ii)
	require.NoError(t, err)
	dec := gob.NewDecoder(bytes.NewBuffer(buf.Bytes()))
	decodedII := itemIndex{}
	err = dec.Decode(&decodedII)
	require.NoError(t, err)
	assert.Equal(t, ii, decodedII)
}
