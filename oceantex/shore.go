package oceantex

import (
	"errors"
	"math"

	"github.com/janearc/celestial/grissom/mesh"
	"github.com/janearc/slayer/titan"
)

var (
	// EarthRadius is the Earth's mean radius in metres, as celestial
	// holds it.
	EarthRadius = num("earth.radius")

	saturated  = num("breaking.saturated")
	crestLimit = num("breaking.crest")
	overRMS    = num("waveheight.rms")
	foamFresh  = num("foam.fresh")
	surfFull   = num("surf.full")
	rockBack   = num("rock.reflectance")
	shoreSteps = int(num("shore.march.steps"))
	shoreWiden = num("shore.march.widen")
)

// Shore is a square of ground under and beside the sea: titan's ground,
// sampled once onto a grid in the sea's own flat space, x east and y
// north of a place, in metres, with heights counted from the sea's mean
// level.
//
// Where the ground is below the sea it is the sea floor, and the waves
// over it are changed by how deep it is. Where it is above, it is land,
// and is drawn as rock.
type Shore struct {
	extent, step float64
	across       int
	// height is the ground above the sea's mean level, in metres. made
	// is how many of those metres titan generated, not measured.
	height, made []float64
	// gain is how much taller or shorter the waves stand over each
	// sample than in deep water, once breaking has cut them down. shoal
	// is how much taller they would stand if nothing broke. surf is how
	// far into the surf a sample is, from none at its outer edge to one
	// where the waves arrive twice too tall for the depth.
	gain, shoal, surf []float64
	// rms is the root-mean-square height of the waves arriving, in
	// metres.
	rms  float64
	top  float64 // the highest ground
	wall *wall
}

// wall is a straight, upright wall the sea reflects off.
type wall struct {
	at, seaward Vec
	returns     float64
}

// NewShore samples a square of titan's ground for a sea to meet. The
// square is extent metres across with its middle at a longitude and
// latitude in degrees, on a world of a radius in metres.
//
// Its samples are step metres apart, and titan is asked for that much
// detail: whatever the map behind it lacks at that size, titan generates,
// and Made says where. level is how high the sea's mean level stands in
// the map's own heights, the tide included.
//
// The square is flat. Over a few kilometres on the Earth that is wrong
// by less than a metre.
func NewShore(ground *titan.Ground, radius, longitude, latitude, level,
	extent, step float64) (*Shore, error) {
	if ground == nil || !(radius > 0) || !(extent > 0) || !(step > 0) ||
		extent/step < 2 {
		return nil, errors.New("oceantex: a shore has ground, a radius, " +
			"and an extent of at least two steps")
	}
	limits, err := readLimits()
	if err != nil {
		return nil, err
	}
	ratio := extent / step
	if !finite(ratio) || ratio > float64(limits.ShoreCells) ||
		!finite(radius) || !finite(longitude) || !finite(latitude) || !finite(level) {
		return nil, errors.New("oceantex: invalid or unbounded shore")
	}
	across := int(math.Ceil(ratio)) + 1
	if across > limits.ShoreCells/across {
		return nil, errors.New("oceantex: shore exceeds cell budget")
	}

	shore := &Shore{extent: extent, step: step, across: across,
		height: make([]float64, across*across),
		made:   make([]float64, across*across),
		top:    math.Inf(-1)}
	perDegree := radius * degrees
	for row := range across {
		for column := range across {
			x := float64(column)*step - extent/2
			y := float64(row)*step - extent/2
			at, err := mesh.NewPoint(
				longitude+x/(perDegree*math.Cos(latitude*degrees)),
				latitude+y/perDegree)
			if err != nil {
				return nil, err
			}
			sample, err := ground.At(at, step)
			if err != nil {
				return nil, err
			}
			here := row*across + column
			shore.height[here] = sample.Height() - level
			shore.made[here] = sample.Made()
			shore.top = math.Max(shore.top, shore.height[here])
		}
	}
	return shore, nil
}

// Wall tells the shore that the sea meets an upright wall along a
// straight line through a place, facing a direction along the sea, and
// how much of a wave's height the wall sends back: one for a sheer cliff
// in deep water, less for anything that breaks the wave up.
//
// The sea in front of the wall is then the waves arriving and the same
// waves mirrored in it, which is what a wave does at a wall: it is the
// method of images, and the reason the sea at a cliff's foot heaves
// twice as high as the sea a mile out. The shore's own ground need not
// be straight. Where it is not, the mirror is still this line, and the
// reflection is the less true the further the ground strays from it.
func (shore *Shore) Wall(at, seaward Vec, returns float64) error {
	seaward.Z = 0
	if seaward == (Vec{}) || !(returns >= 0 && returns <= 1) {
		return errors.New("oceantex: a wall faces somewhere along the " +
			"sea, and returns between none and all of a wave")
	}
	shore.wall = &wall{at, seaward.unit(), returns}
	return nil
}

// index is the grid's sample at or just below a place, and how far
// toward the next the place is. The place is held to the square.
func (shore *Shore) index(x, y float64) (column, row int, tx, ty float64) {
	gx := (x + shore.extent/2) / shore.step
	gy := (y + shore.extent/2) / shore.step
	last := float64(shore.across - 1)
	gx, gy = math.Min(math.Max(gx, 0), last), math.Min(math.Max(gy, 0), last)
	column = min(int(gx), shore.across-2)
	row = min(int(gy), shore.across-2)
	return column, row, gx - float64(column), gy - float64(row)
}

// read is a grid's value at a place, on the straight lines between its
// four nearest samples.
func (shore *Shore) read(grid []float64, x, y float64) float64 {
	column, row, tx, ty := shore.index(x, y)
	at := row*shore.across + column
	low := grid[at]*(1-tx) + grid[at+1]*tx
	high := grid[at+shore.across]*(1-tx) + grid[at+shore.across+1]*tx
	return low*(1-ty) + high*ty
}

// Holds is whether a place is inside the shore's square.
func (shore *Shore) Holds(x, y float64) bool {
	return math.Abs(x) <= shore.extent/2 && math.Abs(y) <= shore.extent/2
}

// Height is the ground's height above the sea's mean level at a place,
// in metres. Below nought it is sea floor.
func (shore *Shore) Height(x, y float64) float64 {
	return shore.read(shore.height, x, y)
}

// Made is how many of the ground's metres at a place titan generated.
func (shore *Shore) Made(x, y float64) float64 {
	return shore.read(shore.made, x, y)
}

// slope is how fast the ground rises east and north at a place.
func (shore *Shore) slope(x, y float64) (east, north float64) {
	reach := shore.step / 2
	east = (shore.Height(x+reach, y) - shore.Height(x-reach, y)) / shore.step
	north = (shore.Height(x, y+reach) - shore.Height(x, y-reach)) / shore.step
	return east, north
}

// meet works out, for a sea, what each sample of the shore does to the
// waves over it.
//
// Shoaling. A wave carries energy forward at its group speed. Coming
// into shallower water it slows, and the same energy arriving has to
// stand taller: its height grows by the root of the deep-water group
// speed over the local one. That is linear wave theory, the same that
// gives the dispersion relation, and it has no numbers of its own.
//
// Breaking. Waves cannot stand taller than the water is deep allows. In
// the surf their root-mean-square height is held to 0.42 of the depth
// [TG82]. Where shoaling asks for more than that, the waves are cut down
// to it. Which crests are white is worked out wave by wave, in foam.
//
// GLUE: one wave stands for the whole sea, the one at the spectrum's
// peak; every wave is shoaled by its factor. The waves' directions and
// lengths are left as they were in deep water, so there is no bending
// toward the shore and no bunching up.
func (shore *Shore) meet(sea Sea) {
	shore.gain = make([]float64, len(shore.height))
	shore.shoal = make([]float64, len(shore.height))
	shore.surf = make([]float64, len(shore.height))
	if !(sea.Wind > 0 && sea.Fetch > 0) {
		return
	}
	peak := sea.Peak()
	deep := Sea{Wind: sea.Wind, Fetch: sea.Fetch}
	_, outside := deep.turning(peak * peak / Gravity)
	arriving := sea.Height() / overRMS // root-mean-square height
	shore.rms = arriving
	for at, height := range shore.height {
		depth := -height
		if depth <= 0 {
			continue // land
		}
		here := Sea{Depth: depth}
		_, inside := here.turning(here.number(peak))
		gain := math.Sqrt(outside / inside)
		shore.shoal[at] = gain
		if most := saturated * depth; gain*arriving > most {
			shore.surf[at] = math.Min((gain*arriving/most-1)/surfFull, 1)
			gain = most / arriving
		}
		shore.gain[at] = gain
	}
}

// number is the wave number of a wave of an angular frequency in this
// sea's depth: the dispersion relation, omega^2 = g k tanh(k h), solved
// for k by Newton's method from the deep-water answer.
func (s Sea) number(omega float64) float64 {
	k := omega * omega / Gravity
	if s.Depth <= 0 {
		return k
	}
	for range 30 {
		tanh := math.Tanh(k * s.Depth)
		sech := 1 / math.Cosh(k*s.Depth)
		miss := Gravity*k*tanh - omega*omega
		rate := Gravity * (tanh + k*s.Depth*sech*sech)
		next := k - miss/rate
		if math.Abs(next-k) <= 1e-12*k {
			return next
		}
		k = next
	}
	return k
}

// mirror is a place on the sea reflected in the shore's wall, how much of
// a wave the wall returns, and whether there is a wall at all.
//
// A place behind the wall's line mirrors to the front of it, as one in
// front mirrors to behind. Behind a real wall there is no sea to ask
// about; but the ground need not follow the line, and where water lies
// a little behind it the sea must not show a seam along the line.
func (shore *Shore) mirror(x, y float64) (mx, my, returns float64,
	walled bool) {
	if shore.wall == nil {
		return 0, 0, 0, false
	}
	w := shore.wall
	out := (x-w.at.X)*w.seaward.X + (y-w.at.Y)*w.seaward.Y
	return x - 2*out*w.seaward.X, y - 2*out*w.seaward.Y, w.returns, true
}

// foam is how much of the sea at a place is white, from none to all,
// given how high the open sea's surface stands there before the shore
// has done anything to it.
//
// Outside the surf, a wave breaks when its height reaches 0.78 of the
// depth [Mc94]. The surface is shoaled as if nothing had broken, and a
// crest's height above the mean is taken for half its wave's height.
// Where that wave is too tall for the water under it, its crest is white.
//
// Inside the surf, where the waves as a whole are held to the depth
// [TG82], a wave that has broken stays broken: it runs in as a line of
// white water. So there every crest is white, the more so the higher it
// stands and the further into the surf it has come.
//
// GLUE: the half; a crest counting as fully white at half the waves'
// root-mean-square height; and foam being on crests and nowhere else.
// Real surf leaves foam behind it, and throws spray up a cliff. Neither
// is here.
func (shore *Shore) foam(x, y, open float64) float64 {
	depth := -shore.Height(x, y)
	if depth <= 0 || open <= 0 {
		return 0
	}
	wave := 2 * open * shore.read(shore.shoal, x, y)
	breaking := math.Max(wave/(crestLimit*depth)-1, 0) / surfFull
	broken := shore.read(shore.surf, x, y) * 2 * open / shore.rms
	return math.Min(math.Max(breaking, broken), 1)
}
