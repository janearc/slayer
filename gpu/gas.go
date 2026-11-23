package gpu

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/trace"
)

// the numbers of a gas's phase function that the kernel is told, each true
// by definition: gasOn is the flag of a gas volume; rayleighLobe is three
// over sixteen pi, the constant of rayleigh's lobe, which integrates to one
// over the sphere.
//
// fourPi is the area of the unit sphere in steradians, the divisor of a
// henyey-greenstein lobe; lobePower is the exponent of that lobe's
// denominator, three halves; and lobeTwice is the two in its cross term.
const (
	gasOn        = 1
	rayleighLobe = 3 / (16 * math.Pi)
	fourPi       = 4 * math.Pi
	lobePower    = 1.5
	lobeTwice    = 2
)

// airing is a gas as the kernel reads one: its extinction and scatter at
// the eye's three bands, per unit of its medium's space at unit density,
// and the asymmetry of its phase function.
type airing struct {
	extinction, scatter maths.Light
	asymmetry           float64
}

// airingOf is a gas medium as the kernel reads it, or why it cannot be:
// the numbers are the tracer's own, asked of trace for the eye's bands,
// and an asymmetry that is no lobe is refused where the tracer panics.
func airingOf(air gas.Optical) (*airing, error) {
	found := air.Gas()
	if !found.IsLobe() {
		return nil, unsupported{fmt.Sprintf(
			"a gas whose asymmetry is %v, which is no lobe: it is between "+
				"minus one and one", found.Phase)}
	}
	extinction, scatter := trace.EyeOptics(found)
	return &airing{extinction: extinction, scatter: scatter,
		asymmetry: found.Phase}, nil
}

// shapeOf is the medium that gives a gas its shape and its density from
// place to place: the shape of a gas.Of, or any other gas medium itself.
func shapeOf(air gas.Optical) medium.Medium {
	if over, ok := air.(gas.Of); ok {
		return over.Shape
	}
	return air
}

// noAir fills the fields of a volume that is not a gas, and leaves its
// box in the world empty: every writer of a volume's row calls it, and
// boxVolumes writes the box once every row is in.
func noAir(given values) {
	given["scatter"], given["air"] = nil, nil
	given["thermal"], given["thin"], given["phase"] = wholes(-1, 0), nil, nil
	given["beam"], given["flow"] = nil, nil
	given["world_low"], given["world_high"] = nil, nil
}

// putLast writes words over the start of a field of the volume appended
// last.
func (encoded *scene) putLast(name string, words []uint32) {
	row := (encoded.counts.volumes - 1) * volumeStruct.size()
	start := row + fieldOffset(volumeStruct, name)
	for part, word := range words {
		binary.LittleEndian.PutUint32(encoded.volumes.bytes[start+4*part:],
			word)
	}
}

// floatWords are numbers as the words of the floats that hold them.
func floatWords(numbers ...float64) []uint32 {
	words := make([]uint32, len(numbers))
	for part, number := range numbers {
		words[part] = math.Float32bits(float32(number))
	}
	return words
}

// airLast makes the volume appended last a gas: its tint is its
// extinction, its scatter and asymmetry are written, and it is flagged.
func (encoded *scene) airLast(found airing) {
	extinction, scatter := found.extinction, found.scatter
	encoded.putLast("tint", floatWords(extinction.R, extinction.G,
		extinction.B))
	encoded.putLast("scatter", floatWords(scatter.R, scatter.G, scatter.B,
		found.asymmetry))
	encoded.putLast("air", []uint32{gasOn})
}
