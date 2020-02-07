package currency

import "math"

// PenniesInDollar represents that there are 100 pennies in a dollar
const PenniesInDollar float64 = 100

// DollarsToPennies converts a floating point value in dollars to pennies. If `roundUp` is true any
// fractional value that is less than 1 penny will be rounded up and added as an additional penny,
// eg 13.001 -> 1301. If the dollars amount is negative, and there is a fractional amount
// remaining and roundUp is true, the fractional amount will be rounded to a penny and added
// negatively to the output value, eg (-13.001 -> -1301). Otherwise, if roundUp is false, that
// value will be discarded.
func DollarsToPennies(dollars float64, roundUp bool) int {
	// Multiply the dollars amount by PenniesInDollar to convert to pennies.
	// Split the floating point penny number around the decimal point. The left portion is the
	// integer and the right part is the remaining fraction.
	integerComponent, fractionComponent := math.Modf(dollars * PenniesInDollar)
	// Convert the integerComponent (which is a float64) to a int to get the base dollar amount
	pennies := int(integerComponent)
	// If there is any additional remaining fractional amount, always round up to the nearest
	// penny and add that to the fixed amount.
	if roundUp && fractionComponent > 0 {
		pennies++
	} else if roundUp && fractionComponent < 0 {
		pennies--
	}
	return pennies
}
