package core

import (
	"sync"

	"github.com/golang/geo/s1"
	"github.com/golang/geo/s2"
)

// EarthRadiusMeters is an approximate representation of the earth's radius in meters.
const EarthRadiusMeters = 6371008.8

// maxCellLevel is the number of levels to specify a leaf cell in s2 -- this is copied
// from s2 because they do not export this value
const maxCellLevel = 30

// cellItems is a map of cell ids to the id of an item geographically contained in that cell
type cellItems map[uint64]map[int]bool

// itemIndex keeps track of which cells a given item belongs to in order to enable fast deletions
type itemIndex struct {
	cellPosition uint64
	cellLevel    int
}

// GeoLocationCache implements the GeoLocationCollection interface and provides a location based
// cache
type GeoLocationCache struct {
	// cells is a map of cell level to the items contained in each cell at that zoom level
	cells map[int]cellItems
	// items maps each item id stored to its associated cells to enable fast deletions
	items map[int][]itemIndex
	mutex sync.RWMutex
}

// GeoLocationCollection defines the interface for interacting with Geo-based collections
type GeoLocationCollection interface {
	Set(id int, latitude, longitude float64)
	Delete(id int)
	ItemsWithinDistance(latitude, longitude, distanceMeters float64, params SearchCoveringParameters) ([]int, SearchCoveringResult)
}

// NewGeoLocationCache constructs a new GeoLocationCache object
func NewGeoLocationCache() *GeoLocationCache {
	return &GeoLocationCache{
		cells: make(map[int]cellItems),
		items: make(map[int][]itemIndex),
	}
}

// Set adds an item by id to the geo location cache at a particular latitude and longitude
func (glc *GeoLocationCache) Set(id int, latitude, longitude float64) {
	glc.mutex.Lock()
	defer glc.mutex.Unlock()
	if _, ok := glc.items[id]; ok {
		glc.delete(id, false)
	} else {
		glc.items[id] = make([]itemIndex, 0)
	}
	leafCellID := s2.CellIDFromLatLng(s2.LatLngFromDegrees(latitude, longitude))
	for level := maxCellLevel; level >= 0; level-- {
		if _, ok := glc.cells[level]; !ok {
			glc.cells[level] = make(cellItems)
		}
		cellPos := leafCellID.Parent(level).Pos()
		if _, ok := glc.cells[level][cellPos]; !ok {
			glc.cells[level][cellPos] = map[int]bool{id: true}
		} else {
			glc.cells[level][cellPos][id] = true
		}
		ii := itemIndex{
			cellPosition: cellPos,
			cellLevel:    level,
		}
		glc.items[id] = append(glc.items[id], ii)
	}
}

// Delete removes an item by id from the geo location cache
func (glc *GeoLocationCache) Delete(id int) {
	glc.delete(id, true)
}

// delete is the internal version of Delete that actually performs the deletion.
// This function contains an optional lockMutex flag. The mutex should always
// be locked when deleting unless it is locked by the caller.
func (glc *GeoLocationCache) delete(id int, lockMutex bool) {
	if lockMutex {
		glc.mutex.Lock()
		defer glc.mutex.Unlock()
	}
	itemIndices, ok := glc.items[id]
	if !ok {
		return
	}
	for _, index := range itemIndices {
		delete(glc.cells[index.cellLevel][index.cellPosition], id)
	}
	delete(glc.items, id)
}

// SearchCoveringResult are the boundaries of the cells used in the requested search
type SearchCoveringResult [][][]float64

// SearchCoveringParameters controls the algorithm and parameters used by S2 to determine the covering for the
// requested search area
type SearchCoveringParameters struct {
	LevelMod        int  `json:"level_mod"`
	MaxCells        int  `json:"max_cells"`
	MaxLevel        int  `json:"max_level"`
	MinLevel        int  `json:"min_level"`
	UseFastCovering bool `json:"use_fast_covering"`
}

// ItemsWithinDistance returns all ids stored in the geo location cache within a distanceMeters radius from the provided
// latitude an longitude. Note that this is an approximation and items further than distanceMeters may be returned, but
// it is guaranteed that all item ids returned are within distanceMeters. The caller of this function
// must specify all parameters used to generate cell covering as well as whether or not the coverer will use the
// standard covering algorithm or the fast covering algorithm which may be less precise.
func (glc *GeoLocationCache) ItemsWithinDistance(
	latitude, longitude, distanceMeters float64, params SearchCoveringParameters,
) ([]int, SearchCoveringResult) {
	// First, generate a spherical cap with an arc length of distanceMeters centered on the given latitude/longitude
	// This is the angle required (in radians) to trace an arc length of distanceMeters on the surface of the sphere
	capAngle := s1.Angle(distanceMeters / EarthRadiusMeters)
	capCenter := NewPointFromLatLng(latitude, longitude)
	searchCap := s2.CapFromCenterAngle(capCenter, capAngle)

	// TODO: tune/make the parameters to RegionCoverer configurable
	coverer := s2.RegionCoverer{
		MaxLevel: params.MaxLevel,
		MinLevel: params.MinLevel,
		LevelMod: params.LevelMod,
		MaxCells: params.MaxCells,
	}
	region := s2.Region(searchCap)
	var cellUnion s2.CellUnion
	if params.UseFastCovering {
		cellUnion = coverer.FastCovering(region)
	} else {
		cellUnion = coverer.Covering(region)
	}

	glc.mutex.RLock()
	defer glc.mutex.RUnlock()
	foundIds := make([]int, 0)
	cellBounds := make(SearchCoveringResult, 0, len(cellUnion))
	for _, cell := range cellUnion {
		// get vertices in counter-clockwise order starting from the lower left
		vertices := make([][]float64, 5)
		for i := 0; i < 4; i++ {
			vertex := s2.CellFromCellID(cell).Vertex(i)
			ll := s2.LatLngFromPoint(vertex)
			vertices[i] = []float64{ll.Lng.Degrees(), ll.Lat.Degrees()}
		}
		// close the polygon loop
		vertices[4] = vertices[0]
		cellBounds = append(cellBounds, vertices)
		for k := range glc.cells[cell.Level()][cell.Pos()] {
			foundIds = append(foundIds, k)
		}
	}

	return foundIds, SearchCoveringResult(cellBounds)
}

// NewPointFromLatLng constructs an s2 point from a lat/lon ordered pair
func NewPointFromLatLng(latitude, longitude float64) s2.Point {
	latLng := s2.LatLngFromDegrees(latitude, longitude)
	return s2.PointFromLatLng(latLng)
}

// EarthDistanceMeters calculates the distance in meters between two points on the surface of the Earth
func EarthDistanceMeters(p1, p2 s2.Point) float64 {
	return float64(p1.Distance(p2)) * EarthRadiusMeters
}
