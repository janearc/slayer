package medium

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// the numbers of a wake's gas that are true by definition: wakePlus is the
// sign of the infinity a path nowhere near is, wakeSquare is the power of the
// ridge's bell, a gaussian, and wakeOctave is the fewest octaves a grain has.
//
// the rest are chosen, for the picture, in choices.yaml: wake-fade is where
// the gas begins to thin toward the edge, as a share of the reach; wake-swept
// is where the channel's clearing begins, as a share of its clear; wake-ridge
// is the width of the banks, as a share of the clear; wake-bank is how many
// times the open gas the banks hold; wake-lift is how many times the cubed
// noise is lifted, so that its mean gas stays near the thick.
const (
	wakePlus   = 1 // defined: the sign of positive infinity
	wakeSquare = 2 // defined: a gaussian's power
	wakeOctave = 1 // defined: the fewest octaves a grain has
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
	if !weatherFinite(radius) || radius <= nought ||
		!weatherFinite(origin.X+origin.Y+origin.Z) ||
		!weatherFinite(space.Scale) || space.Scale < nought {
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
	if !weatherFinite(scale) || scale <= nought || octaves < wakeOctave {
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

// Wake is a cloud of gas with the channel something has swept through it:
// ragged gas of density about Thick everywhere in a ball of Reach, thinning to
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
	bad := !weatherFinite(reach) || reach <= nought ||
		!weatherFinite(clear) || clear <= nought ||
		!weatherFinite(thick) || thick < nought ||
		grain == nil || len(paths) == nought
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
		paths: held, fade: chosen.wakeFade, swept: chosen.wakeSwept,
		ridge: chosen.wakeRidge, bank: chosen.wakeBank,
		lift: chosen.wakeLift}, nil
}

// Reach is the radius of the wake's ball.
func (wake *Wake) Reach() float64 { return wake.reach }

// Clear is the radius of the channel about each path.
func (wake *Wake) Clear() float64 { return wake.clear }

// Thick is the gas's density away from the edge and the channel, before the
// noise.
func (wake *Wake) Thick() float64 { return wake.thick }

// Grain is the noise the gas is ragged by.
func (wake *Wake) Grain() Grain { return wake.grain }

// Paths are the paths of the channel, each a copy: changing one does not
// change the wake.
func (wake *Wake) Paths() []*Path {
	paths := make([]*Path, nought, len(wake.paths))
	for _, held := range wake.paths {
		own := held
		paths = append(paths, &own)
	}
	return paths
}

// Fade is where the gas begins to thin toward the edge, as a share of the
// reach.
func (wake *Wake) Fade() float64 { return wake.fade }

// Swept is where the channel's clearing begins, as a share of its clear.
func (wake *Wake) Swept() float64 { return wake.swept }

// Ridge is the width of the banks along the channel's edge, as a share of
// the clear, and how many times the open gas they hold.
func (wake *Wake) Ridge() (width, bank float64) { return wake.ridge, wake.bank }

// Lift is how many times the cubed noise is lifted.
func (wake *Wake) Lift() float64 { return wake.lift }

// Bounds is the cube that holds the wake, its reach each way from the
// middle.
func (wake *Wake) Bounds() maths.Box {
	return cube(wake.reach)
}

// Span is the wake's ball, so a march starts on the gas.
func (wake *Wake) Span(ray maths.Ray, near, far float64) (float64, float64,
	bool) {
	return sphere(wake.reach, ray, near, far)
}

// fromPaths is how far the point is from the nearest path, the circle at its
// radius in its own plane.
func (wake *Wake) fromPaths(point maths.Vec) float64 {
	nearest := math.Inf(wakePlus)
	for _, one := range wake.paths {
		local := one.space.In(point)
		away := math.Hypot(math.Hypot(local.X, local.Z)-one.radius, local.Y)
		nearest = math.Min(nearest, away)
	}
	return nearest
}

// Density is the wake's gas at the point: ragged, thin to its edge, and
// cleared round the paths. It is never below zero.
func (wake *Wake) Density(point maths.Vec) float64 {
	if point.Len() > wake.reach {
		return nought
	}
	scale := wake.grain.scale
	// ragged at every size. Gradient noise can pass minus one, which would
	// take the raggedness below zero, and its cube with it, so it is kept
	// at zero
	rag := math.Max(nought, halfway+halfway*noise.Fractal(wake.grain.seed,
		point.X/scale, point.Y/scale, point.Z/scale, wake.grain.octaves))
	// cubed, so most of it is thin and the wisps stand out against the dark
	base := wake.thick * rag * rag * rag * wake.lift
	// thinning to nothing toward its edge, so it has no rim to show
	base *= unity - maths.Smooth(wake.fade*wake.reach, wake.reach, point.Len())
	// how far from the nearest path
	distance := wake.fromPaths(point)
	swept := maths.Smooth(wake.swept*wake.clear, wake.clear, distance)
	ridge := math.Exp(-math.Pow((distance-wake.clear)/(wake.ridge*wake.clear),
		wakeSquare))
	return base * (swept + wake.bank*ridge)
}
