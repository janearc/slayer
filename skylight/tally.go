package skylight

import "math"

// a shape is the bins of a field at each kept height: local solar
// zeniths, view cosines and azimuths.
type shape struct {
	zeniths, views, azimuths int
}

// index is where a bin lies in a field's flat list: by kept height,
// then zenith, then view, then azimuth.
func (s shape) index(keep, zenith, view, azimuth int) int {
	return ((keep*s.zeniths+zenith)*s.views+view)*s.azimuths + azimuth
}

// a tally is one walker's sums: for each bin, the weight of the
// crossings in it, each over the cosine of its way from the vertical,
// and how many crossings there were.
type tally struct {
	shape
	sums   []float64
	counts []int64
}

// newTally is an empty tally of the grid's shape.
func newTally(grid Grid) tally {
	size := len(grid.Heights) * grid.Zeniths * grid.Views * grid.Azimuths
	return tally{shape: shape{grid.Zeniths, grid.Views, grid.Azimuths},
		sums: make([]float64, size), counts: make([]int64, size)}
}

// add tallies a crossing of the kept height keep at position by a
// photon going along heading with weight.
//
// The observer's view is the reverse of the photon's way. Its azimuth is
// from the sun's way along the ground there, folded so that either side
// of it is one: at the subsolar and antisolar points, where the sun has
// no way along the ground, it is nought, since there both parts are.
func (t *tally) add(keep int, position, heading vector, weight float64) {
	up := position.unit()
	view := -heading.dot(up)
	zenith := math.Atan2(math.Hypot(position.x, position.y), position.z)
	bin := t.index(keep, binOf(zenith/math.Pi, t.zeniths),
		binOf((view+one)/two, t.views),
		binOf(azimuthOf(up, heading, view)/math.Pi, t.azimuths))
	t.sums[bin] += weight * slant(view)
	t.counts[bin]++
}

// azimuthOf is the angle, from nought to pi, between the ground-level
// parts of the view, the reverse of heading, and of the way to the sun,
// plus z, at a place whose vertical is up. Each part is measured along
// the sun's ground-level way and across it, which both scale alike.
func azimuthOf(up, heading vector, view float64) float64 {
	along := -heading.z - up.z*view
	across := heading.y*up.x - heading.x*up.y
	return math.Atan2(math.Abs(across), along)
}

// slant is one over the size of a crossing's cosine from the vertical,
// the radiance estimator's weight. Under the grazing cosine it would
// grow without bound, so there it is the mean of one over the cosine
// over crossings in that band, two over grazing, for even radiance.
func slant(cosine float64) float64 {
	size := math.Abs(cosine)
	if size < grazing {
		return two / grazing
	}
	return one / size
}

// binOf is the bin, of bins, that a share from nought to one falls in;
// a share at one, or past either end by a rounding, is in the end bin.
func binOf(share float64, bins int) int {
	return min(max(int(share*float64(bins)), none), bins-one)
}
