package branching

import (
	"errors"
	"math"

	"github.com/janearc/slayer/maths"
)

// Draws supplies stable choices for an arrangement without owning a seed.
type Draws interface {
	Draw(row, index, which int) float64
	Swing(row, index, which int) float64
}

// Ring places parts around a centre with optional lean, variation and twist.
// Construct with NewRing to validate its finite dimensions and limits.
type Ring struct {
	Count                              int
	Radius, Lean, Stray, Uneven, Twist float64
}

// ErrRing reports an invalid ring configuration.
var ErrRing = errors.New("manufacture: a ring needs one part or more, " +
	"a radius of nought or above with an end, a lean that is a number, " +
	"a stray and an unevenness from nought to under one, and a twist " +
	"from nought to one")

// NewRing checks the number, radius, lean and variation shares.
func NewRing(count int, radius, lean, stray, uneven, twist float64) (Ring, error) {
	finite := func(n float64) bool { return !math.IsNaN(n) && !math.IsInf(n, 0) }
	if count < 1 || radius < 0 || !finite(radius) || !finite(lean) ||
		!(stray >= 0 && stray < 1) || !(uneven >= 0 && uneven < 1) ||
		!(twist >= 0 && twist <= 1) {
		return Ring{}, ErrRing
	}
	return Ring{count, radius, lean, stray, uneven, twist}, nil
}

// Place is the deterministic pose of one ring member, in the ring's space.
func (ring Ring) Place(row, index int, from Draws) maths.Subspace {
	const fullTurn = 2 * math.Pi
	gap := fullTurn / float64(ring.Count)
	angle := gap * (float64(index) + ring.Stray*from.Swing(row, index, 0))
	out := maths.V(math.Cos(angle), 0, -math.Sin(angle))
	tangent := maths.V(-math.Sin(angle), 0, -math.Cos(angle))
	way := maths.Up.Scale(math.Cos(ring.Lean)).Sub(out.Scale(math.Sin(ring.Lean)))
	facing := maths.Orientation{X: tangent, Y: way, Z: tangent.Cross(way)}
	twist := fullTurn * ring.Twist * from.Draw(row, index, 1)
	return maths.Subspace{
		Origin:      out.Scale(ring.Radius),
		Orientation: facing.Then(maths.About(maths.YAxis, twist)),
		Scale:       1 + ring.Uneven*from.Swing(row, index, 2),
	}
}

// Rows and Across describe one row around the ring.
func (ring Ring) Rows() int      { return 1 }
func (ring Ring) Across(int) int { return ring.Count }

// At locates a member without assigning it a physical size.
func (ring Ring) At(_ Size, row, index int, from Draws) (Place, error) {
	return Place{Space: ring.Place(row, index, from)}, nil
}
