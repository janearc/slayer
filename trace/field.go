package trace

import (
	"math"
	"time"

	kerr "github.com/janearc/celestial/hole"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

// placedField is a field placed in the world at one moment: where it
// is, its scale there, and the hole its mass makes.
type placedField struct {
	source *thing.Field
	at     maths.Subspace
	scale  float64
	hole   kerr.Hole
}

// placeFields is every field, where it is at a moment.
func placeFields(world *thing.World, moment time.Duration) []placedField {
	fields := make([]placedField, 0, len(world.Fields))
	for _, each := range world.Fields {
		// written the positive way, so a mass or a reach that is not
		// a number is no field either
		if !(each.Reach > 0 && each.Mass > 0) {
			continue
		}
		// nor is a mass and spin that make no hole, as celestial says
		hole, err := kerr.NewHole(each.Mass, each.Spin)
		if err != nil {
			continue
		}
		at := each.At(moment)
		fields = append(fields, placedField{source: each, at: at,
			scale: unitScale(at),
			hole:  *hole})
	}
	return fields
}

// bends is how many steps a ray may take inside a field before it is
// given up as stuck and taken as captured: a ray near the photon
// sphere winds many times before it leaves, and must be let.
const bends = 40000

// loose is the radius, in masses, past which a ray heading outward is
// let go straight. The bending left beyond it is not small: against
// the hole's own march from far to far, a ray passing 12.9 masses
// from a hole of mass one turns 15.7 degrees here and 23.4 in full.
//
// Measured on 2026-10-04: 0.9 of a degree short passing 10 masses out,
// 7.7 at 12.9, 6.6 at 20, 5.0 at 30 and 6.0 at 39.9, where the reach
// is 40. Passing 8 out or nearer the two agree to a hundredth.
const loose = 12

// enterEpsilon is the least way along a ray a field's reach has to
// lie for the ray to count as going into it: a ray that starts on the
// reach, by rounding, has not just entered.
const enterEpsilon = 1e-9

// reachMargin is how far past the reach, as a share of it, a ray
// leaving a field is carried straight before the picture beyond is
// asked for, so that picture does not begin on the reach itself.
const reachMargin = 1.002

// fieldStrides is how many steps a crossing is cut into on one step of
// a march through a field, where the eye's own march cuts a whole
// crossing into strides: a step of the march is short.
const fieldStrides = 4

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
func (field placedField) in(point maths.Vec) kerr.Vec3 {
	return toHole(field.source.InHole(field.at.In(point)))
}

// entry is a ray going into a field: the field, and how far along the ray it
// goes in, zero when the ray starts inside.
type entry struct {
	field placedField
	dist  float64
}

// entering is the nearest field the ray will enter, and how far along it
// enters: zero when the ray starts inside one. A ray already past a field's
// reach, or missing it, does not enter it.
func entering(fields []placedField, ray maths.Ray) (entry, bool) {
	nearest, found := entry{dist: nowhere}, false
	for _, each := range fields {
		local := each.at.RayIn(ray)
		reach := each.source.Reach
		// the reach sphere about the field's zero, in its own units
		closing := local.At.Dot(local.Dir)
		outside := local.At.Dot(local.At) - reach*reach
		if outside < 0 {
			// inside already
			return entry{field: each}, true
		}
		disc := closing*closing - outside
		if disc < 0 {
			continue
		}
		along := -closing - math.Sqrt(disc)
		if along <= enterEpsilon {
			continue
		}
		if enters := along * each.scale; enters < nearest.dist {
			nearest, found = entry{field: each, dist: enters}, true
		}
	}
	return nearest, found
}

// eye is the eye's four-velocity at a point inside the field: the one
// the field was told, or an eye held still there.
func (field placedField) eye(x kerr.Vec3) kerr.Vec4 {
	if field.source.EyeTime <= 0 {
		return field.hole.Static(x)
	}
	velocity := field.source.InHole(field.source.Eye)
	return kerr.NewVec4(field.source.EyeTime, velocity[0], velocity[1],
		velocity[2])
}

// emitter is a volume's four-velocity at a point of the world, in the hole's
// space. A volume with no beam is still, and so is one whose beam's Speed is
// not between zero and one, round or not.
//
// A round beam whose Speed is between zero and one goes round at the hole's own
// circular rate, celestial's Circular, whatever its Speed and At say: inside a
// field the hole's rate wins.
//
// A beam along its axis flows as Beam.Flow says, away from its origin, as a
// jet's.
func (field placedField) emitter(volume *placedVolume,
	point maths.Vec) kerr.Vec4 {
	x := field.in(point)
	beam := volume.source.Beam
	if beam == nil || beam.Speed <= 0 || beam.Speed >= 1 {
		return field.hole.Static(x)
	}
	if beam.Round {
		return field.hole.Circular(x)
	}
	flow, speed := beam.Flow(volume.at.In(point))
	inWorld := volume.at.Orientation.Out(flow)
	dir := field.source.InHole(field.at.Orientation.In(inWorld).Unit())
	tetrad := axes(field.hole.FrameOf(x, field.hole.Static(x)))
	gamma := 1 / math.Sqrt(1-speed*speed)
	var flowing [4]float64
	for k := range 4 {
		flowing[k] = gamma * (tetrad[0][k] + speed*(dir[0]*tetrad[1][k]+
			dir[1]*tetrad[2][k]+dir[2]*tetrad[3][k]))
	}
	return vec4(flowing)
}

// shifter is a field's gain on the volumes' light along a ray in it:
// the shift of the light's energy from the volume to the eye, to the
// fourth power, held to shiftTop. obs is the eye's four-velocity and
// arrived the ray's momentum where it met the eye.
//
// The volume's own motion is in the emitter's four-velocity, so a
// beam's boost, a cube of the doppler factor, is not applied as well
// inside a field. Outside one it is, so the two differ at the reach.
//
// It holds what is the same for a whole march. The ray's momentum
// where it is changes at every step, and is handed to gain.
type shifter struct {
	field        placedField
	arrived, obs kerr.Vec4
}

// gain is the shift's gain on a crossing's light at a point of the
// world, for a ray whose momentum there is p.
func (shift shifter) gain(crossed crossing, point maths.Vec,
	momentum kerr.Vec4) float64 {
	shifted := kerr.Shift(shift.arrived, shift.obs, momentum,
		shift.field.emitter(crossed.volume, point))
	return capped(shifted * shifted * shifted * shifted)
}

// gainer is a gain on a crossing's light at a point of the world, as
// throughGain asks for it; nil is no gain.
type gainer func(crossed crossing, point maths.Vec) float64

// capped is a gain held to shiftTop at most. A gain that is not a number, from
// a geodesic gone wrong, is zero: that step is dark, where a NaN would spread
// through every sum it joined.
func capped(gain float64) float64 {
	if math.IsNaN(gain) {
		return 0
	}
	return math.Min(shiftTop, gain)
}

// bent is what a ray brings back when it enters a field: the volumes
// before the entry, then a march along the geodesic inside, step by
// step, each step a short straight segment that surfaces are tested on
// and volumes gathered along, their light shifted.
//
// If the ray escapes, the straight picture from where it leaves comes
// after; if it came from the horizon, nothing more.
func bent(scene staged, seen look, into entry) frame.Pixel {
	walker := newMarcher(scene, seen, into)
	for i := 0; i < bends && shows(walker.keep); i++ {
		if drawn, done := walker.march(i); done {
			return drawn
		}
	}
	// captured, or stuck, which is taken as captured: the hole is black
	return walker.black()
}

// marcher is one ray's march through a field: the light in front of
// it, the ray in the hole's space, what the volumes it has crossed add
// and how much of what lies behind still gets through, how far it has
// come, and the world point its next segment starts at.
type marcher struct {
	scene  staged
	field  placedField
	asked  look
	front  maths.Light
	ray    kerr.Ray
	shift  shifter
	keep   maths.Light
	inside maths.Light
	depth  float32
	from   maths.Vec
}

// newMarcher is a march of a ray into a field it enters so far along,
// ready for its first step: what the ray gathered in front of the
// field, and the ray set off in the hole's own space. It is the only
// way to a marcher, so a march never starts half made.
func newMarcher(scene staged, seen look, into entry) *marcher {
	ray, enter := seen.ray, into.dist
	walker := &marcher{scene: scene, field: into.field, asked: seen,
		keep: maths.Grey(1), depth: float32(enter)}
	if enter > 0 {
		// the straight part in front of the field
		before := through(scene, ray, enter)
		walker.keep, walker.front = before.keep, before.added
	}
	walker.start(ray, enter)
	return walker
}

// come is how far the march's look has come: whatever way it had come
// before the march, and the march's own depth so far.
func (walker *marcher) come() float64 {
	return walker.asked.come + float64(walker.depth)
}

// gainAt is the field's gain for a ray of a momentum, as throughGain
// takes it.
func (walker *marcher) gainAt(momentum kerr.Vec4) gainer {
	return func(crossed crossing, point maths.Vec) float64 {
		return walker.shift.gain(crossed, point, momentum)
	}
}

// start sets the march off where the ray enters the field, into the
// hole's own space. An eye outside the reach is far away: its energy is
// the conserved one, and the ray's direction is as an eye held still at
// the boundary sees it.
//
// An eye inside has the four-velocity the field was told, and sees its
// own way, aberrated. That is for the eye's own ray alone: a later leg
// starting inside, off a mirror or out of a march, is no eye's, and
// its direction is as one held still there sees it.
func (walker *marcher) start(ray maths.Ray, enter float64) {
	field, hole := walker.field, walker.field.hole
	x := field.in(ray.Point(enter))
	heading := toHole(field.source.InHole(
		field.at.Orientation.In(ray.Dir).Unit()))
	obs, still := atRest, hole.Static(x)
	if enter == 0 && !walker.asked.onward {
		obs = field.eye(x)
		still = obs
	}
	walker.ray = hole.Arriving(x, hole.FrameOf(x, still), heading)
	walker.shift = shifter{field: field, arrived: walker.ray.Momentum(),
		obs: obs}
	walker.from = field.at.Out(field.source.FromHole(
		parts3(walker.ray.Space())))
}

// march is the march's i-th step, and whether it ended the march with
// the pixel it hands back: out of the field, onto a surface, or into
// the hole, which is black; or a short straight segment along the
// geodesic with the volumes on it gathered, and on to the next.
func (walker *marcher) march(i int) (frame.Pixel, bool) {
	hole, ray := walker.field.hole, walker.ray
	here := parts3(ray.Space())
	radius := hole.Radius(ray.Space())
	velocity := parts4(hole.Velocity(ray))
	outward := here[0]*velocity[1]+here[1]*velocity[2]+
		here[2]*velocity[3] > 0
	// the first step starts on the reach, where rounding may put it a
	// hair outside, so only a later step leaves by the reach
	past := i > 0 && radius > walker.field.source.Reach
	if past || (outward && radius > loose*hole.Mass()) {
		return walker.leave(here, radius, velocity), true
	}
	// a ray that came from the hole's edge hovers there, and one deep
	// inside came from the middle: black both
	if hole.Hovering(ray) || radius < core*hole.Mass() {
		return walker.black(), true
	}
	next := hole.Step(ray, hole.PaceIn(ray))
	arrived := walker.field.source.FromHole(parts3(next.Space()))
	from, onward := walker.from, walker.field.at.Out(arrived)
	seg := onward.Sub(from)
	length := seg.Len()
	// a step that goes nowhere, or that is not a number, as a ray with
	// no momentum makes, is passed over before the march moves at all
	if !(length > 0) || math.IsInf(length, 0) {
		return frame.Pixel{}, false
	}
	walker.ray, walker.from = next, onward
	segment := maths.Ray{At: from, Dir: seg.Scale(1 / length)}
	drawn, ended := walker.run(segment, length, ray.Momentum(),
		fieldStrides)
	if ended {
		return drawn, true
	}
	walker.depth += float32(length)
	return frame.Pixel{}, false
}

// run takes the march along one straight segment of a length. If a
// surface stands on it the march ends there: the volumes in front of
// the surface are gathered, and the pixel is the surface behind them.
//
// If none does, the volumes along the whole of it are gathered, their
// light shifted for a ray of that momentum, and the march goes on.
func (walker *marcher) run(segment maths.Ray, length float64,
	momentum kerr.Vec4, cuts int) (frame.Pixel, bool) {
	met, found := walker.scene.crowd.first(segment, 0)
	if found && met.dist <= length {
		walker.gather(segment, met.dist, momentum, cuts)
		return walker.onto(segment, met), true
	}
	walker.gather(segment, length, momentum, cuts)
	return frame.Pixel{}, false
}

// onto is the pixel of a march that ends on a surface: the surface as
// the straight tracer draws it where the segment meets it, behind what
// the march has gathered.
func (walker *marcher) onto(segment maths.Ray, met hit) frame.Pixel {
	drawn := surfaceAt(walker.scene, walker.asked.along(segment,
		walker.come()), met)
	shown := drawn.Light().Times(walker.keep)
	light := walker.front.Add(walker.inside).Add(shown)
	depth := walker.depth + float32(met.dist)
	return pixelOf(light, 1, depth)
}

// gather adds the volumes along a straight segment to what the march
// has gathered, their light shifted for a ray of that momentum there, the
// segment's crossings cut into so many steps.
func (walker *marcher) gather(segment maths.Ray, length float64,
	momentum kerr.Vec4, cuts int) {
	along := throughGain(walker.scene, segment, length,
		walker.gainAt(momentum), cuts)
	walker.inside = walker.inside.Add(along.added.Times(walker.keep))
	walker.keep = walker.keep.Times(along.keep)
}

// leave is the picture where the ray leaves the field: straight on from
// here along the way it is going to just past the reach, gathering the
// volumes on the way, and then the picture beyond. The straight way is
// up to some eight degrees from true; loose has the numbers.
//
// The picture beyond may enter this field again: a ray moving mostly
// sideways ends its straight run inside the reach, and is marched and
// let go once more. Measured, that nests from 2 deep to 20.
//
// A surface met on the way to the reach ends the ray there, as one met
// on any step of the march does.
func (walker *marcher) leave(here [3]float64, radius float64,
	velocity [4]float64) frame.Pixel {
	field := walker.field
	dir := field.source.FromHole([3]float64{velocity[1], velocity[2],
		velocity[3]}).Unit()
	exit := maths.Ray{At: field.at.Out(field.source.FromHole(here)),
		Dir: field.at.Orientation.Out(dir).Unit()}
	beyond := field.source.Reach * reachMargin
	if left := (beyond - radius) * field.scale; left > 0 {
		drawn, ended := walker.run(exit, left, walker.ray.Momentum(),
			strides)
		if ended {
			return drawn
		}
		exit.At = exit.At.Add(exit.Dir.Scale(left))
		walker.depth += float32(left)
	}
	behind := shade(walker.scene, walker.asked.along(exit, walker.come()))
	shown := behind.Light().Times(walker.keep)
	light := walker.front.Add(walker.inside).Add(shown)
	cover := float32(1 - most(walker.keep)*(1-float64(behind.Cover)))
	depth := walker.depth + behind.Depth
	return pixelOf(light, cover, depth)
}

// black is what the march brings back when the ray came from the hole:
// what lay in front of it and what it gathered, and nothing behind.
func (walker *marcher) black() frame.Pixel {
	return pixelOf(walker.front.Add(walker.inside), 1, walker.depth)
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
