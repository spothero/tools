package currency

import "fmt"

func ExampleDollarsToPennies() {
    fmt.Println(DollarsToPennies(19.136, true))
    // Output: 1914
}
