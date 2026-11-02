package occultation

import "math"

// Baked is a profile as a table a gpu reads the way it reads a texture: the
// light kept straight through each bin, at level zero, and the mean of each two
// from the level below at every level above it, until one bin is left.
//
// a ray reads the level whose bins are as wide as it is, between the two levels
// either side, so stretches finer than it are drawn as their mean light. kept
// in float32, as the kernel keeps it, so the cpu reading it draws what the gpu
// draws.
type Baked struct {
	inner, step float32
	levels      [][]float32
}

// Baked is the profile baked.
func (profile Profile) Baked() Baked {
	level := make([]float32, len(profile.depths))
	for k, depth := range profile.depths {
		level[k] = float32(math.Exp(-depth))
	}
	levels := [][]float32{level}
	for len(level) > 1 {
		up := make([]float32, (len(level)+1)/2)
		for k := range up {
			pair := level[min(2*k+1, len(level)-1)]
			up[k] = (level[2*k] + pair) / 2
		}
		levels, level = append(levels, up), up
	}
	return Baked{inner: float32(profile.inner), step: float32(profile.step),
		levels: levels}
}

// Levels is the table's levels, finest first, each the light kept in
// its bins: what the kernel is handed.
func (baked Baked) Levels() [][]float32 { return baked.levels }

// Inner is the radius in metres the table starts at.
func (baked Baked) Inner() float32 { return baked.inner }

// Step is the width in metres of the finest level's bins.
func (baked Baked) Step() float32 { return baked.step }

// Outer is the radius in metres the table ends at.
func (baked Baked) Outer() float32 {
	return baked.inner + baked.step*float32(len(baked.levels[0]))
}

// Seen is the depth across so many metres of radius about a radius, as light
// sees it, read from the table: the level as wide as the stretch, between the
// two levels either side, each read straight between the middles of the two
// bins either side of the radius.
//
// outside the table, it is zero.
func (baked Baked) Seen(radius, across float32) float32 {
	if radius < baked.inner || radius >= baked.Outer() {
		return 0
	}
	level := float32(0)
	if across > baked.step {
		level = float32(math.Log2(float64(across / baked.step)))
	}
	top := float32(len(baked.levels) - 1)
	level = min(level, top)
	below := int(level)
	above := min(below+1, int(top))
	toward := level - float32(below)
	kept := baked.read(below, radius)*(1-toward) +
		baked.read(above, radius)*toward
	return float32(-math.Log(float64(max(kept, 1e-30))))
}

// read is a level's light kept at a radius, straight between the
// middles of its bins.
func (baked Baked) read(level int, radius float32) float32 {
	bins := baked.levels[level]
	width := baked.step * float32(int(1)<<level)
	along := (radius-baked.inner)/width - 0.5
	k := int(math.Floor(float64(along)))
	toward := along - float32(k)
	low := bins[max(0, min(k, len(bins)-1))]
	high := bins[max(0, min(k+1, len(bins)-1))]
	return low*(1-toward) + high*toward
}
