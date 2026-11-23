package texture

import (
	_ "embed"
	"fmt"
	"math"
	"sync"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/maths"
)

// nought is defined: no amount.
const nought = 0

// unity is defined: one whole, and the share of everything.
const unity = 1

// pair is defined: two.
const pair = 2

// smoothCubic is defined: the 3 of smoothstep, 3t^2 - 2t^3.
const smoothCubic = 3

// half is defined: a half, the middle of a cell.
const half = 0.5

// fullTurn is defined: a whole turn, in radians.
const fullTurn = 2 * math.Pi

// firstPlace is defined: the first place in a list.
const firstPlace = 0

// orbitPower is defined by Kepler's third law: the angular speed of an
// orbit falls as its radius to the power minus three halves.
const orbitPower = -1.5

// noPaint is no light at all, and a texture that paints nothing gives it.
var noPaint maths.Light

// noAxis is the axis a body has when none is said.
var noAxis maths.Vec

// noNumberLight is light that is no number, the same on every machine:
// what a point that is no number is given to paint.
func noNumberLight() maths.Light {
	return maths.Grey(math.NaN())
}

// clamped is x held between minus one and one.
func clamped(x float64) float64 {
	return math.Max(-unity, math.Min(unity, x))
}

// pickedFile is the picked numbers' file, as a refusal names it.
const pickedFile = "picked-numbers.yaml"

// pickedRaw is the picked numbers as the file holds them.
//
//go:embed picked-numbers.yaml
var pickedRaw []byte

// readPicked is the picked numbers file, read and checked.
func readPicked() (numbers.Numbers, error) {
	return numbers.Read(pickedFile, pickedRaw)
}

// pickedNumbers is the picked numbers file, read once.
var pickedNumbers = sync.OnceValues(readPicked)

// pickedNumber is a number someone chose, by its name in the file. A name
// the file does not hold is a mistake in this package's code, which the
// tests find, so asking for one stops the program and says which.
func pickedNumber(name string) float64 {
	read, err := pickedNumbers()
	if err != nil {
		panic(err)
	}
	number, err := read.Get(name)
	if err != nil {
		panic(fmt.Errorf("texture: %w", err))
	}
	return number.Value
}
