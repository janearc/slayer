package medium

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// the numbers of a wake's gas. wakeNone and wakeOne are zero and one, and
// wakePlus is the sign of the infinity a path nowhere near is. wakeHalf is
// the half that maps noise from minus one to one onto nought to one.
// wakeSquare is the power of the ridge's bell, a gaussian.
//
// the rest are picked, for the picture: wakeFade is where the gas begins to
// thin toward the edge, as a share of the reach; wakeSwept is where the
// channel's clearing begins, as a share of its clear; wakeRidge is the width
// of the banks, as a share of the clear; wakeBank is how many times the open
// gas the banks hold; wakeLift is how many times the cubed noise is lifted,
// so that its mean gas stays near the thick.
const (
	wakeNone   = 0.0
	wakeOne    = 1.0
	wakePlus   = 1
	wakeHalf   = 0.5
	wakeSquare = 2
	wakeFade   = 0.45
	wakeSwept  = 0.55
	wakeRidge  = 0.3
	wakeBank   = 2.2
	wakeLift   = 3
)

// Path is a circle a band follows through a wake: its plane, in a subspace
// of the wake's own space, and its radius in the subspace's own units.
type Path struct {
	space  maths.Subspace
	radius float64
}

// NewPath is the circle of radius radius about the origin of space, in its
// x and z plane. It refuses a radius that is not a positive finite number,
// a space with an origin that is not finite, and one with a scale that is
// negative or not finite; a scale of zero is one, as everywhere else.
func NewPath(space maths.Subspace, radius float64) (*Path, error) {
	origin := space.Origin
	if !weatherFinite(radius) || radius <= wakeNone ||
		!weatherFinite(origin.X+origin.Y+origin.Z) ||
		!weatherFinite(space.Scale) || space.Scale < wakeNone {
		return nil, fmt.Errorf("medium: a path of radius %v in a space of "+
			"origin %v and scale %v, want a positive finite radius and a "+
			"finite origin and scale of at least zero", radius, origin,
			space.Scale)
	}
	return &Path{space: space, radius: radius}, nil
}

// Space is the subspace whose x and z plane the path lies in.
func (path Path) Space() maths.Subspace { return path.space }

// Radius is the path's radius, in the subspace's own units.
func (path Path) Radius() float64 { return path.radius }

// Grain is the noise that makes a wake's gas ragged: fractal noise of a seed,
// read at a point over a scale, summed over octaves.
type Grain struct {
	seed    uint64
	scale   float64
	octaves int
}

// NewGrain is the noise of seed, its largest features scale across, over
// octaves layers. It refuses a scale that is not a positive finite number,
// and fewer than one octave.
func NewGrain(seed uint64, scale float64, octaves int) (*Grain, error) {
	if !weatherFinite(scale) || scale <= wakeNone || octaves < 1 {
		return nil, fmt.Errorf("medium: a grain of scale %v and %d octaves, "+
			"want a positive finite scale and at least one octave", scale,
			octaves)
	}
	return &Grain{seed: seed, scale: scale, octaves: octaves}, nil
}

// Seed is the noise's seed.
func (grain Grain) Seed() uint64 { return grain.seed }

// Scale is how many units across the noise's largest features are.
func (grain Grain) Scale() float64 { return grain.scale }

// Octaves is how many layers the noise has.
func (grain Grain) Octaves() int { return grain.octaves }

// Wake is a nebula with the channel something has swept through it: ragged
// gas of density about Thick everywhere in a ball of Reach, thinning to
// nothing toward its edge, almost none within Clear of the paths, and what
// was swept piled in banks along the channel's edges, as a moon clears a gap
// in saturn's rings and banks it up.
type Wake struct {
	// reach is the ball's radius, clear the channel's, and thick the gas's
	// density away from both
	reach, clear, thick float64
	grain               Grain
	paths               []Path
	// the shape of the gas, from the constants above
	fade, swept, ridge, bank, lift float64
}

// NewWake is a wake of a ball of radius reach, a channel of radius clear
// about each of paths, and gas of density thick, ragged as grain is. It
// refuses a reach or clear that is not a positive finite number, a thick
// that is not a finite number of at least zero, no grain, and no paths or a
// nil one.
func NewWake(reach, clear, thick float64, grain *Grain,
	paths []*Path) (*Wake, error) {
	bad := !weatherFinite(reach) || reach <= wakeNone ||
		!weatherFinite(clear) || clear <= wakeNone ||
		!weatherFinite(thick) || thick < wakeNone ||
		grain == nil || len(paths) == 0
	var held []Path
	for _, path := range paths {
		if path == nil {
			bad = true
			break
		}
		held = append(held, *path)
	}
	if bad {
		return nil, fmt.Errorf("medium: a wake of reach %v, clear %v and "+
			"thick %v with grain %v and %d paths, want positive finite "+
			"reach and clear, a finite thick of at least zero, a grain and "+
			"at least one path, none nil", reach, clear, thick, grain,
			len(paths))
	}
	return &Wake{reach: reach, clear: clear, thick: thick, grain: *grain,
		paths: held, fade: wakeFade, swept: wakeSwept, ridge: wakeRidge,
		bank: wakeBank, lift: wakeLift}, nil
}

// Reach is the radius of the wake's ball.
func (n *Wake) Reach() float64 { return n.reach }

// Clear is the radius of the channel about each path.
func (n *Wake) Clear() float64 { return n.clear }

// Thick is the gas's density away from the edge and the channel, before the
// noise.
func (n *Wake) Thick() float64 { return n.thick }

// Grain is the noise the gas is ragged by.
func (n *Wake) Grain() Grain { return n.grain }

// Paths are the paths of the channel, a copy of the wake's own.
func (n *Wake) Paths() []*Path {
	var paths []*Path
	for k := range n.paths {
		paths = append(paths, &n.paths[k])
	}
	return paths
}

// Fade is where the gas begins to thin toward the edge, as a share of the
// reach.
func (n *Wake) Fade() float64 { return n.fade }

// Swept is where the channel's clearing begins, as a share of its clear.
func (n *Wake) Swept() float64 { return n.swept }

// Ridge is the width of the banks along the channel's edge, as a share of
// the clear, and how many times the open gas they hold.
func (n *Wake) Ridge() (width, bank float64) { return n.ridge, n.bank }

// Lift is how many times the cubed noise is lifted.
func (n *Wake) Lift() float64 { return n.lift }

// Bounds is the cube that holds the nebula, its reach each way from the
// middle.
func (n *Wake) Bounds() maths.Box {
	r := n.reach
	return maths.Box{Min: maths.V(-r, -r, -r), Max: maths.V(r, r, r)}
}

// Span is the nebula's ball, so a march starts on the gas.
func (n *Wake) Span(r maths.Ray, near, far float64) (float64, float64, bool) {
	b := r.At.Dot(r.Dir)
	c := r.At.Dot(r.At) - n.reach*n.reach
	disc := b*b - c
	if disc < wakeNone {
		return wakeNone, wakeNone, false
	}
	root := math.Sqrt(disc)
	enter, leave := math.Max(-b-root, near), math.Min(-b+root, far)
	return enter, leave, leave > enter
}

// fromPaths is how far p is from the nearest path, the circle at its radius
// in its own plane.
func (n *Wake) fromPaths(p maths.Vec) float64 {
	nearest := math.Inf(wakePlus)
	for _, one := range n.paths {
		q := one.space.In(p)
		away := math.Hypot(math.Hypot(q.X, q.Z)-one.radius, q.Y)
		nearest = math.Min(nearest, away)
	}
	return nearest
}

// Density is the nebula's gas at p: ragged, thin to its edge, and cleared
// round the paths.
func (n *Wake) Density(p maths.Vec) float64 {
	if p.Len() > n.reach {
		return wakeNone
	}
	scale := n.grain.scale
	// ragged at every size, as a nebula is
	rag := wakeHalf + wakeHalf*noise.Fractal(n.grain.seed, p.X/scale,
		p.Y/scale, p.Z/scale, n.grain.octaves)
	// cubed, so most of it is thin and the wisps stand out against the dark
	base := n.thick * rag * rag * rag * n.lift
	// thinning to nothing toward its edge, so it has no rim to show
	base *= wakeOne - maths.Smooth(n.fade*n.reach, n.reach, p.Len())
	// how far from the nearest path
	d := n.fromPaths(p)
	swept := maths.Smooth(n.swept*n.clear, n.clear, d)
	ridge := math.Exp(-math.Pow((d-n.clear)/(n.ridge*n.clear), wakeSquare))
	return base * (swept + n.bank*ridge)
}
