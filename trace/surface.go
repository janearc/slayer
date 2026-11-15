package trace

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// meet is how far along a ray, between near and far, it first meets this
// thing, in the world's own distances.
func (placed *placedThing) meet(ray maths.Ray, near,
	far float64) (float64, bool) {
	local := placed.at.RayIn(ray)
	nearLocal, farLocal := near/placed.scale, far/placed.scale
	solid := placed.thing.Shape
	var reached float64
	var ok bool
	if hitter, exact := solid.(shape.Hitter); exact {
		reached, ok = hitter.Hit(local, nearLocal, farLocal)
	} else {
		reached, ok = march(solid, local, nearLocal, farLocal)
	}
	along := reached * placed.scale
	return along, ok
}

// landing is where on a mesh of slayer's a ray landed, when it met one:
// which face, and where on it, so its normal needs no search.
type landing struct {
	met     mesh.Met
	meshed  bool
	normal  maths.Vec
	grouped bool
}

// reach is meet, and where the ray landed when the thing is slayer's
// mesh: the same distance, worked the same way, so a search by reach
// finds what a search by meet finds.
func (placed *placedThing) reach(ray maths.Ray, near,
	far float64) (float64, landing, bool) {
	if group, ok := placed.thing.Shape.(*shape.MeshGroup); ok {
		local := placed.at.RayIn(ray)
		distance, normal, hit := group.HitNormal(local, near/placed.scale, far/placed.scale)
		return distance * placed.scale, landing{normal: normal, grouped: hit}, hit
	}
	skin, isMesh := placed.thing.Shape.(*mesh.Mesh)
	if !isMesh {
		along, ok := placed.meet(ray, near, far)
		return along, landing{}, ok
	}
	local := placed.at.RayIn(ray)
	met, ok := skin.Meet(local, near/placed.scale, far/placed.scale)
	return met.Along() * placed.scale, landing{met: met, meshed: ok}, ok
}

// march steps along the ray by the distance to the nearest surface
// until it is close enough to count, and is a miss if it leaves the
// shape's box or takes marchSteps steps without arriving.
//
// A step that long cannot pass through the surface if the shape's
// Distance is a true distance; a shape that says more than the truth
// can be stepped over.
//
// Close enough is touch's share of the shape's reach, or of the
// point's distance from the origin where that is more, so a shape with
// no edge is measured where the march is.
func march(solid shape.Shape, ray maths.Ray, near,
	far float64) (float64, bool) {
	box := solid.Bounds()
	enter, leave, ok := box.Hit(ray, near, far)
	if !ok {
		return 0, false
	}
	size := reach(box)
	if math.IsInf(size, 0) {
		size = 0
	}
	along := enter
	for i := 0; i < marchSteps && along <= leave; i++ {
		at := ray.Point(along)
		gap := solid.Distance(at)
		if math.Abs(gap) < touch*max(size, at.Len()) {
			return along, true
		}
		along += math.Abs(gap)
	}
	return 0, false
}

// hit is a ray meeting a thing: the thing, placed, how far along the
// ray it is, in the world's distances, and where it landed on a mesh.
type hit struct {
	thing placedThing
	dist  float64
	on    landing
}

// normal is the way the thing faces where the ray met it, turned
// against the ray: the face it landed on, for slayer's mesh, or the
// shape asked at the point.
func (met hit) normal(point, dir maths.Vec) maths.Vec {
	if met.on.grouped {
		return met.thing.toward(met.on.normal, dir)
	}
	skin, isMesh := met.thing.thing.Shape.(*mesh.Mesh)
	if !isMesh || !met.on.meshed {
		return met.thing.facing(point, dir)
	}
	return met.thing.toward(skin.NormalAt(met.on.met), dir)
}

// lifted is where the light at a point the ray met is asked from: on
// slayer's mesh shaded smooth, the point raised to the curved surface
// its normals draw, so a light grazing it is not cut into teeth by the
// faces beside it; anywhere else, the point.
func (met hit) lifted(point maths.Vec) maths.Vec {
	skin, isMesh := met.thing.thing.Shape.(*mesh.Mesh)
	if !isMesh || !met.on.meshed {
		return point
	}
	at := met.thing.at
	return at.Out(skin.Lifted(met.on.met, at.In(point)))
}

// look is one ray asked of the world, and what travels with it: the ray, how
// wide the pixel asking is, in radians, so a surface knows how much of itself
// is being asked for, zero for a lone ray, and how many mirrors the ray has
// come off already.
//
// come is how far the look had come before its ray began, where nothing else
// accounts for that way: the earlier steps of a march through a field. A
// surface is that much further from the eye than it is along the ray, for the
// detail it is asked at and the haze over it.
type look struct {
	ray    maths.Ray
	pixel  float64
	bounce int
	come   float64
	// onward is whether the ray is a later leg of the look, off a
	// mirror or out of a field's march, and so not the eye's own
	onward bool
	// crossed is how many clear surfaces the look has bent through, and
	// slipped how many it has passed that were no boundary at all.
	//
	// inside is the clear media the look is in, in the order it
	// entered them, and holding how many of them there are: what a clear
	// surface has on each side of it is worked out from these. A medium
	// is an index and a rank, not a thing: two sheets of one glass are
	// its way in and its way out.
	crossed int
	slipped int
	inside  [mostHeld]held
	holding int
}

// held is a clear medium a look is inside: its index, and its rank among
// clear things that overlap. Things of one index and one rank are one
// medium: the two faces of a pane or of a layer of water, each a sheet
// and a thing of its own, are the way into and the way out of the same
// glass or the same water.
type held struct {
	index float64
	rank  int
}

// of is the medium a clear material is.
func of(material *thing.Material) held {
	return held{index: material.Dense(), rank: material.Rank}
}

const (
	// mostHeld is how many clear media a look can be inside at once.
	// Picked: twice what it can bend through, for the media it enters
	// without bending.
	mostHeld = 2 * refractions
	// slips is how many clear surfaces that are no boundary a look may
	// pass straight through. Picked: twice what it can bend through; a
	// glass of water wants two, one into the water inside the glass's
	// wall and one out of it.
	slips = 2 * refractions
	// air is the index of what a look is in when it is in no clear thing.
	air = 1.0
)

// in is whether the look is inside a clear medium.
func (seen look) in(one held) bool {
	for _, each := range seen.inside[:seen.holding] {
		if each == one {
			return true
		}
	}
	return false
}

// medium is what the look is in, and whether that is anything but the
// air: of the clear media it is inside, the one of the highest rank, and
// among equals the one it entered last. Inside none it is in the air.
func (seen look) medium() (held, bool) {
	in := held{index: air}
	for at, each := range seen.inside[:seen.holding] {
		if at == 0 || each.rank >= in.rank {
			in = each
		}
	}
	return in, seen.holding > 0
}

// without is the look no longer inside a medium, the rest in their
// order. It leaves it once: a look that entered the same medium twice,
// through two things of it, is still inside it once.
func (seen look) without(one held) look {
	kept, left := 0, false
	for _, each := range seen.inside[:seen.holding] {
		if each != one || left {
			seen.inside[kept] = each
			kept++
			continue
		}
		left = true
	}
	for at := kept; at < seen.holding; at++ {
		seen.inside[at] = held{}
	}
	seen.holding = kept
	return seen
}

// boundary is what a clear surface is to a look that meets it.
//
// relative is the index of what the light goes into over the index of
// what it comes from, which is all Snell and Fresnel ask. real is whether
// the surface is a boundary at all: it is not when the thing the look
// enters is outranked by one it is already in, nor when the thing it
// leaves was not the medium it was in, and then the light goes straight
// on, with nothing mirrored and nothing bent.
//
// entering is whether the
// look is going into the thing or coming out of it.
//
// With one clear thing in the air this is what the tracer always did:
// its index going in and the inverse coming out.
type boundary struct {
	relative float64
	real     bool
	entering bool
}

// meets is the boundary a clear surface of a thing is to the look.
func (seen look) meets(material *thing.Material) boundary {
	one := of(material)
	before, some := seen.medium()
	if seen.in(one) {
		after, _ := seen.without(one).medium()
		return boundary{relative: after.index / before.index,
			real: before == one}
	}
	if some && one.rank < before.rank {
		// outranked by what the look is already in: no boundary
		return boundary{relative: 1, entering: true}
	}
	return boundary{relative: one.index / before.index, real: true,
		entering: true}
}

// follows is whether the look can go on past a clear surface: a true
// boundary costs one of its bends, a surface that is none one of its slips, and
// a thing it enters must have room among the things it is inside.
func (seen look) follows(at boundary) bool {
	if at.entering && seen.holding >= mostHeld {
		return false
	}
	if at.real {
		return seen.crossed < refractions
	}
	return seen.slipped < slips
}

// along is the same look down another ray, having come so far: a
// segment of a bent march, or the way out of a field.
func (seen look) along(ray maths.Ray, come float64) look {
	seen.ray, seen.come, seen.onward = ray, come, true
	return seen
}

// mirrored is the look off a mirror, down the mirrored ray: one bounce
// more. It has come no way that is not accounted for, since the
// mirror's own surface is hazed over the way to it.
func (seen look) mirrored(ray maths.Ray) look {
	seen.ray, seen.come, seen.onward = ray, 0, true
	seen.bounce++
	return seen
}

// passed is the look through a clear surface of a thing, down the ray it
// leaves by: inside the thing now if it was not, and out of it if it
// was; one bend more when the surface was a boundary, one slip more when
// it was not.
func (seen look) passed(ray maths.Ray, material *thing.Material,
	at boundary) look {
	seen.ray, seen.come, seen.onward = ray, 0, true
	if at.real {
		seen.crossed++
	} else {
		seen.slipped++
	}
	if !at.entering {
		return seen.without(of(material))
	}
	seen.inside[seen.holding] = of(material)
	seen.holding++
	return seen
}

// turnedBack is the look a clear surface reflected whole, past the
// critical angle: one bend more, and inside whatever it was inside.
func (seen look) turnedBack(ray maths.Ray) look {
	seen.ray, seen.come, seen.onward = ray, 0, true
	seen.crossed++
	return seen
}

// shade is what one look brings back: the first thing its ray meets,
// lit by every lamp that can see that point, or the sky, with whatever
// volumes lie in front of it added over.
func shade(scene *staged, seen look) frame.Pixel {
	ray := seen.ray
	told := scene.walked
	if told != nil {
		// the steps of a mirror's look are told at its depth
		before := told.bounce
		told.bounce = seen.bounce
		defer func() { told.bounce = before }()
	}
	var drawn frame.Pixel
	if into, enters := entering(scene.fields, ray); enters {
		// a surface before the field is the picture as usual
		met, found := scene.crowd.first(ray, 0)
		if !found || met.dist > into.dist {
			if told != nil {
				told.tell("the ray enters a field", "at",
					into.dist)
			}
			return bent(scene, seen, into)
		}
		drawn = surfaceAt(scene, seen, met)
	} else {
		drawn = surface(scene, seen)
	}
	if len(scene.volumes) == 0 {
		return drawn
	}
	gathered := through(scene, ray, float64(drawn.Depth))
	if told != nil {
		told.tell("the volumes along the ray", "keep", gathered.keep,
			"add", gathered.added)
	}
	if gathered == untouched() {
		return drawn
	}
	light := drawn.Light().Times(gathered.keep).Add(gathered.added)
	open := (1 - float64(drawn.Cover)) * most(gathered.keep)
	return pixelOf(light, float32(1-open), drawn.Depth)
}

// surface is what one ray brings back from the surfaces alone: the first
// thing it meets, lit by every lamp that can see that point through the
// volumes in the way, or the sky.
func surface(scene *staged, seen look) frame.Pixel {
	ray := seen.ray
	met, found := scene.crowd.first(ray, 0)
	if !found {
		endless := math.Inf(1)
		beyond := scene.skyAt(ray.Dir)
		// and the sun, where it is drawn as a disc and the ray finds it
		if disc := film.Discs(scene.discs, ray.Dir); disc != (maths.Light{}) {
			beyond = beyond.Add(disc)
		}
		sky := scene.air(ray.Dir, endless, beyond)
		if told := scene.walked; told != nil {
			told.tell("the ray meets nothing, so the sky", "along",
				ray.Dir, "light", sky)
		}
		return pixelOf(sky, 0, float32(endless))
	}
	return surfaceAt(scene, seen, met)
}

// named is what a look met, for a replay to say: its material, its
// shape, and for slayer's mesh the face.
func (met hit) named() string {
	said := fmt.Sprintf("%T", met.thing.thing.Shape)
	if material := met.thing.thing.Material; material != nil &&
		material.Name != "" {
		said = material.Name + " " + said
	}
	if met.on.meshed {
		said += fmt.Sprintf(" face %d", met.on.met.Face())
	}
	return said
}

// surfaceAt is surface for a look already known to meet a thing.
func surfaceAt(scene *staged, seen look, met hit) frame.Pixel {
	ray, placed, dist := seen.ray, met.thing, met.dist
	point := ray.Point(dist)
	local := placed.at.In(point)
	normal := met.normal(point, ray.Dir)
	material := placed.thing.Material
	if material == nil {
		material = grey
	}
	// a wrinkled surface's face tilts from place to place
	if material.Crinkle != nil {
		normal = material.Crinkle.Crinkled(point, normal, ray.Dir.Neg())
	}
	// how far the point is from the eye: along this ray, and whatever
	// way the look had come before it
	away := seen.come + dist
	// the pixel's width where it lands, in the thing's own units
	footprint := away * seen.pixel / placed.scale
	colour := material.Colour
	if placed.paint != nil {
		colour = texture.Sample(placed.paint, local, footprint)
	}
	told := scene.walked
	if told != nil {
		told.tell("the ray meets a thing", "what", met.named(),
			"along", dist, "point", point, "normal", normal,
			"colour", colour)
	}
	emitted := emits(material, placed.shine, local, footprint)
	lifted := met.lifted(point)
	lit := scene.lamplight(lifted, normal, colour)
	// a replay tells the sky's share, so it asks for it here. A picture
	// asks only where the surface keeps its own colour, below: sixteen
	// rays cast for light that is multiplied by zero are not cast.
	var skylit maths.Light
	if told != nil {
		skylit = scene.skylight(lifted, normal, colour)
	}
	sum := emitted.Add(lit).Add(skylit)
	if told != nil {
		told.tell("its own light, the lamps' and the sky's", "emits",
			emitted, "lamps", lit, "sky", skylit, "sum", sum)
	}
	// a glossy surface mirrors whatever the eye would see along the
	// mirrored ray, once, by the share fresnel gives at this look. a
	// clear one bends what lies along the refracted ray in by snell's
	// law: fresnel's share mirrors first, Through says how much of the
	// rest passes, and whatever neither takes lights as the surface
	// itself.
	//
	// a clear surface's share is the unpolarised fresnel reflectance
	// between the air and the thing, by the side the ray arrives from.
	// past the critical angle the bend is a reflection and all of the
	// light reflects: total internal reflection. off a look's first
	// surface the mirror carries it; on a bent leg the bend does, whole.
	//
	// the two stay inside what the kernel can unroll: a mirrored look
	// does not bend, a bent look does not mirror again, and a look
	// bends through at most refractions surfaces.
	//
	// so a partial reflection on an already bent look is paid and not
	// followed: the light is taken from what passes and nothing carries
	// it back. a water ball in a surround of luminance one reads 0.98
	// head on and 0.76 at 85 degrees of incidence, near the rim, where
	// whole light would read one; see TestAWaterBallInASurround.
	//
	// every look starts in the air: a camera inside water is drawn as
	// if it were in the air.
	//
	// a clear surface that reflects all of what arrives, past the
	// budget or off a mirrored look, has nothing to follow it and
	// lights as the opaque surface it is.
	//
	// each colour has its own share, which only a metal makes differ: a
	// metal's comes from its complex index, a clear surface's is
	// fresnel's, the same in all three.
	//
	// a clear thing may stand inside another, water in a glass: what a
	// surface has on each side of it is then not the air and the thing,
	// and the look's boundary says what it is. where the thing entered
	// is outranked by one the look is already in, or the thing left was
	// not what the look was in, the surface is no boundary: nothing
	// mirrors and nothing bends, and the light goes straight on.
	reflects := material.Reflects(-normal.Dot(ray.Dir))
	var at boundary
	if material.Through > 0 {
		at = seen.meets(material)
	}
	if material.Through > 0 && material.Metal == nil {
		reflects = maths.Light{}
		if at.real {
			reflects = maths.Grey(material.SplitsBetween(
				-normal.Dot(ray.Dir), at.relative))
		}
	}
	share := math.Max(reflects.R, math.Max(reflects.G, reflects.B))
	through := math.Min(math.Max(material.Through, 0), 1)
	var bent maths.Vec
	var total bool
	if through > 0 && seen.bounce == 0 && seen.follows(at) {
		bent = ray.Dir
		if at.real {
			bent, total = refracted(ray.Dir, normal, at.relative)
		}
		if total {
			bent = ray.Dir.Sub(normal.Scale(2 * ray.Dir.Dot(normal))).Unit()
			share, reflects = 1, maths.Grey(1)
		}
	} else {
		through = 0
		// a clear surface that reflects all of what arrives, where
		// nothing can follow it, lights as the opaque surface it is
		if material.Through > 0 && share >= 1 {
			share, reflects = 0, maths.Light{}
		}
	}
	// an unfollowed mirror keeps the surface whole, as it always has;
	// on a bent leg fresnel's share is a toll light pays and does not
	// come back, which is why two panes dim twice.
	if share > 0 && seen.crossed == 0 && seen.bounce >= bounces {
		share, reflects = 0, maths.Light{}
	}
	follow := share > 0 && seen.crossed == 0
	if share > 0 || through > 0 {
		// each colour keeps what it does not mirror and does not let
		// through: none of it, for a mirror of all that arrives or a
		// pane that passes all of it; for all but a metal, the same
		// product in all three
		left := maths.RGB((1-reflects.R)*(1-through),
			(1-reflects.G)*(1-through), (1-reflects.B)*(1-through))
		if told == nil && left != (maths.Light{}) {
			sum = sum.Add(scene.skylight(lifted, normal, colour))
		}
		kept := sum.Times(left)
		if follow {
			off := shade(scene, seen.mirrored(scene.bounced(ray,
				point, normal)))
			kept = kept.Add(off.Light().Times(reflects))
			if told != nil {
				told.tell("a mirror, by fresnel's share", "share",
					reflects, "brought", off.Light())
			}
		}
		// total internal reflection is the mirror's where the mirror is
		// followed, and the bend's own, whole, where it is not
		passes := maths.RGB((1-reflects.R)*through,
			(1-reflects.G)*through, (1-reflects.B)*through)
		if total {
			passes = maths.Grey(1)
			if follow {
				passes = maths.Light{}
			}
		}
		if passes != (maths.Light{}) {
			// a reflected bend stays on the ray's side of the
			// surface, and inside whatever it was inside
			side := normal.Scale(-1)
			if total {
				side = normal
			}
			enter := maths.Ray{At: scene.liftedAlong(point, side,
				ray.At), Dir: bent}
			beyond := seen.passed(enter, material, at)
			if total {
				beyond = seen.turnedBack(enter)
			}
			past := shade(scene, beyond)
			kept = kept.Add(past.Light().Times(passes))
			if told != nil {
				told.tell("a clear surface, bent by snell", "through",
					passes, "brought", past.Light())
			}
		}
		sum = kept
	} else if told == nil {
		sum = sum.Add(scene.skylight(lifted, normal, colour))
	}
	// the lamps' highlights, over what it keeps, mirrors and passes
	if material.Satin > 0 {
		glint := scene.glint(lifted, normal, ray.Dir.Neg(), material)
		sum = sum.Add(glint)
		if told != nil {
			told.tell("the lamps' highlights", "glint", glint)
		}
	}
	hazed := scene.air(ray.Dir, away, sum)
	return pixelOf(hazed, 1, float32(dist))
}

// lamplight is what the lamps light a point of a surface with: the
// surface's colour there times each lamp's light, by how squarely the
// lamp lands on it and how much of the lamp gets there, each lamp the
// point asks counted as many times over as asked says.
//
// A lamp that lights from everywhere has already weighed itself for
// the surface.
func (scene *staged) lamplight(point, normal maths.Vec,
	colour maths.Light) maths.Light {
	var sum maths.Light
	told := scene.walked
	squarely := func(toward maths.Vec, light maths.Light) float64 {
		return light.Luminance() * math.Max(0, normal.Dot(toward))
	}
	var room [lampsAsked]lit
	for _, one := range scene.asked(room[:], point, normal, squarely) {
		i, toward, far := one.index, one.toward, one.far
		light := one.light.Scale(one.counts)
		if one.ambient() {
			sum = sum.Add(colour.Times(light))
			if told != nil {
				told.tell("a lamp lighting from everywhere",
					"lamp", i, "light", light)
			}
			continue
		}
		// one facing the lamp, zero edge-on, less than zero turned
		// away; and a lamp that sends nothing here needs no march
		square := normal.Dot(toward)
		if square <= 0 || light == (maths.Light{}) {
			if told != nil {
				told.tell("a lamp that does not light it",
					"lamp", i, "square", square,
					"light", light)
			}
			continue
		}
		from := scene.lifted(point, normal)
		reached := scene.unhidden(i, from, toward, far)
		// a matte surface sends what lands on it out every way alike:
		// its radiance is its colour over pi times the irradiance
		matte := colour.Times(light).Scale(square / math.Pi)
		sum = sum.Add(matte.Times(reached))
		if told != nil {
			told.tell("a lamp", "lamp", i, "toward", toward,
				"far", far, "light", light, "square", square,
				"reached", reached)
		}
	}
	return sum
}

// glint is the highlights a surface's lamps leave on it, seen along
// look: for each lamp that reaches it, the microfacet lobe of a surface
// as rough as its Satin, by the share of each colour it mirrors at the
// way half between the lamp and the eye.
//
// Not the surface's colour: a highlight is the lamp's own light, tinted
// only by what the surface mirrors, so gold's is gold and gloss paint's
// is white. Lamps that light from everywhere leave none.
func (scene staged) glint(point, normal, look maths.Vec,
	material *thing.Material) maths.Light {
	satin := math.Min(material.Satin, 1)
	viewed := normal.Dot(look)
	if !(satin > 0) || !(viewed > 0) {
		return maths.Light{}
	}
	alpha := satin * satin
	var sum maths.Light
	squarely := func(toward maths.Vec, light maths.Light) float64 {
		return light.Luminance() * math.Max(0, normal.Dot(toward))
	}
	var room [lampsAsked]lit
	for _, one := range scene.asked(room[:], point, normal, squarely) {
		light := one.light.Scale(one.counts)
		square := normal.Dot(one.toward)
		if one.ambient() || square <= 0 || light == (maths.Light{}) {
			continue
		}
		half := one.toward.Add(look).Unit()
		lobe := thing.Microfacet(normal.Dot(half), square, viewed, alpha)
		if !(lobe > 0) {
			continue
		}
		from := scene.lifted(point, normal)
		reached := scene.unhidden(one.index, from, one.toward, one.far)
		mirrored := material.Reflects(half.Dot(look))
		sum = sum.Add(light.Times(reached).Times(mirrored).Scale(lobe))
	}
	return sum
}

// unhidden is how much of lamp i's light gets from a point to it along
// toward, through whatever stands between: and for a sun drawn as a
// disc, the share of its disc unhidden, sought along the ways across it
// its Spread gives, so its shadows soften by its width.
func (scene staged) unhidden(i int, from, toward maths.Vec,
	far float64) maths.Light {
	if sun, ok := scene.lamps[i].(film.Sun); ok && sun.Disc && sun.Size > 0 {
		ways := sun.Spread()
		var sum maths.Light
		for _, way := range ways {
			sum = sum.Add(scene.transmit(from, way, far))
		}
		return sum.Scale(1 / float64(len(ways)))
	}
	return scene.transmit(from, toward, far)
}

// refracted is the way a clear surface bends an arriving ray, by
// snell's law. relative is the index of what the ray goes into over the
// index of what it comes from: a thing's index going into it from the
// air, the inverse coming back out, and for water against glass the one
// over the other.
//
// The normal faces the ray, as slayer's normals do. A
// ray too oblique to pass into something less dense does not bend at
// all: total internal reflection, and the caller mirrors it.
func refracted(dir, normal maths.Vec,
	relative float64) (bent maths.Vec, total bool) {
	ratio := 1 / relative
	cosine := -normal.Dot(dir)
	squared := ratio * ratio * (1 - cosine*cosine)
	if squared >= 1 {
		return maths.Vec{}, true
	}
	along := ratio*cosine - math.Sqrt(1-squared)
	return dir.Scale(ratio).Add(normal.Scale(along)).Unit(), false
}

// bounced is the ray a mirror at a point sends an arriving ray off
// along: turned about the normal, and lifted off the surface so it
// does not meet the point it left.
func (scene *staged) bounced(ray maths.Ray, point,
	normal maths.Vec) maths.Ray {
	way := ray.Dir.Sub(normal.Scale(2 * ray.Dir.Dot(normal))).Unit()
	return maths.Ray{At: scene.liftedAlong(point, normal, ray.At),
		Dir: way}
}

// skyAt is the sky's light from the direction toward, or no light
// with no sky.
func (scene *staged) skyAt(toward maths.Vec) maths.Light {
	if scene.sky == nil {
		return maths.Light{}
	}
	return scene.sky.Light(toward.Unit())
}

// air is light that has come dist through the haze toward the eye,
// along the ray's own direction: some of it, and the rest the haze's
// own colour, or the sky's that way.
func (scene *staged) air(along maths.Vec, dist float64,
	light maths.Light) maths.Light {
	if scene.haze.Density <= 0 {
		return light
	}
	haze := scene.haze.Colour
	if scene.haze.FromSky {
		haze = scene.skyAt(along)
	}
	kept := math.Exp(-scene.haze.Density * dist)
	hazed := light.Scale(kept).Add(haze.Scale(1 - kept))
	if told := scene.walked; told != nil {
		told.tell("the haze", "through", dist, "kept", kept, "haze",
			haze, "light", hazed)
	}
	return hazed
}
