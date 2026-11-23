package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/texture"
)

// HarborSealCoat paints a light-phase spotted coat in rest coordinates.
// The colour pattern is chosen for the shot and repeatable by seed.
func HarborSealCoat(seed uint64) texture.Texture { return sealCoat{seed: seed} }

type sealCoat struct{ seed uint64 }

// At is the coat's colour at a point, with no footprint to average over.
func (coat sealCoat) At(point maths.Vec) maths.Light {
	return coat.Within(point, 0)
}

// ground is the coat's colour under its marks, dark on the back and
// light on the belly with a broad mottling, and how far the point lies
// on the light side, from none to one.
func (coat sealCoat) ground(point maths.Vec) (maths.Light, float64) {
	lightSide := 1 - maths.Smooth(-.16, .19, point.Y)
	mottling := noise.Gradient(coat.seed+1, point.X*5, point.Y*7, point.Z*7)
	back := maths.RGB(.24, .235, .205)
	belly := maths.RGB(.52, .485, .410)
	return back.Scale(1 - lightSide).Add(belly.Scale(lightSide)).
		Scale(1 + .26*mottling), lightSide
}

// sealMarkColour is the colour of the coat's marks at their darkest.
var sealMarkColour = maths.RGB(.032, .030, .023)

// sealBellyMarkFade is how much fainter a mark is on the light belly.
const sealBellyMarkFade = .29

// Within is the coat's colour at a point, averaged over a footprint in
// metres: the ground, with spots cut over it from a warped cell pattern.
func (coat sealCoat) Within(point maths.Vec, footprint float64) maths.Light {
	base, lightSide := coat.ground(point)

	// Work in centimetre-scale cells, with a slowly varying domain warp
	// to break up repeated spot outlines and mix blotches with speckles.
	pattern := coat.pattern(point)
	speckle := maths.Smooth(.24, .59,
		noise.Gradient(coat.seed+3, point.X*94, point.Y*94, point.Z*94))
	speckle *= 1 - maths.Smooth(.0025, .012, footprint)
	mark := math.Max(coat.spots(pattern, footprint, 1), .46*speckle)
	mark *= 1 - sealBellyMarkFade*lightSide
	return base.Scale(1 - mark).Add(sealMarkColour.Scale(mark))
}

// pattern is the point in spot-cell coordinates, warped by a slowly
// varying domain warp.
func (coat sealCoat) pattern(point maths.Vec) maths.Vec {
	warp := noise.Gradient(coat.seed+2, point.X*48, point.Y*48, point.Z*48)
	return maths.V(point.X*16+warp*.27, point.Y*21+warp*.21, point.Z*21-warp*.24)
}

// spots is how far a point lies in a spot, with every spot's radius
// multiplied by grow. A grow above one gives the spots' halos.
func (coat sealCoat) spots(at maths.Vec, footprint, grow float64) float64 {
	x, y, z := math.Floor(at.X), math.Floor(at.Y), math.Floor(at.Z)
	fade := .043 + math.Min(.32, footprint*11)
	var mark float64
	const span = 3
	for dx := range span {
		for dy := range span {
			for dz := range span {
				left, below, behind := x+float64(dx-1), y+float64(dy-1), z+float64(dz-1)
				// All spot centres stay inside the central 70% of the cell.
				if coatCellGap(at.X, left) > .76 || coatCellGap(at.Y, below) > .76 ||
					coatCellGap(at.Z, behind) > .76 {
					continue
				}
				mark = math.Max(mark, coat.cellSpot(at, left, below, behind, fade, grow))
			}
		}
	}
	return mark
}

// coatCellGap is how far a coordinate lies outside the middle of its cell,
// between .15 and .85 along from the corner: negative inside it.
func coatCellGap(coordinate, corner float64) float64 {
	return math.Max(corner+.15-coordinate, coordinate-corner-.85)
}

// cellSpot is the strength at a point of the spot its cell holds: the
// centre and radius come from the cell's hash, the radius grown by grow and
// the edge softened by fade.
func (coat sealCoat) cellSpot(at maths.Vec, x, y, z, fade, grow float64) float64 {
	i, j, k := int64(x), int64(y), int64(z)
	radius := grow * (.22 + .21*noise.Hash3(coat.seed+4, i, j, k))
	centre := maths.V(x+.15+.70*noise.Hash3(coat.seed+5, i, j, k),
		y+.15+.70*noise.Hash3(coat.seed+6, i, j, k),
		z+.15+.70*noise.Hash3(coat.seed+7, i, j, k))
	distance := at.Sub(centre).Len()
	return 1 - maths.Smooth(radius-fade, radius+fade, distance)
}
