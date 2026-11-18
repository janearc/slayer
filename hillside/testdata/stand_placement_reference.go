// This program writes stand-placement-reference.json to standard output.
// It is an authored replay fixture, not a measurement of vegetation or
// terrain.
//
// PCG128/DXSM is documented by the Go Authors in math/rand/v2/pcg.go:
// https://go.googlesource.com/go/+/refs/tags/go1.26.10/src/math/rand/v2/pcg.go
// This separate implementation holds the 128-bit state in math/big and
// does one full-width update per draw, rather than the standard library's
// limb arithmetic. It does not import math/rand/v2 or any slayer package.
// The accompanying provenance pins the inspected standard-library sources
// and the explicit fixture parameters.
//
// Run it with: go run hillside/testdata/stand_placement_reference.go
package main

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
)

// Generator constants: the 128-bit multiplier and increment as high and
// low words, the 64-bit DXSM multiplier, and the seed words 7 and 1.
const (
	multiplierHigh = 2549297995355413924
	multiplierLow  = 4865540595714422341
	incrementHigh  = 6364136223846793005
	incrementLow   = 1442695040888963407
	dxsmMultiplier = 0xda942042e4dd58b5
	seedHigh       = 7
	seedLow        = 1
	wordBits       = 64
	stateBits      = 128
	// mantissaBits is the width of the low bits a float64 draw keeps.
	mantissaBits = 53
)

// Scene geometry: the west and north edges, the extent across and down,
// the cell spacing and the one-cell margin, all in metres.
const (
	west    = 1001.0
	north   = 8999.0
	across  = 146
	down    = 96
	spacing = 2
	margin  = 2
	// rowStride is the raster samples per row.
	rowStride = 1000
	// heightDivisor turns a raster sample into metres.
	heightDivisor = 8.0
	// lowest and highest bound the strict height band in metres.
	lowest  = 2000
	highest = 4000
	// specimens is how many places are kept; attemptLimit bounds draws.
	specimens    = 12
	attemptLimit = 4096
	// yawModulus is the number of tenths of a degree in a turn.
	yawModulus = 3600
)

// SplitMix64 constants for the yaw finalizer.
const (
	goldenGamma = 0x9e3779b97f4a7c15
	mixFirst    = 0xbf58476d1ce4e5b9
	mixSecond   = 0x94d049bb133111eb
	mixShiftOne = 30
	mixShiftTwo = 27
	mixShiftEnd = 31
)

// formSeeds is the authored profile seed for each specimen, four
// profiles repeated three times.
var formSeeds = []uint64{28, 29, 30, 31, 28, 29, 30, 31, 28, 29, 30, 31}

// place is one accepted specimen.
type place struct {
	east, north, height float64
	yaw                 uint64
}

// generator holds the 128-bit PCG state.
type generator struct {
	state, multiplier, increment, mask *big.Int
}

// word joins a high and a low 64-bit word into one integer.
func word(high, low uint64) *big.Int {
	joined := new(big.Int).Lsh(new(big.Int).SetUint64(high), wordBits)
	return joined.Or(joined, new(big.Int).SetUint64(low))
}

// newGenerator returns the generator seeded with seedHigh and seedLow.
func newGenerator() *generator {
	mask := new(big.Int).Lsh(big.NewInt(1), stateBits)
	return &generator{
		state:      word(seedHigh, seedLow),
		multiplier: word(multiplierHigh, multiplierLow),
		increment:  word(incrementHigh, incrementLow),
		mask:       mask.Sub(mask, big.NewInt(1)),
	}
}

// uniform advances the state once and returns a float64 in [0, 1) from
// the DXSM output's low 53 bits.
func (g *generator) uniform() float64 {
	g.state.Mul(g.state, g.multiplier)
	g.state.Add(g.state, g.increment)
	g.state.And(g.state, g.mask)
	high := new(big.Int).Rsh(g.state, wordBits).Uint64()
	low := new(big.Int).And(g.state, new(big.Int).SetUint64(math.MaxUint64))
	high ^= high >> (wordBits / 2)
	high *= dxsmMultiplier
	high ^= high >> (wordBits * 3 / 4)
	high *= low.Uint64() | 1
	return float64(high&(1<<mantissaBits-1)) / (1 << mantissaBits)
}

// mix is the XOR-lane SplitMix64 finalizer of the authored replay policy.
func mix(seed uint64, index int) uint64 {
	value := seed ^ (uint64(index+1) * goldenGamma)
	value = (value ^ (value >> mixShiftOne)) * mixFirst
	value = (value ^ (value >> mixShiftTwo)) * mixSecond
	return value ^ (value >> mixShiftEnd)
}

// attempt draws one candidate and reports whether its height is in band.
func (g *generator) attempt(index int) (place, bool) {
	// float64 conversions stop the compiler fusing multiply and add.
	east := west + margin + float64(across*g.uniform())
	northing := north - margin - float64(down*g.uniform())
	column := math.Round((east - west) / spacing)
	row := math.Round((north - northing) / spacing)
	height := (column + rowStride*row) / heightDivisor
	if !(height > lowest && height < highest) {
		return place{}, false
	}
	yaw := mix(formSeeds[index], index) % yawModulus
	return place{east, northing, height, yaw}, true
}

// float renders a float64 as the fixture has always written it: shortest
// digits, and a trailing .0 on whole numbers.
func float(value float64) string {
	text := strconv.FormatFloat(value, 'f', -1, 64)
	if !strings.Contains(text, ".") {
		text += ".0"
	}
	return text
}

// document renders the fixture the way the original producer did: two
// space indent, one key per line, and a closing newline.
func document(attempts int, places []place) string {
	var out strings.Builder
	out.WriteString("{\n  \"seed\": [\n    7,\n    1\n  ],\n")
	fmt.Fprintf(&out, "  \"attempts\": %d,\n  \"places\": [\n", attempts)
	for index, each := range places {
		out.WriteString("    {\n")
		out.WriteString("      \"east\": " + float(each.east) + ",\n")
		out.WriteString("      \"north\": " + float(each.north) + ",\n")
		out.WriteString("      \"height\": " + float(each.height) + ",\n")
		fmt.Fprintf(&out, "      \"yaw_tenths\": %d\n    }", each.yaw)
		if index < len(places)-1 {
			out.WriteString(",")
		}
		out.WriteString("\n")
	}
	out.WriteString("  ]\n}\n")
	return out.String()
}

// main draws candidates until twelve specimens are placed and prints the
// fixture.
func main() {
	source := newGenerator()
	var places []place
	attempts := 0
	for len(places) < specimens {
		attempts++
		if attempts > attemptLimit {
			panic("reference did not find its twelve specimens")
		}
		if found, ok := source.attempt(len(places)); ok {
			places = append(places, found)
		}
	}
	fmt.Print(document(attempts, places))
}
