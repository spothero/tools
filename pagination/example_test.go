package pagination

import (
	"fmt"
)

type Thing string

func (o Thing) ExtractID() string {
	return string(o)
}

func ExamplePagination() {
	pages := []Pageable{
		Thing("foo"),
		Thing("bar"),
		Thing("baz"),
		Thing("boo"),
		Thing("bla"),
	}

	page := GetPageAfterID(pages, "bar", 2)

	fmt.Printf("%v", page)
	//Output: [baz, boo]
}
