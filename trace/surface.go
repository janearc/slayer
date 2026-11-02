package trace

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/texture"
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
	met    mesh.Met
	meshed bool
}

// reach is meet, and where the ray landed when the thing is slayer's
// mesh: the same distance, worked the same way, so a search by reach
// finds what a search by meet finds.
func (placed *placedThing) reach(ray maths.Ray, near,
	far float64) (float64, landing, bool) {
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

// shade is what one look brings back: the first thing its ray meets,
// lit by every lamp that can see that point, or the sky, with whatever
// volumes lie in front of it added over.
func shade(scene staged, seen look) frame.Pixel {
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
func surface(scene staged, seen look) frame.Pixel {
	ray := seen.ray
	met, found := scene.crowd.first(ray, 0)
	if !found {
		endless := math.Inf(1)
		sky := scene.air(ray.Dir, endless, scene.skyAt(ray.Dir))
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
func surfaceAt(scene staged, seen look, met hit) frame.Pixel {
	ray, placed, dist := seen.ray, met.thing, met.dist
	point := ray.Point(dist)
	local := placed.at.In(point)
	normal := met.normal(point, ray.Dir)
	material := placed.thing.Material
	if material == nil {
		material = grey
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
	skylit := scene.skylight(lifted, normal, colour)
	sum := emitted.Add(lit).Add(skylit)
	if told != nil {
		told.tell("its own light, the lamps' and the sky's", "emits",
			emitted, "lamps", lit, "sky", skylit, "sum", sum)
	}
	// a glossy surface mirrors whatever the eye would see along the
	// mirrored ray, once, by the share fresnel gives at this look
	share := material.Mirrors(-normal.Dot(ray.Dir))
	if share > 0 && seen.bounce < bounces {
		off := shade(scene, seen.mirrored(scene.bounced(ray, point,
			normal)))
		sum = sum.Scale(1 - share).Add(off.Light().Scale(share))
		if told != nil {
			told.tell("a mirror, by fresnel's share", "share",
				share, "brought", off.Light(), "sum", sum)
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
func (scene staged) lamplight(point, normal maths.Vec,
	colour maths.Light) maths.Light {
	var sum maths.Light
	told := scene.walked
	squarely := func(toward maths.Vec, light maths.Light) float64 {
		return light.Luminance() * math.Max(0, normal.Dot(toward))
	}
	for _, one := range scene.asked(point, normal, squarely) {
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
		reached := scene.transmit(from, toward, far)
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

// bounced is the ray a mirror at a point sends an arriving ray off
// along: turned about the normal, and lifted off the surface so it
// does not meet the point it left.
func (scene staged) bounced(ray maths.Ray, point,
	normal maths.Vec) maths.Ray {
	way := ray.Dir.Sub(normal.Scale(2 * ray.Dir.Dot(normal))).Unit()
	return maths.Ray{At: scene.lifted(point, normal), Dir: way}
}

// skyAt is the sky's light from the direction toward, or no light
// with no sky.
func (scene staged) skyAt(toward maths.Vec) maths.Light {
	if scene.sky == nil {
		return maths.Light{}
	}
	return scene.sky.Light(toward.Unit())
}

// air is light that has come dist through the haze toward the eye,
// along the ray's own direction: some of it, and the rest the haze's
// own colour, or the sky's that way.
func (scene staged) air(along maths.Vec, dist float64,
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
