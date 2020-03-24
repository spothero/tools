package pagination

// IDExtractor extracts an ID from an untyped parameter. It's expected that
// IDExtractors will downcast element to the expected type and panic if the
// downcast fails.
type IDExtractor func(element interface{}) interface{}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Pageable is an type that can be paginated when present in a collection.
type Pageable interface {
	// ExtractID derives an identifier that is guaranteed to be unique relative
	// to any elements with which it may coexist in a given collection
	ExtractID() string
}

// GetPageAfterID paginates elements. The first element of the returned page
// directly follows the element in elements having an ID equivalent to after. At
// most pageSize results are included in the returned page.
func GetPageAfterID(
	elements []Pageable,
	after string,
	pageSize uint,
) []Pageable {
	if after == "" {
		// zero-after indicates start with the first page
		return elements[0:minInt(int(pageSize), len(elements))]
	}

	afterIdx := -1
	for idx, element := range elements {
		if after == element.ExtractID() {
			afterIdx = idx
		}
	}

	if afterIdx == -1 {
		return []Pageable{}
	}

	start := afterIdx + 1
	end := minInt(start+int(pageSize), len(elements))

	return elements[start:end]
}
