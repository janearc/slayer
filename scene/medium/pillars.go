package medium

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// stackColumns is how many columns Density places on the stack, so a Pillars
// of that many or fewer, built by a literal, allocates nothing per call.
const stackColumns = 8 // defined: the size Prepared's documentation states

// The shapes of the pillars are chosen for the picture, in choices.yaml.
//
// Placement: the columns' base is the width over the count times
// pillarsBaseDivisor; a column's height is pillarsHeightLeast of Height plus
// up to pillarsHeightRange more; the hashes of x, z and height are the seed
// plus pillarsSeedX, pillarsSeedZ and pillarsSeedHeight.
//
// The wall: a pillarsWall share of Height, ragged by pillarsWallRag of Height
// of noise read at pillarsWallScale over pillarsWallOctaves octaves, soft over
// pillarsWallEdge of Height.
//
// The columns: tapering by pillarsTaper; their noise read at
// pillarsNoiseScale over the base, from the seed plus pillarsSeedNoise, over
// pillarsOctaves octaves, building the column out by pillarsBuild of a base;
// the edge soft over pillarsEdge of a base.
//
// The filaments: noise stretched
// pillarsStretch times along y, read at pillarsStreak over the base from the
// seed plus pillarsSeedStreak, over pillarsStreakOctaves octaves, thinning
// where it is below pillarsStreakThreshold.
var (
	fog                  = choice("pillars-fog")
	pillarsBaseDivisor   = choice("pillars-base-divisor")
	pillarsHeightLeast   = choice("pillars-height-least")
	pillarsHeightRange   = choice("pillars-height-range")
	pillarsSeedX         = choiceSeed("pillars-seed-x")
	pillarsSeedZ         = choiceSeed("pillars-seed-z")
	pillarsSeedHeight    = choiceSeed("pillars-seed-height")
	pillarsWall          = choice("pillars-wall")
	pillarsWallRag       = choice("pillars-wall-rag")
	pillarsWallScale     = choice("pillars-wall-scale")
	pillarsWallOctaves   = choiceCount("pillars-wall-octaves")
	pillarsWallEdge      = choice("pillars-wall-edge")
	pillarsTaper         = choice("pillars-taper")
	pillarsNoiseScale    = choice("pillars-noise-scale")
	pillarsSeedNoise     = choiceSeed("pillars-seed-noise")
	pillarsOctaves       = choiceCount("pillars-octaves")
	pillarsBuild         = choice("pillars-build")
	pillarsEdge          = choice("pillars-edge")
	pillarsStreak        = choice("pillars-streak")
	pillarsStretch       = choice("pillars-stretch")
	pillarsSeedStreak    = choiceSeed("pillars-seed-streak")
	pillarsStreakOctaves = choiceCount("pillars-streak-octaves")
)

// Pillars is a nebula's pillars of gas, as the eagle's: Count columns standing
// on a wall of gas along the bottom, up y, none taller than Height, spread
// across Width, each tapering as it rises, rounded at the tip, and eaten into
// by fractal noise so the edges are ragged.
//
// The columns and the wall are Thick; a thin fog fills the rest of the box.
// Seed makes a nebula its own.
//
// Filaments is a dressing: fine streaks drawn up the columns by a finer noise
// stretched along them, as the pictures show and the gas at this scale does
// not, zero none and one all streaks.
type Pillars struct {
	Count         int
	Height, Width float64
	Thick         float64
	Seed          uint64
	Filaments     float64

	// held is the columns, set by Prepared.
	held     []column
	prepared bool
}

// column is one of the pillars: where it stands, how tall, how wide
// at the foot.
type column struct {
	x, z, height, base float64
}

// columns is the pillars, each placed, sized and tapered by hash; a count of
// zero or less is none.
func (pillars Pillars) columns() []column { return pillars.appendColumns(nil) }

// appendColumns is columns, appended to out, so that a caller with a buffer
// of room for them does not allocate.
func (pillars Pillars) appendColumns(out []column) []column {
	if pillars.Count <= nought {
		return out
	}
	half := pillars.Width / double
	base := pillars.Width / (float64(pillars.Count) * pillarsBaseDivisor)
	for index := range pillars.Count {
		hashed := int64(index)
		out = append(out, column{
			x: -half + pillars.Width*(float64(index)+halfway)/
				float64(pillars.Count) +
				(noise.Hash(pillars.Seed+pillarsSeedX, hashed)-halfway)*base,
			z: (noise.Hash(pillars.Seed+pillarsSeedZ, hashed) - halfway) *
				double * base,
			height: pillars.Height * (pillarsHeightLeast +
				pillarsHeightRange*
					noise.Hash(pillars.Seed+pillarsSeedHeight, hashed)),
			base: base,
		})
	}
	return out
}

// Prepared is the same pillars with their columns placed once, so that
// Density, asked at every step of a march, does not place them again. A
// Pillars built by a literal gives the same answers without it: it places
// its columns at each call, in a buffer on the stack when Count is eight or
// fewer.
//
// The columns depend on Count, Height, Width and Seed only; a prepared
// value does not see a later edit to any of them, and does not depend on
// Thick or Filaments, which may change freely. A prepared value is not
// written to again, so render workers may share it.
func (pillars Pillars) Prepared() Pillars {
	pillars.held, pillars.prepared = pillars.columns(), true
	return pillars
}

// table is the columns, those Prepared held or else placed into buf.
func (pillars Pillars) table(buf []column) []column {
	if pillars.prepared {
		return pillars.held
	}
	return pillars.appendColumns(buf)
}

// Tips is where the pillars end: the top of each column, where the
// young stars are found.
func (pillars Pillars) Tips() []maths.Vec {
	var tips []maths.Vec
	var stack [stackColumns]column
	for _, one := range pillars.table(stack[:nought]) {
		tips = append(tips, maths.V(one.x, one.height, one.z))
	}
	return tips
}

// Bounds is the box the pillars stand in, as deep as it is wide.
func (pillars Pillars) Bounds() maths.Box {
	half := pillars.Width / double
	return corners(maths.V(-half, nought, -half),
		maths.V(half, pillars.Height, half))
}

// wall is the gas along the bottom, a share of the height, ragged on top: how
// much of the pillars' Thick it is at the point, or zero above it.
func (pillars Pillars) wall(point maths.Vec) float64 {
	ragged := pillarsWallRag * pillars.Height * noise.Fractal(pillars.Seed,
		point.X/pillars.Height*pillarsWallScale, nought,
		point.Z/pillars.Height*pillarsWallScale, pillarsWallOctaves)
	top := pillarsWall*pillars.Height + ragged
	if !(point.Y < top) {
		return nought
	}
	edge := maths.Smooth(top-pillarsWallEdge*pillars.Height, top, point.Y)
	return unity - edge
}

// rise is one column's share of Thick at the point: tapering as it rises,
// domed at the tip, eaten into by noise, and streaked when Filaments says so.
func (pillars Pillars) rise(one column, index int, point maths.Vec) float64 {
	height, base := one.height, one.base
	if point.Y > height {
		return nought
	}
	radius := base * (unity - pillarsTaper*point.Y/height)
	// the tip, a dome a radius deep
	if from := height - radius; point.Y > from {
		dome := (point.Y - from) / radius
		radius *= math.Sqrt(math.Max(nought, unity-dome*dome))
	}
	across := math.Hypot(point.X-one.x, point.Z-one.z)
	// eaten into by noise: the edge wanders, the tip is knobbed
	frequency := pillarsNoiseScale / base
	seed := pillars.Seed + pillarsSeedNoise + uint64(index)
	x, y, z := point.X*frequency, point.Y*frequency, point.Z*frequency
	grain := noise.Fractal(seed, x, y, z, pillarsOctaves)
	// built out by the noise, by up to about a whole base and in
	// by a seventh, half a base out on the whole, so the columns
	// are fat, solid and knobbed rather than wispy
	across += (grain - halfway) * pillarsBuild * base
	got := unity - maths.Smooth(radius-pillarsEdge*base, radius, across)
	if pillars.Filaments > nought && got > nought {
		// streaks up the column: a noise stretched along y, ragged at
		// every scale, thinning the gas where it is low
		streak := pillarsStreak / base
		fine := noise.Fractal(seed+pillarsSeedStreak, point.X*streak,
			point.Y*streak/pillarsStretch, point.Z*streak,
			pillarsStreakOctaves)
		got *= unity - pillars.Filaments*math.Min(unity,
			math.Max(nought, halfway-fine))
	}
	return got
}

// Density is the wall, the columns and the fog, whichever is most.
func (pillars Pillars) Density(point maths.Vec) float64 {
	if pillars.Height <= nought || pillars.Width <= nought ||
		pillars.Count <= nought {
		return nought
	}
	half := pillars.Width / double
	if point.Y < nought || point.Y > pillars.Height ||
		math.Abs(point.X) > half || math.Abs(point.Z) > half {
		return nought
	}
	most := math.Max(fog, pillars.wall(point))
	// the columns, each its own
	var stack [stackColumns]column
	for index, one := range pillars.table(stack[:nought]) {
		most = math.Max(most, pillars.rise(one, index, point))
	}
	return pillars.Thick * most
}
