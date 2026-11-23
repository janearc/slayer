package oceantex

import (
	"math"
	"runtime"
	"sync"
)

var (
	// crestRounds is how many fixed-point rounds find where the water
	// at a place came from, when a field is read where its water is.
	crestRounds = int(num("crest.rounds"))

	// BreakingAcceleration is the downward acceleration of the surface,
	// as a share of gravity, past which a wave breaks: the threshold of
	// Snyder and Kennedy's mechanism for whitecaps.
	BreakingAcceleration = num("breaking.acceleration")

	// MirrorCalm is the wind, in metres a second, under which the sea is
	// like a mirror: the top of Beaufort force 0, one knot.
	MirrorCalm = num("beaufort.mirror")
)

// CubicTaps is how many samples a side cubic convolution reads.
const CubicTaps = 4

// CascadeCut is the share of its grid's Nyquist wave number each band of
// WaveCascades carries waves up to: a band's shortest wave is two over
// this many samples long.
func CascadeCut() float64 { return cascadeCut }

// CubicWeights are cubic convolution's four weights at a fraction t of
// the way from the second sample to the third (Keys 1981, eq. 15 with a
// = -1/2).
func CubicWeights(t float64) [CubicTaps]float64 {
	square, cube := t*t, t*t*t
	return [CubicTaps]float64{(-cube + 2*square - t) / 2,
		(3*cube - 5*square + 2) / 2, (-3*cube + 4*square + t) / 2,
		(cube - square) / 2}
}

// Smooth is one of the field's grids at a place in metres, read by cubic
// convolution (Keys 1981, a = -1/2, the Catmull-Rom spline): it passes
// through every sample as the straight-line read does, and its slope is
// continuous across sample lines, where the straight-line read's kinks.
// The field repeats in every direction.
func (f *Field) Smooth(grid []float64, x, y float64) float64 {
	gx := x / f.Extent * float64(f.N)
	gy := y / f.Extent * float64(f.N)
	xFloor, yFloor := math.Floor(gx), math.Floor(gy)
	across := CubicWeights(gx - xFloor)
	down := CubicWeights(gy - yFloor)
	var total float64
	for row := range CubicTaps {
		at := wrapped(int(yFloor)+row-1, f.N) * f.N
		var line float64
		for column := range CubicTaps {
			line += across[column] *
				grid[at+wrapped(int(xFloor)+column-1, f.N)]
		}
		total += down[row] * line
	}
	return total
}

// wrapped is index taken into a grid of count samples that repeats.
func wrapped(index, count int) int {
	return (index%count + count) % count
}

// Crested is the field read where its water is. A Field says, of the
// water that rests at each sample, how high the waves lift it and how far
// they carry it sideways; a surface drawn from the heights alone has the
// round crests of a sum of sines.
//
// Crested says, at each sample, the height of the water the waves have
// carried there, which is Gerstner's wave for one wave and Tessendorf's
// choppy sea for many: crests peaked, troughs long and flat. The
// displacement is the waves' own, times choppy.scale, and nothing is
// sharpened by eye.
//
// In the field it returns, Height, SlopeX and SlopeY are the carried
// surface's height and slope at the sample, the mean height taken out so
// the sea keeps its level; PushX and PushY are how far the water now at
// the sample has been carried, so it came from the sample less the push;
// and the stretches are that push's gradient there, one less the inverse
// of the carrying's own Jacobian.
//
// Where the carrying would fold the surface through itself the Jacobian's
// determinant is held at view.fold.floor.
func (f *Field) Crested() *Field {
	cells := f.N * f.N
	made := &Field{N: f.N, Extent: f.Extent,
		Height: make([]float64, cells), SlopeX: make([]float64, cells),
		SlopeY: make([]float64, cells), PushX: make([]float64, cells),
		PushY: make([]float64, cells), StretchXX: make([]float64, cells),
		StretchXY: make([]float64, cells), StretchYY: make([]float64, cells)}
	var workers sync.WaitGroup
	rows := make(chan int)
	for range min(runtime.GOMAXPROCS(0), f.N) {
		workers.Go(func() {
			for row := range rows {
				made.carryRow(f, row)
			}
		})
	}
	for row := range f.N {
		rows <- row
	}
	close(rows)
	workers.Wait()
	made.level()
	return made
}

// carryRow fills one row of a crested field from the field it is of.
func (made *Field) carryRow(f *Field, row int) {
	texel := f.Extent / float64(f.N)
	for column := range f.N {
		x, y := float64(column)*texel, float64(row)*texel
		px, py := f.source(x, y)
		made.carry(row*f.N+column, f, px, py)
	}
}

// source is where the water the waves have carried to (x, y) came from:
// the fixed point of p = (x, y) - push(p), by crestRounds rounds.
func (f *Field) source(x, y float64) (px, py float64) {
	px, py = x, y
	for range crestRounds {
		px, py = x-chop*f.Smooth(f.PushX, px, py),
			y-chop*f.Smooth(f.PushY, px, py)
	}
	return px, py
}

// carry fills sample at of a crested field with the water that came from
// (px, py) of the field it is of. The carried surface's slope is the
// resting surface's through the inverse of the carrying's Jacobian, which
// is symmetric.
func (made *Field) carry(at int, f *Field, px, py float64) {
	a := 1 + chop*f.Smooth(f.StretchXX, px, py)
	b := chop * f.Smooth(f.StretchXY, px, py)
	d := 1 + chop*f.Smooth(f.StretchYY, px, py)
	det := math.Max(a*d-b*b, foldFloor)
	xx, xy, yy := d/det, -b/det, a/det
	slopeX, slopeY := f.Smooth(f.SlopeX, px, py), f.Smooth(f.SlopeY, px, py)
	made.Height[at] = f.Smooth(f.Height, px, py)
	made.SlopeX[at] = xx*slopeX + xy*slopeY
	made.SlopeY[at] = xy*slopeX + yy*slopeY
	made.PushX[at] = chop * f.Smooth(f.PushX, px, py)
	made.PushY[at] = chop * f.Smooth(f.PushY, px, py)
	made.StretchXX[at], made.StretchXY[at], made.StretchYY[at] = 1-xx,
		-xy, 1-yy
}

// level takes the mean out of a field's heights. Carried water gathers
// at the crests, so the carried surface's mean is a little under the
// resting one's; the sea's level is the set's to say, not the waves'.
func (made *Field) level() {
	var sum float64
	for _, height := range made.Height {
		sum += height
	}
	mean := sum / float64(len(made.Height))
	for index := range made.Height {
		made.Height[index] -= mean
	}
}

// WhitecapReflectance is the effective reflectance in the visible of
// whitecaps of every age at sea, Koepke 1984 (koepke.foam.reflectance).
func WhitecapReflectance() float64 { return foamReflectance }

// Whitecaps is the share of the open sea that whitecaps cover under a
// wind in metres a second ten metres up: Monahan and O'Muircheartaigh's
// fit (monahan.scale, monahan.power), foam of every age together.
func Whitecaps(wind float64) float64 { return whitecaps(wind) }
