package world

import "github.com/janearc/slayer/maths"

// LandShare is how much of the ground under a pixel is land at a point,
// from nought (all sea) to one (all land), by the same reckoning the
// ground is painted by: Scrooged's, the height read from the world's
// tree down to the pixel. angle is the pixel's width as an angle at the
// world's middle in radians, and nought is as fine as a point is read;
// mix is the Scrooged mix, and a mix of nought is the coarse tiles' own
// class, wholly land or wholly sea.
//
// It is what a camera is found from, so that a coast stays where the
// camera is whatever the terrain's numbers do.
func (w *World) LandShare(p maths.Vec, angle, mix float64) float64 {
	point := p.Unit()
	if mix <= 0 {
		if Land(w.Biome(w.Tiling.Nearest(point))) {
			return 1
		}
		return 0
	}
	ground := &scrooged{w: w, tree: w.tree(), mix: mix}
	return ground.landShare(point, angle)
}
