package trace

import (
	"fmt"
	"math"
	"time"

	kerr "github.com/janearc/celestial/hole"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

// field is a field placed in the world at a time.
type field struct {
	f     *thing.Field
	at    maths.Subspace
	scale float64
	hole  kerr.Hole
}

// placeFields is every field, where it is at time t.
func placeFields(w *thing.World, t time.Duration) []field {
	fs := make([]field, 0, len(w.Fields))
	for _, f := range w.Fields {
		if f.Reach <= 0 || f.Mass <= 0 {
			continue
		}
		if !(math.Abs(f.Spin) < f.Mass) {
			// at or past its mass's spin a hole has no horizon,
			// and the march round it means nothing
			panic(fmt.Sprintf("trace: a field of mass %v spins at %v: "+
				"a hole spins under its mass", f.Mass, f.Spin))
		}
		hole, err := kerr.NewHole(f.Mass, f.Spin)
		if err != nil {
			panic(fmt.Sprintf("trace: %v", err))
		}
		at := f.At(t)
		s := at.Scale
		if s == 0 {
			s = 1
		}
		fs = append(fs, field{f, at, s, *hole})
	}
	return fs
}

// bends is how many steps a ray may take inside a field before it is
// given up as stuck and taken as captured: a ray near the photon
// sphere winds many times before it leaves, and must be let.
const bends = 40000

// loose is the radius, in masses, past which a ray heading outward is
// let go straight. the bending left beyond it is not small.
//
// measured against the exact schwarzschild integral, a ray at an impact
// of 20 masses bends 7.0 degrees here where 13.5 is true, and one at 39.9
// bends 0.24 where 6.2 is. kept as it was, as ruled.
const loose = 12

// atRest is the four-velocity of an eye at rest far from the hole,
// whose energy is the conserved one.
var atRest = kerr.NewVec4(1, 0, 0, 0)

// core is the radius, in masses, inside which a ray is taken as come
// from the hole's middle: black, as one hovering at the horizon is.
const core = 0.2

// shiftTop is the most a shift's gain may be, so one hot pixel does not
// bloom the frame.
const shiftTop = 50

// in is a point of the world in the hole's space.
func (f field) in(world maths.Vec) kerr.Vec3 {
	return toHole(f.f.InHole(f.at.In(world)))
}

// entering is the nearest field the ray will enter, and how far along it
// enters: zero when the ray starts inside one. A ray already past a field's
// reach, or missing it, does not enter it.
func entering(fs []field, r maths.Ray) (field, float64, bool) {
	best, at, found := field{}, nowhere, false
	for _, f := range fs {
		local := f.at.RayIn(r)
		reach := f.f.Reach
		// the reach sphere about the field's zero, in its own units
		b := local.At.Dot(local.Dir)
		c := local.At.Dot(local.At) - reach*reach
		if c < 0 {
			// inside already
			return f, 0, true
		}
		disc := b*b - c
		if disc < 0 {
			continue
		}
		t := -b - math.Sqrt(disc)
		if t <= 1e-9 {
			continue
		}
		if d := t * f.scale; d < at {
			best, at, found = f, d, true
		}
	}
	return best, at, found
}

// eye is the eye's four-velocity at a point inside the field: the one
// the field was told, or an eye held still there.
func (f field) eye(x kerr.Vec3) kerr.Vec4 {
	if f.f.EyeTime <= 0 {
		return f.hole.Static(x)
	}
	v := f.f.InHole(f.f.Eye)
	return kerr.NewVec4(f.f.EyeTime, v[0], v[1], v[2])
}

// emitter is a volume's four-velocity at a point of the world, in the
// hole's space: still, or its beam's flow. A round beam goes round at
// the hole's own circular rate, celestial's Circular: inside a field
// the hole's rate wins over the beam's Speed and At.
//
// A beam along its axis flows as Beam.Flow says, away from its origin,
// as a jet's.
func (f field) emitter(v volume, world maths.Vec) kerr.Vec4 {
	x := f.in(world)
	b := v.v.Beam
	if b == nil || b.Speed <= 0 || b.Speed >= 1 {
		return f.hole.Static(x)
	}
	if b.Round {
		return f.hole.Circular(x)
	}
	flow, speed := b.Flow(v.at.In(world))
	dir := f.f.InHole(f.at.Orientation.In(v.at.Orientation.Out(flow)).Unit())
	tetrad := axes(f.hole.FrameOf(x, f.hole.Static(x)))
	gamma := 1 / math.Sqrt(1-speed*speed)
	var u [4]float64
	for k := range 4 {
		u[k] = gamma * (tetrad[0][k] + speed*(dir[0]*tetrad[1][k]+
			dir[1]*tetrad[2][k]+dir[2]*tetrad[3][k]))
	}
	return vec4(u)
}

// shifter is a field's gain on the volumes' light along a ray in it:
// the shift of the light's energy, from the volume to an eye with
// four-velocity obs and momentum p0, to the fourth, the volume's own
// beaming in it so the beam is not counted again.
//
// p is the ray's momentum where it is, which the march hands in at
// each step, so one shifter serves the whole march.
type shifter struct {
	f          field
	p0, obs, p kerr.Vec4
}

// gain is the shift's gain on a crossing's light at a point of the
// world.
//
// an emitter with no four-velocity there, static inside the ergosphere
// or orbiting inside the photon orbit, has no shift: it gives nothing,
// where it gave no number, which spread through the pixel.
func (s *shifter) gain(c crossing, world maths.Vec) float64 {
	g := kerr.Shift(s.p0, s.obs, s.p, s.f.emitter(c.v, world))
	if !(g >= 0) {
		return 0
	}
	return math.Min(shiftTop, g*g*g*g)
}

// bent is what a ray brings back when it enters a field: the volumes
// before the entry, then a march along the geodesic inside, step by
// step, each step a short straight segment that surfaces are tested on
// and volumes gathered along, their light shifted.
//
// If the ray escapes, the straight picture from where it leaves comes
// after; if it came from the horizon, nothing more.
func bent(sc staged, r maths.Ray, f field, enter, pixel float64,
	bounce int) frame.Pixel {
	m := marcher{sc: sc, f: f, pixel: pixel, bounce: bounce,
		keep:  maths.Grey(1),
		depth: float32(enter)}
	if enter > 0 {
		// the straight part in front of the field
		m.keep, m.front = through(sc, r, enter)
	}
	m.start(r, enter)
	for i := 0; i < bends && shows(m.keep); i++ {
		if px, done := m.march(i); done {
			return px
		}
	}
	// captured, or stuck, which is taken as captured: the hole is black
	return m.black()
}

// marcher is one ray's march through a field: the light in front of
// it, the ray in the hole's space, what the volumes it has crossed add
// and how much of what lies behind still gets through, how far it has
// come, and the world point its next segment starts at.
type marcher struct {
	sc     staged
	f      field
	pixel  float64
	bounce int
	front  maths.Light
	ray    kerr.Ray
	shift  shifter
	keep   maths.Light
	inside maths.Light
	depth  float32
	from   maths.Vec
}

// start sets the march off where the ray enters the field, into the
// hole's own space. an eye outside the reach is far away: its energy is
// the conserved one, and the ray's direction is as an eye held still at
// the boundary sees it.
//
// an eye inside has the four-velocity the field was told, and sees its
// own way, aberrated.
func (m *marcher) start(r maths.Ray, enter float64) {
	f, hole := m.f, m.f.hole
	x := f.in(r.Point(enter))
	n := toHole(f.f.InHole(f.at.Orientation.In(r.Dir).Unit()))
	obs, still := atRest, hole.Static(x)
	if enter == 0 {
		obs, still = f.eye(x), f.eye(x)
	}
	m.ray = hole.Arriving(x, hole.FrameOf(x, still), n)
	m.shift = shifter{f: f, p0: m.ray.Momentum(), obs: obs}
	m.from = f.at.Out(f.f.FromHole(parts3(m.ray.Space())))
}

// march is the march's i-th step, and whether it ended the march with
// the pixel it hands back: out of the field, onto a surface, or into
// the hole, which is black; or a short straight segment along the
// geodesic with the volumes on it gathered, and on to the next.
func (m *marcher) march(i int) (frame.Pixel, bool) {
	hole, ray := m.f.hole, m.ray
	here := parts3(ray.Space())
	rad := hole.Radius(ray.Space())
	v := parts4(hole.Velocity(ray))
	outward := here[0]*v[1]+here[1]*v[2]+here[2]*v[3] > 0
	// the first step starts on the reach, where rounding may put it a
	// hair outside, so only a later step leaves by the reach
	if (i > 0 && rad > m.f.f.Reach) ||
		(outward && rad > loose*hole.Mass()) {
		return m.leave(here, rad, v), true
	}
	// a ray that came from the hole's edge hovers there, and one deep
	// inside came from the middle: black both
	if hole.Hovering(ray) || rad < core*hole.Mass() {
		return m.black(), true
	}
	next := hole.Step(ray, hole.PaceIn(ray))
	a, b := m.from, m.f.at.Out(m.f.f.FromHole(parts3(next.Space())))
	m.ray, m.from = next, b
	seg := b.Sub(a)
	length := seg.Len()
	if length <= 0 {
		return frame.Pixel{}, false
	}
	segment := maths.Ray{At: a, Dir: seg.Scale(1 / length)}
	// a surface on this step
	if p, dist, ok := first(m.sc.ps, segment, 0); ok && dist <= length {
		px := surfaceAt(m.sc, segment, m.pixel, m.bounce, p, dist)
		l := m.front.Add(m.inside.Add(px.Light().Times(m.keep)))
		return pixelOf(l, 1, m.depth), true
	}
	// the volumes on this step, their light shifted
	m.shift.p = ray.Momentum()
	k, added := throughGain(m.sc, segment, length, &m.shift, 4)
	m.inside = m.inside.Add(added.Times(m.keep))
	m.keep = m.keep.Times(k)
	m.depth += float32(length)
	return frame.Pixel{}, false
}

// leave is the picture where the ray leaves the field: straight on from
// here along the way it is going, to the reach, gathering the volumes on
// the way, and then the picture beyond.
//
// it is degrees from true, not a part of one (see loose); and the picture
// beyond may meet the reach again, and enter this field afresh, as an eye
// at rest there.
func (m *marcher) leave(here [3]float64, rad float64,
	v [4]float64) frame.Pixel {
	f := m.f
	dir := f.f.FromHole([3]float64{v[1], v[2], v[3]}).Unit()
	exit := maths.Ray{At: f.at.Out(f.f.FromHole(here)),
		Dir: f.at.Orientation.Out(dir).Unit()}
	if left := f.f.Reach*1.002 - rad; left > 0 {
		m.shift.p = m.ray.Momentum()
		k, added := throughGain(m.sc, exit, left*f.scale, &m.shift,
			strides)
		m.inside = m.inside.Add(added.Times(m.keep))
		m.keep = m.keep.Times(k)
		exit.At = exit.At.Add(exit.Dir.Scale(left * f.scale))
	}
	behind := shade(m.sc, exit, m.pixel, m.bounce)
	l := m.front.Add(m.inside.Add(behind.Light().Times(m.keep)))
	return pixelOf(l, float32(1-most(m.keep)*(1-float64(behind.Cover))),
		m.depth+behind.Depth)
}

// black is what the march brings back when the ray came from the hole:
// what lay in front of it and what it gathered, and nothing behind.
func (m *marcher) black() frame.Pixel {
	return pixelOf(m.front.Add(m.inside), 1, m.depth)
}

// The tracer's sums run on plain parts, as the scene's InHole and
// FromHole give and take them; celestial's hole keeps its vectors whole.
// These carry one to the other, part for part, with no arithmetic.

// toHole is a point or a way in the hole's space, as celestial takes it.
func toHole(parts [3]float64) kerr.Vec3 {
	return kerr.NewVec3(parts[0], parts[1], parts[2])
}

// parts3 is a point or a way of celestial's, as its three parts.
func parts3(point kerr.Vec3) [3]float64 {
	return [3]float64{point.AlongX(), point.AlongY(), point.AlongZ()}
}

// parts4 is a four-vector of celestial's, time first, as its parts.
func parts4(vector kerr.Vec4) [4]float64 {
	return [4]float64{vector.Time(), vector.AlongX(), vector.AlongY(),
		vector.AlongZ()}
}

// vec4 is four parts, time first, as celestial's four-vector.
func vec4(parts [4]float64) kerr.Vec4 {
	return kerr.NewVec4(parts[0], parts[1], parts[2], parts[3])
}

// axes is an observer's tetrad as rows of parts: the observer first,
// then the x, y and z axes, the order celestial's Frame keeps them in.
func axes(tetrad kerr.Frame) [4][4]float64 {
	return [4][4]float64{parts4(tetrad.Observer()), parts4(tetrad.XAxis()),
		parts4(tetrad.YAxis()), parts4(tetrad.ZAxis())}
}
