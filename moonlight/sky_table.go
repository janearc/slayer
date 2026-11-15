package moonlight

import (
	"encoding/binary"
	"fmt"
	"maps"
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
)

// A sky's table as bytes, so a run can keep it on disk and the next read
// it back instead of working it out again: every number of the way to
// its light and of its table, each a float64's bits, little-endian. Read
// back, it is the same sky to the bit.

// skyNumbers is how many numbers a sky's bytes hold: the three of its
// light's way, then the three of each point of its table. Defined.
const skyNumbers = 3 + skyRises*skyRounds*3

// float64Bytes is the bytes a float64 takes. Defined.
const float64Bytes = 8

// SkyBytes is how many bytes Bytes gives and SkyFromBytes takes.
const SkyBytes = skyNumbers * float64Bytes

// Bytes is the sky's way to its light and its table, as bytes.
func (sky *Sky) Bytes() []byte {
	numbers := make([]float64, 0, skyNumbers)
	numbers = append(numbers, sky.source.X, sky.source.Y, sky.source.Z)
	for rise := range skyRises {
		for round := range skyRounds {
			point := sky.table[rise][round]
			numbers = append(numbers, point.R, point.G, point.B)
		}
	}
	made := make([]byte, 0, SkyBytes)
	for _, number := range numbers {
		made = binary.LittleEndian.AppendUint64(made,
			math.Float64bits(number))
	}
	return made
}

// SkyFromBytes is the sky Bytes gave, or a refusal of bytes that are not
// a sky's, by their length.
func SkyFromBytes(raw []byte) (*Sky, error) {
	if len(raw) != SkyBytes {
		return nil, fmt.Errorf("moonlight: a sky is %d bytes, not %d",
			SkyBytes, len(raw))
	}
	next := func() float64 {
		number := math.Float64frombits(binary.LittleEndian.Uint64(raw))
		raw = raw[float64Bytes:]
		return number
	}
	sky := &Sky{source: maths.V(next(), next(), next())}
	for rise := range skyRises {
		for round := range skyRounds {
			sky.table[rise][round] = maths.RGB(next(), next(), next())
		}
	}
	return sky, nil
}

// Bytes is the spectrum as bytes, for a key: each nanometre it has, in
// order, and its light there, each as eight bytes, little-endian. Two
// spectra are the same bytes only where every number is the same.
func (light Spectrum) Bytes() []byte {
	waves := slices.Sorted(maps.Keys(light))
	made := make([]byte, 0, len(waves)*2*float64Bytes)
	for _, nm := range waves {
		made = binary.LittleEndian.AppendUint64(made, uint64(nm))
		made = binary.LittleEndian.AppendUint64(made,
			math.Float64bits(light[nm]))
	}
	return made
}
