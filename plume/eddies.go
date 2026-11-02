package plume

import (
	"math"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/medium"
)

// the eddies' numbers, all picked by looking. that a plume's largest
// eddies are about its own width, and rise with it, is the turbulent
// plume as recalled, not read.
const (
	// eddy is how big the largest eddies are, as a share of the
	// column's radius where they are
	eddy = 0.6
	// sway is how far the eddies push the soot aside, as a share of
	// the column's radius
	sway = 0.35
	// clump is how much thicker or thinner the eddies make the soot
	// about its mean: one plus clump times the noise
	clump = 0.7
	// octaves is how many sizes of eddy, each half the one before
	octaves = 4
)

// smokeSeeds moves the smoke's noise off the seeds a flame of the same
// fire takes for its own (flame's body reads its seed and the few after
// it), so the smoke over a fire does not roll with the flame under it:
// the word plume's bytes, picked.
const smokeSeeds = 0x706c756d65

// Tick is the column at a time, its eddies risen with it.
func (column *Column) Tick(at time.Duration) medium.Medium {
	ticked := *column
	ticked.at = at
	return &ticked
}

// rolled is where a point's soot comes from once the eddies have moved it, and
// how much thicker the eddies make it there.
//
// the eddies are noise as big as eddy times the radius at the point's height,
// read up the column by phase: the air at a height and a time has the eddies
// the same air had lower down, earlier.
//
// they push the soot across and thicken or thin it about its mean, which the
// noise, with its mean of zero, leaves where the plume says on average.
func (column *Column) rolled(p maths.Vec) (maths.Vec, float64) {
	radius := column.Radius(column.above(p.Y))
	size := eddy * radius
	x, y, z := p.X/size, column.phase(p.Y)/size, p.Z/size
	seed := column.fire.Seed ^ smokeSeeds
	push := sway * radius
	moved := maths.V(p.X+push*noise.Fractal(seed, x, y, z, octaves), p.Y,
		p.Z+push*noise.Fractal(seed+1, x, y, z, octaves))
	thicker := 1 + clump*noise.Fractal(seed+2, x, y, z, octaves)
	return moved, max(0, thicker)
}

// phase is where up the column the eddies at a height are, at the
// column's time, in metres: how long the air took to rise there from
// where the plume starts, less the time, at the plume's speed where it
// starts.
//
// air rising with the plume keeps its phase, so its eddies rise with
// it, slowing as it slows, and never shear with time.
//
// by mccaffrey's speed, the heat to the third over the height to the
// third times 1.12, the time to rise from where the plume starts is
// three quarters of the difference of the heights to the four thirds,
// over 1.12 times the heat to the third.
//
// below where the plume starts the air keeps the speed there.
func (column *Column) phase(height float64) float64 {
	start := column.start
	speed := column.Speed(start)
	rise := (height - start) / speed
	if height > start {
		rise = 0.75 * (math.Pow(height, 4.0/3) -
			math.Pow(start, 4.0/3)) / (1.12 * math.Cbrt(column.fire.Heat))
	}
	return speed * (rise - column.at.Seconds())
}
