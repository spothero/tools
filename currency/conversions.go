package currency

import "math"

// PenniesInDollar represents that there are 100 pennies in a dollar
const PenniesInDollar float64 = 100

// DollarsToPennies converts a floating point value in dollars to pennies. If `roundUp` is true any
// fractional value that is less than 1 penny will be rounded up and added as an additional penny.
// Otherwise, that value will be discarded.
func DollarsToPennies(dollars float64, roundUp bool) uint {
	// Multiply the dollars amount by PenniesInDollar to convert to pennies.
	// Split the floating point penny number around the decimal point. The left portion is the
	// integer and the right part is the remaining fraction.
	integerComponent, fractionComponent := math.Modf(dollars * PenniesInDollar)
	// Convert the integerComponent (which is a float64) to a uint to get the base dollar amount
	pennies := uint(integerComponent)
	// If there is any additional remaining fractional amount, always round up to the nearest
	// penny and add that to the fixed amount.
	if roundUp && fractionComponent > 0 {
		pennies++
	}
	return pennies
}
