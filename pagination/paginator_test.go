package pagination

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testStruct struct {
	id string
}

func (ts testStruct) ExtractID() string {
	return ts.id
}

func TestMinInt(t *testing.T) {
	assertTest := assert.New(t)

	tests := []struct {
		name     string
		first    int
		second   int
		expected int
	}{
		{
			name:     "first input is larger",
			first:    10,
			second:   20,
			expected: 10,
		},
		{
			name:     "second input is larger",
			first:    20,
			second:   10,
			expected: 10,
		},
		{
			name:     "equal inputs",
			first:    10,
			second:   10,
			expected: 10,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(_ *testing.T) {
			actual := minInt(test.first, test.second)
			assertTest.Equal(actual, test.expected)
		})
	}
}

func TestGetPageByID(t *testing.T) {
	assertTest := assert.New(t)

	tests := []struct {
		name        string
		after       string
		panicString string
		input       []testStruct
		expected    []testStruct
		pageSize    uint
	}{
		{
			name:     "paging from the beginning",
			after:    "",
			pageSize: 1,
			input: []testStruct{
				{id: "a"},
				{id: "b"},
			},
			expected: []testStruct{
				{id: "a"},
			},
			panicString: "",
		},
		{
			name:     "after offset",
			after:    "b",
			pageSize: 2,
			input: []testStruct{
				{id: "a"},
				{id: "b"},
				{id: "c"},
				{id: "d"},
				{id: "e"},
				{id: "f"},
			},
			expected: []testStruct{
				{id: "c"},
				{id: "d"},
			},
			panicString: "",
		},
		{
			name:     "after offset til the end of the resultset",
			after:    "b",
			pageSize: 10000,
			input: []testStruct{
				{id: "a"},
				{id: "b"},
				{id: "c"},
				{id: "d"},
				{id: "e"},
				{id: "f"},
			},
			expected: []testStruct{
				{id: "c"},
				{id: "d"},
				{id: "e"},
				{id: "f"},
			},
			panicString: "",
		},
		{
			name:     "complete resultset",
			after:    "",
			pageSize: 6,
			input: []testStruct{
				{id: "a"},
				{id: "b"},
				{id: "c"},
				{id: "d"},
				{id: "e"},
				{id: "f"},
			},
			expected: []testStruct{
				{id: "a"},
				{id: "b"},
				{id: "c"},
				{id: "d"},
				{id: "e"},
				{id: "f"},
			},
			panicString: "",
		},
		{
			name:     "nonexistent after",
			after:    "bogus",
			pageSize: 6,
			input: []testStruct{
				{id: "a"},
				{id: "b"},
				{id: "c"},
				{id: "d"},
				{id: "e"},
				{id: "f"},
			},
			expected:    []testStruct{},
			panicString: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(_ *testing.T) {
			elements := make([]Pageable, len(test.input))
			for idx, item := range test.input {
				elements[idx] = item
			}

			var page []Pageable
			if test.panicString == "" {
				assertTest.NotPanics(func() {
					page = GetPageAfterID(elements, test.after, test.pageSize)
				})
			} else {
				assertTest.PanicsWithValue(test.panicString, func() {
					page = GetPageAfterID(elements, test.after, test.pageSize)
				})
			}

			actual := make([]testStruct, len(page))
			for idx, element := range page {
				typed, ok := element.(testStruct)
				if !ok {
					panic("failed downcast in reconstituting typed array")
				}

				actual[idx] = typed
			}

			assertTest.Equal(actual, test.expected)
		})
	}
}
