// Package crystal derives photographic grain from crystal size and density.
// Nutting’s law supplies expected counts; seeded variation approximates the
// count fluctuations that give a developed frame its grain.
package crystal

import (
	"errors"
	"math"

	"github.com/janearc/slayer/ninjafilm/hash"
	"github.com/janearc/slayer/ninjafilm/number"
)

// none is no crystals at all. It is defined as zero.
const none = 0

// whole is the whole of a share. It is defined as one.
const whole = 1

// quarter is what a disc's area divides by. It is pi times its width
// squared over four, defined by the area of a disc.
const quarter = 4

// doubled is the two in Box and Muller's transform. It takes the root
// of minus two times a logarithm and turns by two pi. It is defined by
// that transform.
const doubled = 2

// errCrystal is a crystal with no width.
var errCrystal = errors.New("ninjafilm: a crystal has a finite width " +
	"above nought")

// Crystal is one developed grain of a stock, by its mean width in
// micrometres. The grain of a stock comes from this and nothing else.
type Crystal struct{ across float64 }

// New is a crystal of a width, or why not.
func New(across float64) (made Crystal, err error) {
	if !number.Positive(across) {
		return made, errCrystal
	}
	return Crystal{across}, nil
}

// shadow is the area one crystal blocks in square micrometres. It is a
// disc of its width.
func (grain Crystal) shadow() float64 {
	return math.Pi * grain.across * grain.across / quarter
}

// Expected is how many crystals make a density over an area. It is
// Nutting's law. The density is the crystals' shadows as a share of the
// area, times the logarithm of e.
func (grain Crystal) Expected(density, area float64) float64 {
	return density * area / (math.Log10E * grain.shadow())
}

// Density is what a count of crystals over an area comes to. It undoes
// Expected.
func (grain Crystal) Density(count, area float64) float64 {
	return count * math.Log10E * grain.shadow() / area
}

// Counted is how many crystals are found where some number were
// expected. Crystals fall by chance. So the count wanders about the
// number expected by its square root, as a Poisson count does when it
// is not small.
//
// The draw of chance says how far. There are never fewer than none.
func Counted(expected, chance float64) float64 {
	return math.Max(none, expected+math.Sqrt(expected)*chance)
}

// rest is what is left of a whole when a share is taken out of it.
func rest(share float64) float64 { return whole - share }

// Chance is a draw from the bell curve for a site of a frame. The same seed,
// site and frame give the same draw forever. It is Box and Muller's way from
// two even draws, which the emulsion's hash makes.
func Chance(seed uint64, x, y, frame int64) float64 {
	first := hash.Three(seed, x, y, frame)
	second := hash.Three(^seed, x, y, frame)
	left := math.Log(rest(first))
	radius := math.Sqrt(-doubled * left)
	turn := doubled * math.Pi * second
	return radius * math.Cos(turn)
}
