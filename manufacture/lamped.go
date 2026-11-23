package manufacture

import (
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
)

// lampBeyond is how far out from a lamp's middle its light starts, in
// metres, so that the lamp's own box does not shade what it lights. A
// choice: past the corner of a lamp's box, a quarter of a metre across.
const lampBeyond = 0.3

// lamped is the good carrying a light for each drawn part that is a lamp:
// a bare light of the lamp's colour, if it has one of its own, and
// otherwise of the black body of its heat, and its luminance, in the kit's
// relative light as a set's lamp is, shining every way from the middle of
// the part's mesh.
func lamped(made good, lamps []cities.Rendered) (good, error) {
	for _, one := range lamps {
		shining, err := lampLight(one.Lamp)
		if err != nil {
			return good{}, err
		}
		beyond := lampBeyond
		if one.Lamp.Beyond > 0 {
			beyond = one.Lamp.Beyond
		}
		made = made.lit(shining, middleOf(one), maths.Up,
			measure.Metres(beyond))
	}
	return made, nil
}

// middleOf is the middle of the box round a drawn part's mesh, where the
// part stands.
func middleOf(one cities.Rendered) maths.Vec {
	inf := math.Inf(1)
	low, high := maths.V(inf, inf, inf), maths.V(-inf, -inf, -inf)
	for _, triangle := range one.Mesh.Triangles {
		for _, corner := range triangle {
			at := one.Place.Out(corner)
			low = maths.V(math.Min(low.X, at.X), math.Min(low.Y, at.Y),
				math.Min(low.Z, at.Z))
			high = maths.V(math.Max(high.X, at.X), math.Max(high.Y, at.Y),
				math.Max(high.Z, at.Z))
		}
	}
	return low.Add(high).Scale(middleShare)
}

// middleShare is how far along from the low corner to the high one the
// middle of a box is.
const middleShare = 0.5
