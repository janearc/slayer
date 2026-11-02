// Package trace renders a world from a camera at a specified time. This is the
// older, audited tracer; the top-level trace package contains newer fixes and
// is the migration target for its callers.
//
// Camera rays find the nearest surface and integrate the volumes in front of
// it. Surface lighting uses lamps and shadows; volume integration adds emission
// and scattering while attenuating the light behind each sample. Surfaces and
// volumes can shadow one another.
//
// Volumes do not indirectly illuminate other objects. The global haze model
// also omits attenuation of lamp light. These are modelling limitations, not
// additional light-transport paths.
package trace

import (
	"math"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
	tracer "github.com/janearc/slayer/trace"
)

// touch is how close a march has to come to a surface to count as
// meeting it, and lift how far a shadow ray starts off the surface so it
// does not meet the point it left: each a share of the lengths about it,
// as march and lifting say.
//
// Shares and not lengths, so a scene a thousandth or a thousand times as
// big draws the same; the same shares as slayer's tracer (trace).
const (
	touch = 1e-6
	lift  = 1e-6
	steps = 256
	// bounces is how many times a ray may mirror off a glossy surface:
	// once, so a hall of mirrors ends.
	bounces = 1
	nowhere = math.MaxFloat64
	// strides is how many steps a ray takes across a volume it crosses,
	// and shades how many it takes toward a lamp from each of those
	strides = 48
	shades  = 10
	// faint is how little of the light behind has to be left for a march
	// to stop early. what is left is kept, not the rest of the depth, so a
	// bright backdrop past thick smoke shows: 9.7 for a sky of 1e4 behind
	// a depth of nine, where 1.2 is true. kept as it was, as ruled
	faint = 1e-3
	// evenly is how much of a lamp an isotropic scatterer sends toward the
	// eye, against a surface's cosine: a lit cloud beside a lit wall
	evenly = 0.5
)

// grey is what a thing with no material is made of.
var grey = &thing.Material{Name: "grey", Colour: maths.Grey(0.8)}

// eye is the three wavelengths a gas is asked at for the eye's three
// channels, red, green and blue, since maths.Light is three numbers
// and not a spectrum: the sRGB primaries' dominant wavelengths, near
// enough.
//
// Rayleigh's fourth power across them is a blue sky and a red limb.
// When light is per band this goes.
var eye = [3]maths.Band{{Nanometres: 610}, {Nanometres: 550},
	{Nanometres: 465}}

// byEye is a light from a gas's number at each of the eye's bands.
func byEye(f func(band maths.Band) float64) maths.Light {
	return maths.RGB(f(eye[0]), f(eye[1]), f(eye[2]))
}

// placed is a thing where it is at one moment.
type placed struct {
	thing *thing.Thing
	at    maths.Subspace
	scale float64
	// the material's textures as they are at the picture's time
	tex, shine, rough texture.Texture
}

// ticked is a texture as it is at time t: itself unless it moves.
func ticked(tx texture.Texture, t time.Duration) texture.Texture {
	if moving, ok := tx.(texture.Timed); ok {
		return moving.Tick(t)
	}
	return tx
}

// emits is the light a material gives off at a point, from its shine
// as it is at the picture's time, or its glow.
func emits(m *thing.Material, shine texture.Texture, local maths.Vec) maths.Light {
	if shine != nil {
		return shine.At(local)
	}
	return m.Glow
}

// place is every thing that can be seen, where it is at time t.
func place(w *thing.World, t time.Duration) []placed {
	ps := make([]placed, 0, len(w.Things))
	for _, th := range w.Things {
		if th.Hidden() {
			continue
		}
		at := th.At(t)
		s := at.Scale
		if s == 0 {
			s = 1
		}
		p := placed{thing: th, at: at, scale: s}
		if m := th.Material; m != nil {
			p.tex, p.shine = ticked(m.Texture, t), ticked(m.Shine, t)
			p.rough = ticked(m.Rough, t)
		}
		ps = append(ps, p)
	}
	return ps
}

// volume is a volume where it is at one moment.
type volume struct {
	v     *thing.Volume
	at    maths.Subspace
	scale float64
	// the medium and the material's textures as they are at the
	// picture's time
	med               medium.Medium
	tex, shine, rough texture.Texture
	// gas is the volume's optics when its medium is a gas, per unit
	// of the medium's space at unit density, and nil for a painted
	// volume.
	//
	// A gas volume's density is a share of its full density, one at
	// a world's surface, and its material's colour and radiant are
	// not read: the gas says what light meets, in every band.
	gas *gas.Gas
}

// placeVolumes is every volume that can be seen, where it is at time t.
func placeVolumes(w *thing.World, t time.Duration) []volume {
	vs := make([]volume, 0, len(w.Volumes))
	for _, v := range w.Volumes {
		if v.Hidden() || v.Medium == nil {
			continue
		}
		at := v.At(t)
		s := at.Scale
		if s == 0 {
			s = 1
		}
		pv := volume{v: v, at: at, scale: s, med: v.Medium}
		if moving, ok := v.Medium.(medium.Timed); ok {
			pv.med = moving.Tick(t)
		}
		if o, ok := pv.med.(gas.Optical); ok {
			g := o.Gas()
			// asked here, on the caller's goroutine, so a gas
			// that cannot scatter is refused by name before a
			// worker meets it, where it would end the process
			g.PhaseAt(1)
			pv.gas = &g
		}
		if m := v.Material; m != nil {
			pv.tex, pv.shine = ticked(m.Texture, t), ticked(m.Shine, t)
			pv.rough = ticked(m.Rough, t)
		}
		vs = append(vs, pv)
	}
	return vs
}

// staged is a world as one picture sees it: the world, and its things,
// volumes and fields where they are at the picture's time. it is not
// package scene, which is what there is; it is what there is, placed.
type staged struct {
	w  *thing.World
	ps []placed
	vs []volume
	fs []field
	// spread is how big the world is where floats are concerned, as
	// slayer's tracer measures it, which a ray leaving a surface is
	// lifted by a share of
	spread float64
}

// stage is the world placed at time t.
func stage(w *thing.World, t time.Duration) staged {
	return staged{w, place(w, t), placeVolumes(w, t), placeFields(w, t),
		tracer.Spread(w, t)}
}

// boxReach is how big a box is where floats are concerned: its size or
// its distance from the origin, whichever is more, as trace measures it.
func boxReach(box maths.Box) float64 {
	return math.Max(box.Max.Sub(box.Min).Len(),
		math.Max(box.Min.Len(), box.Max.Len()))
}

// lifted is where a ray leaving a surface at a point starts, off it along
// its normal so it does not meet the point it left. from is where the ray
// that met the point started.
func (sc staged) lifted(point, normal, from maths.Vec) maths.Vec {
	return point.Add(normal.Scale(sc.lifting(point, from)))
}

// lifting is how far off a surface a ray leaving it at a point starts:
// lift's share of the world's spread, the point's distance from the origin
// or the distance of where the ray that met it started, whichever is most.
//
// The last is for a world with no edge met near the origin, which has no
// other length to measure by.
func (sc staged) lifting(point, from maths.Vec) float64 {
	return lift * math.Max(sc.spread, math.Max(point.Len(), from.Len()))
}

// pixelOf is a pixel of light l, covered and as deep as given.
func pixelOf(l maths.Light, cover, depth float32) frame.Pixel {
	return frame.Pixel{R: float32(l.R), G: float32(l.G), B: float32(l.B),
		Cover: cover, Depth: depth}
}

// crossing is where a ray enters and leaves one volume, in the world's
// distances, and the ray in the volume's own space.
type crossing struct {
	v            volume
	enter, leave float64
	local        maths.Ray
}

// crossings is every volume the ray passes through before far, nearest
// entry first.
func crossings(vs []volume, r maths.Ray, far float64) []crossing {
	var cs []crossing
	for _, v := range vs {
		local := v.at.RayIn(r)
		limit := math.Min(far, nowhere) / v.scale
		var enter, leave float64
		var ok bool
		if sp, exact := v.med.(medium.Spanner); exact {
			enter, leave, ok = sp.Span(local, 0, limit)
		} else {
			enter, leave, ok = v.med.Bounds().Hit(local, 0, limit)
		}
		if !ok || leave <= enter {
			continue
		}
		cs = append(cs, crossing{v, enter * v.scale, leave * v.scale, local})
	}
	if len(cs) > 1 {
		sort.Slice(cs, func(i, j int) bool {
			return cs[i].enter < cs[j].enter
		})
	}
	return cs
}

// through is what the ray gathers from the volumes between the eye and
// far: how much of the light from far still gets through, and the light
// the volumes add in front of it.
//
// Volumes are added up front to back, step by step: where two overlap,
// as an aurora round a giant or clouds in air, their steps are taken in
// turn by how far along the ray each is.
func through(sc staged, r maths.Ray, far float64) (keep maths.Light,
	added maths.Light) {
	return throughGain(sc, r, far, nil, strides)
}

// shows is whether enough light still gets through for a march to go
// on: any channel of keep above faint.
func shows(keep maths.Light) bool {
	return keep.R > faint || keep.G > faint || keep.B > faint
}

// most is the channel of keep that lets the most through, which is what
// a pixel's cover is taken from.
func most(keep maths.Light) float64 {
	return math.Max(keep.R, math.Max(keep.G, keep.B))
}

// throughGain is through with the volumes' light scaled by a field's
// shift, nil for none, and a crossing cut into n steps, since the bent
// march's segments are short and want few.
func throughGain(sc staged, r maths.Ray, far float64, gain *shifter,
	n int) (keep maths.Light, added maths.Light) {
	keep = maths.Grey(1)
	cs := crossings(sc.vs, r, far)
	var gbuf [8]float64
	gs := gbuf[:0]
	for _, c := range cs {
		g := 1.0
		if gain != nil {
			g = gain.gain(c, r.Point((c.enter+c.leave)/2))
		}
		gs = append(gs, g)
	}
	st := stepping(cs, n)
	for shows(keep) {
		k, d, step, ok := st.take()
		if !ok {
			break
		}
		c := cs[k]
		local := c.local.Point(d / c.v.scale)
		density := c.v.med.Density(local)
		if density <= 0 {
			continue
		}
		if c.v.gas != nil {
			// a gas: what each channel loses and gathers over the
			// step, by the gas's own numbers at the eye's bands
			keep, added = gasStep(sc, c, local, r.Point(d), r.Dir,
				density*step/c.v.scale, gs[k], keep, added)
			continue
		}
		// the step's own thickness, in the medium's own units
		absorb := 1 - math.Exp(-density*step/c.v.scale)
		light := scattered(sc, c, local, r.Point(d), r.Dir, gain == nil)
		// each channel as the one number was, in the same order,
		// so a painted volume's picture is the one it always was
		added.R += light.R * (keep.R * absorb * gs[k])
		added.G += light.G * (keep.G * absorb * gs[k])
		added.B += light.B * (keep.B * absorb * gs[k])
		keep.R *= 1 - absorb
		keep.G *= 1 - absorb
		keep.B *= 1 - absorb
	}
	return keep, added
}

// gasStep is one step of a ray through a gas: each of the eye's
// channels keeps what beer's law leaves of it over the step, and
// gathers what the gas scatters toward the eye of every lamp's light
// that reaches the point.
//
// That is the gas's scattering at the band, its phase at the lamp's
// angle to the eye, and the density there. thick is the step's density
// times its length in the medium's own units.
func gasStep(sc staged, c crossing, local, world, dir maths.Vec,
	thick, gain float64, keep, added maths.Light) (maths.Light,
	maths.Light) {
	g := c.v.gas
	lost := byEye(func(b maths.Band) float64 {
		return 1 - math.Exp(-g.Extinction(b)*thick)
	})
	scatter := byEye(g.Scatter)
	var in maths.Light
	for _, lamp := range sc.w.Lamps {
		toward, far, light := lamp.Light(world, dir.Neg())
		if light == (maths.Light{}) {
			continue
		}
		if toward == (maths.Vec{}) {
			// from everywhere at once: as a painted volume takes it
			in = in.Add(light.Scale(evenly))
			continue
		}
		shadow := maths.Ray{At: world, Dir: toward}
		if blocked(sc.ps, shadow, far) {
			continue
		}
		reach := light.Scale(clear(sc.vs, world, toward)).Times(
			gasClear(sc.vs, world, toward))
		// the angle scattered through: from the lamp's way of
		// travel, minus toward, to the eye's, minus dir
		cos := toward.Unit().Neg().Dot(dir.Neg())
		in = in.Add(reach.Scale(g.PhaseAt(cos)))
	}
	// what this step sends on: the light gathered times what the gas
	// scatters of it, over the step's thickness, of what still shows
	gathered := in.Times(scatter).Scale(thick * gain)
	added = added.Add(gathered.Times(keep))
	keep = keep.Times(maths.RGB(1-lost.R, 1-lost.G, 1-lost.B))
	return keep, added
}

// gasClear is how much of each channel of a lamp's light reaches a
// point through every gas volume between, a march toward the lamp
// through each one, as clear is for the painted volumes; one where
// there is none.
func gasClear(vs []volume, world, toward maths.Vec) maths.Light {
	depth := maths.Light{}
	any := false
	for _, v := range vs {
		if v.gas == nil {
			continue
		}
		r := v.at.RayIn(maths.Ray{At: world, Dir: toward.Unit()})
		var enter, leave float64
		var ok bool
		if sp, exact := v.med.(medium.Spanner); exact {
			enter, leave, ok = sp.Span(r, 0, nowhere)
		} else {
			enter, leave, ok = v.med.Bounds().Hit(r, 0, nowhere)
		}
		if !ok || leave <= enter {
			continue
		}
		any = true
		step := (leave - enter) / shades
		k := 2 / step
		jitter := 0.5 + 0.5*noise.Gradient(77, world.X*k, world.Y*k,
			world.Z*k)
		ext := byEye(v.gas.Extinction)
		for i := 0; i < shades; i++ {
			at := r.Point(enter + (float64(i)+jitter)*step)
			depth = depth.Add(ext.Scale(v.med.Density(at) * step))
		}
	}
	if !any {
		return maths.Grey(1)
	}
	return maths.RGB(math.Exp(-depth.R), math.Exp(-depth.G),
		math.Exp(-depth.B))
}

// hasGas is whether any volume of the scene is a gas.
func hasGas(vs []volume) bool {
	for _, v := range vs {
		if v.gas != nil {
			return true
		}
	}
	return false
}

// stepper hands out the steps across a ray's crossings front to back:
// each crossing cut into n steps, and the steps of crossings that overlap
// taken in turn by how far along the ray each is. Crossings that do not
// overlap come one after another, as they always did.
type stepper struct {
	cs   []crossing
	n    int
	next []int
	buf  [8]int
}

// stepping is a stepper over the crossings, each cut into n steps.
func stepping(cs []crossing, n int) *stepper {
	s := &stepper{cs: cs, n: n}
	s.next = s.buf[:0]
	for range cs {
		s.next = append(s.next, 0)
	}
	return s
}

// take is the nearest step not yet taken: which crossing it is in, how
// far along the ray its middle is, and how long it is. A tie goes to
// the crossing entered first.
func (s *stepper) take() (k int, d, step float64, ok bool) {
	k = -1
	for j, c := range s.cs {
		i := s.next[j]
		if i >= s.n {
			continue
		}
		sj := (c.leave - c.enter) / float64(s.n)
		dj := c.enter + (float64(i)+0.5)*sj
		if k < 0 || dj < d {
			k, d, step = j, dj, sj
		}
	}
	if k < 0 {
		return 0, 0, 0, false
	}
	s.next[k]++
	return k, d, step, true
}

// scattered is the light a bit of volume sends toward the eye: what it
// scatters of every lamp's light that reaches it, past the surfaces and
// through the volumes in the way, evenly in all directions, and its glow.
func scattered(sc staged, c crossing, local, world, dir maths.Vec,
	beam bool) maths.Light {
	m := c.v.v.Material
	if m == nil {
		m = grey
	}
	colour := m.Colour
	if c.v.tex != nil {
		colour = c.v.tex.At(local)
	}
	sum := emits(m, c.v.shine, local)
	if beam {
		sum = sum.Scale(boosted(c.v, local, dir))
	}
	facing := dir.Neg()
	for _, lamp := range sc.w.Lamps {
		toward, far, light := lamp.Light(world, facing)
		reach := evenly
		if toward != (maths.Vec{}) {
			shadow := maths.Ray{At: world, Dir: toward}
			if blocked(sc.ps, shadow, far) {
				continue
			}
			reach *= clear(sc.vs, world, toward)
		}
		l := colour.Times(light).Scale(reach)
		if toward != (maths.Vec{}) && hasGas(sc.vs) {
			l = l.Times(gasClear(sc.vs, world, toward))
		}
		sum = sum.Add(l)
	}
	return sum
}

// shone is how much lamplight reaches a point in the world, as
// luminance, past the surfaces and through the volumes in the way.
func shone(sc staged, world, facing maths.Vec) float64 {
	sum := 0.0
	for _, lamp := range sc.w.Lamps {
		toward, far, light := lamp.Light(world, facing)
		reach := 1.0
		if toward != (maths.Vec{}) {
			shadow := maths.Ray{At: world, Dir: toward}
			if blocked(sc.ps, shadow, far) {
				continue
			}
			reach = clear(sc.vs, world, toward)
			if hasGas(sc.vs) {
				reach *= gasClear(sc.vs, world, toward).Luminance()
			}
		}
		sum += light.Luminance() * reach
	}
	return sum
}

// boosted is how much brighter a bit of a volume looks for its beam,
// toward the eye that the ray runs away from: one for a volume at rest.
func boosted(v volume, local, dir maths.Vec) float64 {
	if v.v.Beam == nil {
		return 1
	}
	return v.v.Beam.Boost(v.at.Orientation.In(dir.Neg()), local)
}

// clear is how much of a lamp's light reaches a point in the world
// through every volume the way crosses: a short march toward the lamp
// through each, out to where it leaves it. not only between: a volume
// past a lamp near the point is counted too, as if it stood in the way.
func clear(vs []volume, world, toward maths.Vec) float64 {
	depth := 0.0
	for _, v := range vs {
		if v.gas != nil {
			// a gas's density is a share, not an extinction;
			// gasClear has it
			continue
		}
		r := v.at.RayIn(maths.Ray{At: world, Dir: toward.Unit()})
		var enter, leave float64
		var ok bool
		if sp, exact := v.med.(medium.Spanner); exact {
			enter, leave, ok = sp.Span(r, 0, nowhere)
		} else {
			enter, leave, ok = v.med.Bounds().Hit(r, 0, nowhere)
		}
		if !ok || leave <= enter {
			continue
		}
		step := (leave - enter) / shades
		// the samples start part way along the first step, by a smooth
		// noise over the point two wobbles to a step, so a coarse march
		// does not lay its steps down as tiers of shadow across a
		// volume, and neighbouring points agree, so there is no grain
		k := 2 / step
		jitter := 0.5 + 0.5*noise.Gradient(77, world.X*k, world.Y*k, world.Z*k)
		for i := 0; i < shades; i++ {
			at := r.Point(enter + (float64(i)+jitter)*step)
			depth += v.med.Density(at) * step
		}
	}
	return math.Exp(-depth)
}

// meet is how far along r, between near and far, the ray first meets this
// thing, in the world's own distances.
func (p placed) meet(r maths.Ray, near, far float64) (float64, bool) {
	local := p.at.RayIn(r)
	n, f := near/p.scale, far/p.scale
	s := p.thing.Shape
	var d float64
	var ok bool
	if h, exact := s.(shape.Hitter); exact {
		d, ok = h.Hit(local, n, f)
	} else {
		d, ok = march(s, local, n, f)
	}
	return d * p.scale, ok
}

// march steps along the ray by the distance to the nearest surface, which is
// always safe, until it is close enough to count or has gone too far.
//
// Close enough is touch's share of the shape's reach, the point's
// distance from the origin or the ray's start's, whichever is most: a
// float is as fine as the numbers it was made from, and a shape with no
// edge is measured where the march is and where it came from.
func march(s shape.Shape, r maths.Ray, near, far float64) (float64, bool) {
	box := s.Bounds()
	enter, leave, ok := box.Hit(r, near, far)
	if !ok {
		return 0, false
	}
	size := boxReach(box)
	if math.IsInf(size, 0) || math.IsNaN(size) {
		size = 0
	}
	size = math.Max(size, r.At.Len())
	t := enter
	for i := 0; i < steps && t <= leave; i++ {
		at := r.Point(t)
		d := s.Distance(at)
		if math.Abs(d) < touch*math.Max(size, at.Len()) {
			return t, true
		}
		t += math.Abs(d)
	}
	return 0, false
}

// first is the nearest thing the ray meets, and how far along it is.
func first(ps []placed, r maths.Ray, near float64) (placed, float64, bool) {
	best, dist := placed{}, nowhere
	for _, p := range ps {
		if d, ok := p.meet(r, near, dist); ok && d < dist {
			best, dist = p, d
		}
	}
	return best, dist, dist < nowhere
}

// blocked is whether anything stands between a point and a lamp far along
// the ray; far is +Inf for a lamp at infinity.
func blocked(ps []placed, r maths.Ray, far float64) bool {
	far = math.Min(far, nowhere)
	for _, p := range ps {
		if p.thing.Clear() {
			continue
		}
		if _, ok := p.meet(r, 0, far); ok {
			return true
		}
	}
	return false
}

// shade is what one ray brings back: the first thing it meets, lit by every
// lamp that can see that point, or the sky, with whatever volumes lie in front
// of it added over.
//
// pixel is how wide the pixel asking is, in radians, so a surface knows how
// much of itself is being asked for; zero for a lone ray.
func shade(sc staged, r maths.Ray, pixel float64, bounce int) frame.Pixel {
	var px frame.Pixel
	if f, enter, ok := entering(sc.fs, r); ok {
		// a surface before the field is the picture as usual
		p, dist, hit := first(sc.ps, r, 0)
		if !hit || dist > enter {
			return bent(sc, r, f, enter, pixel, bounce)
		}
		px = surfaceAt(sc, r, pixel, bounce, p, dist)
	} else {
		px = surface(sc, r, pixel, bounce)
	}
	if len(sc.vs) == 0 {
		return px
	}
	far := float64(px.Depth)
	keep, added := through(sc, r, far)
	if keep == maths.Grey(1) && added == (maths.Light{}) {
		return px
	}
	l := px.Light().Times(keep).Add(added)
	px.R, px.G, px.B = float32(l.R), float32(l.G), float32(l.B)
	px.Cover = float32(1 - (1-float64(px.Cover))*most(keep))
	return px
}

// surface is what one ray brings back from the surfaces alone: the first
// thing it meets, lit by every lamp that can see that point through the
// volumes in the way, or the sky.
func surface(sc staged, r maths.Ray, pixel float64, bounce int) frame.Pixel {
	p, dist, ok := first(sc.ps, r, 0)
	if !ok {
		l := air(sc.w, r.Dir, math.Inf(1), sky(sc.w, r.Dir))
		return pixelOf(l, 0, float32(math.Inf(1)))
	}
	return surfaceAt(sc, r, pixel, bounce, p, dist)
}

// surfaceAt is surface for a ray already known to meet p at dist.
func surfaceAt(sc staged, r maths.Ray, pixel float64, bounce int, p placed,
	dist float64) frame.Pixel {
	point := r.Point(dist)
	local := p.at.In(point)
	normal := p.at.Orientation.Out(shape.Normal(p.thing.Shape, local)).Unit()
	if normal.Dot(r.Dir) > 0 {
		normal = normal.Neg() // seen from inside, or from beneath the ground
	}
	m := p.thing.Material
	if m == nil {
		m = grey
	}
	colour := m.Colour
	if p.tex != nil {
		// the pixel's width where it lands, in the thing's own units
		colour = texture.Sample(p.tex, local, dist*pixel/p.scale)
	}
	sum := emits(m, p.shine, local)
	for _, lamp := range sc.w.Lamps {
		toward, far, light := lamp.Light(point, normal)
		if toward == (maths.Vec{}) {
			// from everywhere, already weighed for this surface
			sum = sum.Add(colour.Times(light))
			continue
		}
		square := normal.Dot(toward) // 1 facing it, 0 edge-on, below 0 turned away
		if square <= 0 {
			continue
		}
		from := sc.lifted(point, normal, r.At)
		if blocked(sc.ps, maths.Ray{At: from, Dir: toward}, far) {
			continue
		}
		reach := square * clear(sc.vs, from, toward)
		l := colour.Times(light).Scale(reach)
		if hasGas(sc.vs) {
			l = l.Times(gasClear(sc.vs, from, toward))
		}
		sum = sum.Add(l)
	}
	// a glossy surface mirrors whatever the eye would see along the
	// mirrored ray, once, by the share fresnel gives at this look
	if share := m.Mirrors(-normal.Dot(r.Dir)); share > 0 && bounce < bounces {
		mirrored := r.Dir.Sub(normal.Scale(2 * r.Dir.Dot(normal))).Unit()
		from := sc.lifted(point, normal, r.At)
		seen := shade(sc, maths.Ray{At: from, Dir: mirrored}, pixel,
			bounce+1)
		sum = sum.Scale(1 - share).Add(seen.Light().Scale(share))
	}
	return pixelOf(air(sc.w, r.Dir, dist, sum), 1, float32(dist))
}

// sky is the world's sky in the direction toward, or no light without one.
func sky(w *thing.World, toward maths.Vec) maths.Light {
	if w.Sky == nil {
		return maths.Light{}
	}
	return w.Sky.Light(toward.Unit())
}

// air is light that has come dist through the world's haze toward the eye,
// along toward: some of it, and the rest the haze's own colour.
func air(w *thing.World, toward maths.Vec, dist float64,
	l maths.Light) maths.Light {
	if w.Haze.Density <= 0 {
		return l
	}
	haze := w.Haze.Colour
	if w.Haze.FromSky {
		haze = sky(w, toward)
	}
	through := math.Exp(-w.Haze.Density * dist)
	return l.Scale(through).Add(haze.Scale(1 - through))
}

// Trace is what one ray brings back from the world at time t: a lone
// ray, with no pixel round it.
func Trace(w *thing.World, r maths.Ray, t time.Duration) frame.Pixel {
	return shade(stage(w, t), r, 0, 0)
}

// pixelAngle is how wide one pixel of a picture is, in radians: the
// angle between the rays through the middle of the picture and the pixel
// beside it.
func pixelAngle(eye maths.Subspace, lens film.Sensor, width, height int) float64 {
	a := film.Look(eye, lens, width/2, height/2, width, height, maths.Band{})
	b := film.Look(eye, lens, width/2+1, height/2, width, height, maths.Band{})
	return math.Acos(math.Max(-1, math.Min(1, a.Dir.Dot(b.Dir))))
}

// Render is the world at time t, seen from eye through lens, as a frame of
// width by height pixels. Rows are drawn on every core at once; each pixel
// depends on nothing but the world, so the frame is the same however the
// rows are shared out.
func Render(w *thing.World, eye maths.Subspace, lens film.Sensor,
	t time.Duration, width, height int) *frame.Frame {
	sc := stage(w, t)
	pixel := pixelAngle(eye, lens, width, height)
	if w.Pixel > 0 {
		pixel = w.Pixel
	}
	f := frame.New(width, height)
	rows := make(chan int, height)
	for y := 0; y < height; y++ {
		rows <- y
	}
	close(rows)
	var wg sync.WaitGroup
	for range max(1, runtime.NumCPU()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for y := range rows {
				for x := 0; x < width; x++ {
					r := film.Look(eye, lens, x, y, width, height,
						maths.Band{})
					f.Set(x, y, shade(sc, r, pixel, 0))
				}
			}
		}()
	}
	wg.Wait()
	return f
}

// Sampled is Render with n by n rays through every pixel, averaged as light,
// so each pixel is the light across its whole area and not the light at its
// middle. A detail finer than a pixel, like a checker near the horizon, then
// comes out as its average instead of as a pattern of its own.
//
// At n of one or less it is Render.
func Sampled(w *thing.World, eye maths.Subspace, lens film.Sensor,
	t time.Duration, width, height, n int) *frame.Frame {
	if n <= 1 {
		return Render(w, eye, lens, t, width, height)
	}
	fine := Render(w, eye, lens, t, width*n, height*n)
	f := frame.New(width, height)
	k := 1 / float32(n*n)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			var p frame.Pixel
			p.Depth = float32(math.Inf(1))
			for sy := 0; sy < n; sy++ {
				for sx := 0; sx < n; sx++ {
					q := fine.At(x*n+sx, y*n+sy)
					p.R += q.R
					p.G += q.G
					p.B += q.B
					p.Cover += q.Cover
					p.Depth = min(p.Depth, q.Depth)
				}
			}
			p.R, p.G, p.B, p.Cover = p.R*k, p.G*k, p.B*k, p.Cover*k
			f.Set(x, y, p)
		}
	}
	return f
}

// passes is how many things a ray may pass through in a band that gets
// through them, before what is behind is taken as dark.
const passes = 8

// RenderIn is Render in one band of the spectrum: what a sensor for
// that band would record, a narrow filter's picture even within the
// visible. Nothing is lit by lamps; what shows is what gives off the
// band, through what passes it.
//
// A thing whose material has no Radiant is opaque and dark; one with a
// Radiant passes the band by its Through to the power of the path
// through it, and adds what it Emits.
//
// A volume with a Radiant does the same wherever it has any density,
// and one without thins the band as it thins light. The sky is dark.
// The pixel's light is the band's strength, the same in all three
// parts. The zero band is the whole visible at once, which is Render.
func RenderIn(w *thing.World, eye maths.Subspace, lens film.Sensor,
	t time.Duration, width, height int, band maths.Band) *frame.Frame {
	if band.Nanometres == 0 {
		// the zero band is the whole visible at once
		return Render(w, eye, lens, t, width, height)
	}
	// nothing bends in a band: the fields are left out
	sc := staged{w: w, ps: place(w, t), vs: placeVolumes(w, t),
		spread: tracer.Spread(w, t)}
	f := frame.New(width, height)
	rows := make(chan int, height)
	for y := 0; y < height; y++ {
		rows <- y
	}
	close(rows)
	var wg sync.WaitGroup
	for range max(1, runtime.NumCPU()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for y := range rows {
				for x := 0; x < width; x++ {
					r := film.Look(eye, lens, x, y, width, height,
						band)
					f.Set(x, y, inBand(sc, r, band, passes))
				}
			}
		}()
	}
	wg.Wait()
	return f
}

// inBand is what one ray brings back in a band outside the visible: the
// first thing met gives off what it gives off, echoes what radars reach
// it, and passes what it passes of what lies behind, and so on through
// what passes the band.
func inBand(sc staged, r maths.Ray, band maths.Band, left int) frame.Pixel {
	p, dist, ok := first(sc.ps, r, 0)
	far := math.Inf(1)
	if ok {
		far = dist
	}
	keep, added := throughIn(sc, r, far, band)
	px := frame.Pixel{Depth: float32(far), Cover: float32(1 - keep)}
	strength := added
	if ok {
		px.Cover = 1
		m := p.thing.Material
		if m != nil && m.Radiant != nil {
			strength += keep * m.Radiant.Emits(band)
			strength += keep * echo(sc, p, r, dist, band)
			if through := m.Radiant.Through(band); through > 0 && left > 0 {
				if sp, can := p.thing.Shape.(shape.Spanner); can {
					local := p.at.RayIn(r)
					enter, leave, crossed := sp.Span(local, 0, nowhere)
					if crossed {
						path := leave - enter
						past := leave * p.scale
						end := r.Point(past)
						off := sc.lifting(end, r.At)
						out := r.Point(past + off)
						next := maths.Ray{At: out, Dir: r.Dir}
						behind := inBand(sc, next, band, left-1)
						strength += keep * math.Pow(through, path) *
							float64(behind.R)
					}
				}
			}
		}
	}
	px.R, px.G, px.B = float32(strength), float32(strength), float32(strength)
	return px
}

// echo is what a thing scatters back of the radars that reach it in a
// band: for each beam, how squarely it lands times how rough the thing
// is there times what it returns, if no surface stands in the way and
// through whatever volumes do.
func echo(sc staged, p placed, r maths.Ray, dist float64,
	band maths.Band) float64 {
	m := p.thing.Material
	back := m.Radiant.Scatters(band)
	if back <= 0 {
		return 0
	}
	point := r.Point(dist)
	local := p.at.In(point)
	normal := p.at.Orientation.Out(shape.Normal(p.thing.Shape, local)).Unit()
	if normal.Dot(r.Dir) > 0 {
		normal = normal.Neg()
	}
	// how rough the thing is there, as its roughness is at the
	// picture's time: rough everywhere alike without one
	rough := 1.0
	if p.rough != nil {
		rough = p.rough.At(local).Luminance()
	}
	sum := 0.0
	for _, lamp := range sc.w.Lamps {
		b, is := lamp.(film.Beamer)
		if !is {
			continue
		}
		strength := b.Beams(band)
		if strength <= 0 {
			continue
		}
		toward, far, _ := lamp.Light(point, normal)
		square := normal.Dot(toward)
		if square <= 0 {
			continue
		}
		from := sc.lifted(point, normal, r.At)
		if blocked(sc.ps, maths.Ray{At: from, Dir: toward}, far) {
			continue
		}
		shadow := maths.Ray{At: from, Dir: toward}
		keep, _ := throughIn(sc, shadow, far, band)
		sum += back * rough * square * strength * keep
	}
	return sum
}

// throughIn is what the ray gathers from the volumes in a band outside
// the visible: how much of the band from far gets through, and how much
// the volumes give off in front of it.
func throughIn(sc staged, r maths.Ray, far float64,
	band maths.Band) (keep, added float64) {
	keep = 1
	cs := crossings(sc.vs, r, far)
	st := stepping(cs, strides)
	for keep > faint {
		k, d, step, ok := st.take()
		if !ok {
			break
		}
		c := cs[k]
		m := c.v.v.Material
		local := c.local.Point(d / c.v.scale)
		density := c.v.med.Density(local)
		if density <= 0 {
			continue
		}
		if g := c.v.gas; g != nil {
			// a gas passes the band by its own extinction there,
			// and gives off nothing in it: its lines are not wired
			// yet
			keep *= math.Exp(-g.Extinction(band) * density * step /
				c.v.scale)
			continue
		}
		absorb := 1 - math.Exp(-density*step/c.v.scale)
		if m != nil && m.Radiant != nil {
			// what the gas gives off in the band, where it is, as
			// much as the gas there absorbs
			emit := m.Radiant.Emits(band) *
				boosted(c.v, local, r.Dir)
			if l, is := m.Radiant.(thing.Lit); is {
				// an ionisation front: the gas glows where the
				// lamps reach it, as its line wants
				at := r.Point(d)
				light := shone(sc, at, r.Dir.Neg())
				emit *= math.Pow(light, l.Excitation(band))
			}
			added += keep * absorb * emit
			// the band passes by through to the power of the path
			// at unit density: thin gas is clear
			through := m.Radiant.Through(band)
			absorb = 1 - math.Pow(through, density*step/c.v.scale)
		}
		keep *= 1 - absorb
	}
	return keep, added
}
