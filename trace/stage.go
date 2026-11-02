package trace

import (
	"math"
	"sort"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/skytable"
)

// placedThing pairs a scene object with its placement at the render time.
type placedThing struct {
	thing *thing.Thing
	placement
}

// placement caches a surface or volume transform, effective scale, and
// time-evaluated material textures. paint corresponds to Texture; absent
// textures remain nil.
type placement struct {
	at                  maths.Subspace
	scale               float64
	paint, shine, rough texture.Texture
}

// newPlacement evaluates a transform and optional material at the render time.
func newPlacement(at maths.Subspace, material *thing.Material,
	moment time.Duration) placement {
	put := placement{at: at, scale: unitScale(at)}
	if material != nil {
		put.paint = ticked(material.Texture, moment)
		put.shine = ticked(material.Shine, moment)
		put.rough = ticked(material.Rough, moment)
	}
	return put
}

// ticked evaluates animated textures at a time and leaves static ones
// unchanged.
func ticked(painted texture.Texture, moment time.Duration) texture.Texture {
	if moving, ok := painted.(texture.Timed); ok {
		return moving.Tick(moment)
	}
	return painted
}

// emits samples the time-evaluated shine at the local pixel footprint, falling
// back to the material’s uniform Glow, or zero without a material.
func emits(material *thing.Material, shine texture.Texture,
	local maths.Vec, footprint float64) maths.Light {
	if shine != nil {
		return texture.Sample(shine, local, footprint)
	}
	return material.Glow
}

// unitScale interprets a zero subspace scale as one, matching maths.Subspace.
// This helper centralizes that convention until the transform exposes its
// effective scale directly.
func unitScale(at maths.Subspace) float64 {
	if at.Scale == 0 {
		return 1
	}
	return at.Scale
}

// facing evaluates the world-space normal at point and orients it against dir.
// Interior hits therefore use the opposite of the outward normal.
func (placed placedThing) facing(point, dir maths.Vec) maths.Vec {
	local := placed.at.In(point)
	return placed.toward(shape.Normal(placed.thing.Shape, local), dir)
}

// toward transforms a local normal into world space and orients it against dir.
func (placed placedThing) toward(out, dir maths.Vec) maths.Vec {
	normal := placed.at.Orientation.Out(out).Unit()
	if normal.Dot(dir) > 0 {
		return normal.Neg()
	}
	return normal
}

// place evaluates visible surface placements at a time, omitting hidden objects
// and objects without geometry.
func place(world *thing.World, moment time.Duration) []placedThing {
	things := make([]placedThing, 0, len(world.Things))
	for _, each := range world.Things {
		if each.Hidden() || each.Shape == nil {
			continue
		}
		things = append(things, placedThing{thing: each,
			placement: newPlacement(each.At(moment), each.Material,
				moment)})
	}
	return things
}

// placedVolume caches a volume’s placement at the render time.
type placedVolume struct {
	source *thing.Volume
	placement
	// med is the medium as it is at the picture's moment
	med medium.Medium
	// gas is the volume's optics when its medium is a gas, per unit
	// of the medium's space at unit density, and nil for a painted
	// volume.
	//
	// A gas volume's density is a share of its full density, one at
	// a world's surface. Nothing of its material is read, not its
	// colour, glow, shine or radiant, and neither is its beam: the gas
	// says what light meets, in every band, and gives none off.
	gas *gas.Gas
	// extinction and scatter are the gas's own numbers at the eye's
	// three bands, asked once when the volume is placed: they are the
	// same at every step of every ray of a picture.
	extinction, scatter maths.Light
	// tint is how many times its density each channel meets, for a
	// Tinted medium, asked once when the volume is placed; nil for a
	// medium as thick to every channel.
	tint *maths.Light
}

// placeVolumes evaluates visible volumes at a time, omitting hidden entries and
// volumes without a medium.
func placeVolumes(world *thing.World, moment time.Duration) []placedVolume {
	volumes := make([]placedVolume, 0, len(world.Volumes))
	for _, each := range world.Volumes {
		if each.Hidden() || each.Medium == nil {
			continue
		}
		volume := placedVolume{source: each, med: each.Medium,
			placement: newPlacement(each.At(moment), each.Material,
				moment)}
		if moving, ok := each.Medium.(medium.Timed); ok {
			volume.med = moving.Tick(moment)
		}
		if tinted, ok := volume.med.(Tinted); ok {
			tint := tinted.Tint()
			volume.tint = &tint
		}
		if optical, ok := volume.med.(gas.Optical); ok {
			found := optical.Gas()
			volume.gas = &found
			volume.extinction = byEye(found.Extinction)
			volume.scatter = byEye(found.Scatter)
		}
		volumes = append(volumes, volume)
	}
	return volumes
}

// staged is a world evaluated for one render time: placed objects, volumes, and
// fields, plus a snapshot of lamps, sky, and haze. The lamp slice is copied, so
// later additions do not enter this render. Geometry, media, and materials
// remain shared and must not change until rendering finishes.
type staged struct {
	lamps []thing.Lamp
	sky   thing.Sky
	// lighting is the sky as a table of each cell's mean, which is what
	// lights a surface, as the gpu's kernel reads it; nil with no sky
	lighting *skytable.Table
	haze     thing.Haze
	crowd    crowd
	volumes  []placedVolume
	fields   []placedField
	// gassy is whether any volume is a gas, worked out once a picture
	gassy bool
	// walked is a replay's steps, told as the pixel is made; nil in
	// every render but a replay
	walked *walk
	// spread is the reach of the box round every thing with an edge, which
	// a ray leaving a surface is lifted by a share of; zero when no thing
	// has an edge
	spread float64
}

// lifted offsets a ray origin along the surface normal to avoid intersecting
// the surface it just left.
func (scene staged) lifted(point, normal maths.Vec) maths.Vec {
	return point.Add(normal.Scale(scene.lifting(point)))
}

// lifting computes the ray-origin offset as lift times the larger of scene
// extent and distance from the origin. The latter handles unbounded geometry
// excluded from the scene bounds.
func (scene staged) lifting(point maths.Vec) float64 {
	return lift * max(scene.spread, point.Len())
}

// Spread returns the extent of finite object bounds at the render time, or zero
// if none exist. CPU and GPU use it to scale ray-origin offsets consistently
// across scene sizes.
func Spread(world *thing.World, moment time.Duration) float64 {
	things := place(world, moment)
	boxes := make([]maths.Box, len(things))
	for i := range things {
		boxes[i] = worldBox(&things[i])
	}
	return spreadOf(boxes)
}

// spreadOf returns the combined extent of bounded boxes, or zero if none exist.
func spreadOf(boxes []maths.Box) float64 {
	around := emptyBox()
	for _, box := range boxes {
		if edged(box) {
			around = joined(around, box)
		}
	}
	size := reach(around)
	if math.IsInf(size, 0) || math.IsNaN(size) {
		return 0
	}
	return size
}

// stage evaluates the world at a render time.
func stage(world *thing.World, moment time.Duration) staged {
	volumes := placeVolumes(world, moment)
	// the lamps are copied before the sky is first read, so a sky that
	// changes the world when it is read changes no lamp of this picture
	lamps := append([]thing.Lamp(nil), world.Lamps...)
	var lighting *skytable.Table
	if world.Sky != nil {
		lighting = skytable.Of(world.Sky)
	}
	things := gathered(place(world, moment))
	return staged{
		lamps:    lamps,
		sky:      world.Sky,
		lighting: lighting,
		haze:     world.Haze,
		crowd:    things,
		volumes:  volumes,
		fields:   placeFields(world, moment),
		gassy:    hasGas(volumes),
		spread:   spreadOf(things.boxes),
	}
}

// reached holds a lamp’s attenuated light and direction at a point. A zero
// direction identifies ambient illumination.
type reached struct {
	toward maths.Vec
	light  maths.Light
}

// ambient reports the lamps’ zero-direction convention for illumination from
// all directions, such as a flood.
func (got reached) ambient() bool {
	return got.toward == (maths.Vec{})
}

// reaching evaluates a lamp at a point and normal. Zero contributions skip
// shadow work; ambient light is unshadowed; directed light uses transmit.
func (scene staged) reaching(lamp thing.Lamp, point,
	facing maths.Vec) reached {
	toward, far, light := lamp.Light(point, facing)
	got := reached{toward: toward, light: light}
	if light == (maths.Light{}) || got.ambient() {
		return got
	}
	got.light = light.Times(scene.transmit(point, toward, far))
	return got
}

// transmit returns per-channel transmission from a lamp far units along toward
// to the point. Surfaces block the path; intervening volumes and gases
// attenuate it.
func (scene staged) transmit(from, toward maths.Vec,
	far float64) maths.Light {
	if scene.crowd.blocked(maths.Ray{At: from, Dir: toward}, far) {
		return maths.Light{}
	}
	passed := paintedClear(scene.volumes, from, toward, far)
	if !scene.gassy {
		return passed
	}
	return passed.Times(gasClear(scene.volumes, from, toward, far))
}

// extent stores entry and exit distances through a volume in local units.
type extent struct{ enter, leave float64 }

// span clips a local ray to a volume before limit, preferring the medium’s Span
// method and otherwise using its bounds. Misses and tangencies have no
// integration interval.
func (volume placedVolume) span(ray maths.Ray, limit float64) (extent, bool) {
	var inside extent
	var crossed bool
	if exact, can := volume.med.(medium.Spanner); can {
		inside.enter, inside.leave, crossed = exact.Span(ray, 0, limit)
	} else {
		box := volume.med.Bounds()
		inside.enter, inside.leave, crossed = box.Hit(ray, 0, limit)
	}
	if !crossed || inside.leave <= inside.enter {
		return extent{}, false
	}
	return inside, true
}

// crossing stores world-space entry and exit distances and the ray transformed
// into the volume’s local coordinates.
type crossing struct {
	volume       *placedVolume
	enter, leave float64
	local        maths.Ray
}

// crossings returns volume intersections before far, sorted by entry distance.
func crossings(volumes []placedVolume, ray maths.Ray, far float64) []crossing {
	var across []crossing
	for i := range volumes {
		volume := &volumes[i]
		local := volume.at.RayIn(ray)
		inside, crossed := volume.span(local, math.Min(far,
			nowhere)/volume.scale)
		if !crossed {
			continue
		}
		across = append(across, crossing{volume: volume,
			enter: inside.enter * volume.scale,
			leave: inside.leave * volume.scale, local: local})
	}
	if len(across) > 1 {
		sort.SliceStable(across, func(i, after int) bool {
			return across[i].enter < across[after].enter
		})
	}
	return across
}
