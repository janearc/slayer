package shortcut

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// defaultResolution is the grid's cells a side when a set says baked
// and no number: ninety-six, where a wisp of smoke stays within an eye
// of exact. the first tracer said forty-eight did not; nothing here shows
// it, and at sixteen pixels square forty-eight is within the budget too.
const defaultResolution = 96

// largest is the most cells a side a bake may have: 256 is 134 MB of
// grid. past it a bake asks for gigabytes, and past two million cells a
// side the count of them overflows.
const largest = 256

// Baked is a medium whose density was computed once into a grid over its bounds
// and is read back by blending the eight cells round a point, instead of
// evaluating its noise at every sample.
//
// Outside the bounds it is zero, as most media are; Plume, Jet and Tail reach
// past theirs, and a bake cuts them there. It spans as the medium spans and
// ticks as it ticks, baking again at each new time. a point of no number, or a
// box flat in any way, is zero too.
type Baked struct {
	Of         medium.Medium
	Resolution int
	box        maths.Box
	cells      []float64
}

// Bake is a medium baked at a resolution. one past largest is refused
// by name, since the grid would not fit; Parse refuses it first.
func Bake(m medium.Medium, resolution int) *Baked {
	if resolution < 2 {
		resolution = defaultResolution
	}
	if resolution > largest {
		panic(fmt.Sprintf("shortcut: a bake of %d cells a side is past "+
			"%d, more memory than a picture is worth", resolution, largest))
	}
	b := &Baked{Of: m, Resolution: resolution, box: m.Bounds()}
	n := resolution
	b.cells = make([]float64, n*n*n)
	size := b.box.Max.Sub(b.box.Min)
	for k := 0; k < n; k++ {
		for j := 0; j < n; j++ {
			for i := 0; i < n; i++ {
				p := b.box.Min.Add(maths.V(
					size.X*float64(i)/float64(n-1),
					size.Y*float64(j)/float64(n-1),
					size.Z*float64(k)/float64(n-1)))
				b.cells[(k*n+j)*n+i] = m.Density(p)
			}
		}
	}
	return b
}

// Density is the baked density at p: the eight cells round it blended.
func (b *Baked) Density(p maths.Vec) float64 {
	n := b.Resolution
	size := b.box.Max.Sub(b.box.Min)
	fx := (p.X - b.box.Min.X) / size.X * float64(n-1)
	fy := (p.Y - b.box.Min.Y) / size.Y * float64(n-1)
	fz := (p.Z - b.box.Min.Z) / size.Z * float64(n-1)
	// written so that no number, never within, is outside
	if !(fx >= 0 && fy >= 0 && fz >= 0 && fx <= float64(n-1) &&
		fy <= float64(n-1) && fz <= float64(n-1)) {
		return 0
	}
	i, j, k := int(fx), int(fy), int(fz)
	if i >= n-1 {
		i = n - 2
	}
	if j >= n-1 {
		j = n - 2
	}
	if k >= n-1 {
		k = n - 2
	}
	tx, ty, tz := fx-float64(i), fy-float64(j), fz-float64(k)
	at := func(di, dj, dk int) float64 { return b.cells[((k+dk)*n+(j+dj))*n+(i+di)] }
	lerp := func(a, c, t float64) float64 { return a + (c-a)*t }
	x00 := lerp(at(0, 0, 0), at(1, 0, 0), tx)
	x10 := lerp(at(0, 1, 0), at(1, 1, 0), tx)
	x01 := lerp(at(0, 0, 1), at(1, 0, 1), tx)
	x11 := lerp(at(0, 1, 1), at(1, 1, 1), tx)
	return math.Max(0, lerp(lerp(x00, x10, ty), lerp(x01, x11, ty), tz))
}

// Bounds is the medium's.
func (b *Baked) Bounds() maths.Box { return b.box }

// Span is the medium's own span when it has one, else its box's.
func (b *Baked) Span(r maths.Ray, near, far float64) (enter, leave float64, ok bool) {
	if sp, has := b.Of.(medium.Spanner); has {
		return sp.Span(r, near, far)
	}
	return b.box.Hit(r, near, far)
}

// Tick is the medium at a time, baked again: a moving medium costs its
// bake each frame, which is still once a frame and not once a sample.
func (b *Baked) Tick(t time.Duration) medium.Medium {
	if moving, ok := b.Of.(medium.Timed); ok {
		return Bake(moving.Tick(t), b.Resolution)
	}
	return b
}

// Apply is the world with the plan's shortcuts taken where they can
// be today: every volume whose material's name the plan bakes is baked
// at the plan's resolution. The other shortcuts leave the world as it
// is until render/trace takes them.
//
// a gas volume baked loses its gas: the tracer no longer sees Gas() and
// takes the plain density's way, a different picture (0.47 apart on a
// hazy puff at 24). kept as it was, as ruled.
//
// The world handed back is a copy; the one given is untouched, so the
// exact render is always to hand.
func Apply(w *thing.World, p Plan) *thing.World {
	out := *w
	out.Volumes = make([]*thing.Volume, 0, len(w.Volumes))
	for _, v := range w.Volumes {
		name := ""
		if v.Material != nil {
			name = v.Material.Name
		}
		s := p.ForPart(name)
		if !s.Baked || v.Medium == nil {
			out.Volumes = append(out.Volumes, v)
			continue
		}
		baked := *v
		baked.Medium = Bake(v.Medium, s.Resolution)
		out.Volumes = append(out.Volumes, &baked)
	}
	return &out
}
