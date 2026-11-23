// This program writes the authored Gaussian ignition fixture,
// fuel_ignition_reference.json. It is an independent evaluation: the
// exponential is summed as a series in math/big and no slayer package is
// used, so the expected values are not computed by the code under test.
//
// Run it with: go run hillside/testdata/fuel_ignition_reference.go [path]
// The JSON goes to path, or to defaultOutput when no path is given.
package main

import (
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
)

// defaultOutput is where the fixture is written when no path is given.
const defaultOutput = "/tmp/slayer-fuel-ignition-reference.json"

// denominator is the divisor of the squared-radius ratios: 1.6 m squared
// over a half-metre cell, scaled to integers, gives numerators over 512.
const denominator = 512

// peakExcess is the continuous peak excess temperature in kelvin.
const peakExcess = 900

// significantDigits is how many significant decimal digits the
// exponential keeps, as the original producer's BigDecimal series did
// when asked for 80 digits.
const significantDigits = 80

// workingBits is the binary precision of the series sum, well beyond
// significantDigits plus any leading zeros.
const workingBits = 1200

// seriesTerms is the number of series terms summed. The largest argument
// is under 7, so terms past 400 are far below workingBits.
const seriesTerms = 400

// sample names one selected cell centre and its exact squared-radius
// ratio numerator.
type sample struct {
	name      string
	numerator int64
}

// samples lists the selected cell centres in output order.
var samples = []sample{{"central", 25}, {"adjacent", 125}, {"corner", 3425}}

// record text that the fixture carries verbatim.
const (
	purposeText = "Authored Gaussian ignition fixture, " +
		"not a published measurement."
	geometryText = "16x8 half-metre cells, continuous peak excess900K, " +
		"radius1.6m. Squared-radius ratios at selected cell centres " +
		"are exact25/512,125/512,3425/512."
	arithmeticText = "Independent math/big evaluation at " +
		"requested80 decimal digits."
)

// newFloat returns a zero big.Float at working precision.
func newFloat() *big.Float {
	return new(big.Float).SetPrec(workingBits)
}

// exponential returns exp(-numerator/denominator) by its alternating
// power series.
func exponential(numerator int64) *big.Float {
	argument := newFloat().Quo(
		newFloat().SetInt64(numerator), newFloat().SetInt64(denominator))
	term := newFloat().SetInt64(1)
	sum := newFloat().SetInt64(1)
	for k := int64(1); k < seriesTerms; k++ {
		term.Mul(term, argument)
		term.Quo(term, newFloat().SetInt64(k))
		term.Neg(term)
		sum.Add(sum, term)
	}
	return sum
}

// fractionDigits returns the decimal places that give value
// significantDigits significant digits: the leading zeros after the
// point are added to significantDigits.
func fractionDigits(value *big.Float) int {
	places := significantDigits
	shifted := newFloat().Mul(value, newFloat().SetInt64(10))
	for shifted.Cmp(newFloat().SetInt64(1)) < 0 {
		places++
		shifted.Mul(shifted, newFloat().SetInt64(10))
	}
	return places
}

// scaled returns value times ten to the places, rounded half up to a
// whole number.
func scaled(value *big.Float, places int) *big.Int {
	power := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(places)), nil)
	product := newFloat().Mul(value, newFloat().SetInt(power))
	product.Add(product, newFloat().SetFloat64(0.5))
	whole, _ := product.Int(nil)
	return whole
}

// decimal writes an integer carrying places decimal places as plain
// decimal text, dropping trailing zeros but keeping one digit after the
// point.
func decimal(whole *big.Int, places int) string {
	digits := whole.String()
	for len(digits) <= places {
		digits = "0" + digits
	}
	split := len(digits) - places
	fraction := strings.TrimRight(digits[split:], "0")
	if fraction == "" {
		fraction = "0"
	}
	return digits[:split] + "." + fraction
}

// excess returns the excess temperature, in kelvin, as decimal text.
func excess(numerator int64) string {
	value := exponential(numerator)
	places := fractionDigits(value)
	whole := scaled(value, places)
	return decimal(whole.Mul(whole, big.NewInt(peakExcess)), places)
}

// document renders the fixture the way the original producer did: two
// space indent, one key per line, and a closing newline.
func document(values map[string]string) string {
	var out strings.Builder
	out.WriteString("{\n")
	out.WriteString("  \"purpose\": " + strconv.Quote(purposeText) + ",\n")
	out.WriteString("  \"geometry\": " + strconv.Quote(geometryText) + ",\n")
	out.WriteString("  \"arithmetic\": " + strconv.Quote(arithmeticText) + ",\n")
	out.WriteString("  \"excessKelvin\": {\n")
	for index, each := range samples {
		line := "    " + strconv.Quote(each.name) + ": " +
			strconv.Quote(values[each.name])
		if index < len(samples)-1 {
			line += ","
		}
		out.WriteString(line + "\n")
	}
	out.WriteString("  }\n}\n")
	return out.String()
}

// main evaluates the three samples, writes the fixture and prints each
// value.
func main() {
	path := defaultOutput
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	values := map[string]string{}
	for _, each := range samples {
		values[each.name] = excess(each.numerator)
	}
	if err := os.WriteFile(path, []byte(document(values)), 0o644); err != nil {
		panic(err)
	}
	for _, each := range samples {
		fmt.Printf("%s: %s\n", each.name, values[each.name])
	}
}
