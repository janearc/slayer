package trace

import (
	"fmt"
	"math"
	"slices"
	"sync/atomic"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/beamstep"
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
// back to the material's uniform Glow, or zero without a material.
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

// placedVolume caches a volume's placement at the render time.
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
	// bounds is the medium's box in its own space, asked once when the
	// volume is placed: a bent plume works its box out afresh at every
	// ask, from numbers that are the same for the whole picture
	bounds maths.Box
	// world is that box in the world, and walled whether it has one: a
	// ray that misses it misses the volume, and is told so before it is
	// taken into the volume's space. A medium of no finite box is not
	// walled, and every ray asks it.
	world  maths.Box
	walled bool
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
			volume.extinction, volume.scatter = EyeOptics(found)
		}
		volume.bounds = volume.med.Bounds()
		volume.world, volume.walled = boxOut(volume.at, volume.bounds)
		volumes = append(volumes, volume)
	}
	return volumes
}

// refusal is why the staged scene cannot be drawn, or nil: a field that
// celestial says is no hole, or a gas volume whose asymmetry is no lobe,
// which would panic in a worker at its first lamp.
func (scene *staged) refusal() error {
	if scene.fault != nil {
		return scene.fault
	}
	switch scene.oceanSunFault.Load() {
	case oceanSunInvalidPath:
		return fmt.Errorf("trace: ocean mean-plane sun path has an invalid intersection or direction")
	case oceanSunNoBoundary:
		if miss := scene.oceanSunMiss.Load(); miss != nil {
			return fmt.Errorf("trace: ocean mean-plane source path missed %s boundary: world ray origin=%+v direction=%+v; local origin=%+v direction=%+v signed-gap=%g; view ray origin=%+v direction=%+v signed-gap=%g scale=%g sample-local=%+v sample-gap=%g pre-sample-hit=(%g,%t) shape-hit=(%g,%t) crowd-first=(%g,%t,boundary=%t) sample=%+v along=%g state={crossed:%d slipped:%d bounce:%d holding:%d onward:%t last-clear-TIR:%t}", miss.shape, miss.worldOrigin, miss.worldDirection, miss.localOrigin, miss.localDirection, miss.localSignedGap, miss.viewOrigin, miss.viewDirection, miss.viewSignedGap, miss.viewScale, miss.viewSampleLocal, miss.viewSampleGap, miss.viewBeforeSampleDist, miss.viewBeforeSampleHit, miss.viewBoundaryDistance, miss.viewBoundaryHit, miss.viewCrowdDistance, miss.viewCrowdHit, miss.viewCrowdBoundary, miss.sample, miss.sampleAlong, miss.crossed, miss.slipped, miss.bounce, miss.holding, miss.onward, miss.lastClearTIR)
		}
		return fmt.Errorf("trace: ocean mean-plane source path does not meet the configured boundary; the path may leave its finite footprint")
	}
	if scene.clearInscatterFault != nil {
		switch scene.clearInscatterFault.Load() {
		case clearInscatterAmbientUnsupported:
			return fmt.Errorf("trace: clear-medium in-scattering refuses a directionless ambient lamp")
		case clearInscatterNumericalOverflow:
			return fmt.Errorf("trace: clear-medium in-scattering produced an unrepresentable source or path")
		}
	}
	for _, each := range scene.crowd.things {
		material := each.thing.Material
		if material != nil && material.WakeFoam != nil && material.WakeFoam.Coverage > 0 &&
			(material.Texture != nil || material.Shine != nil ||
				material.Glow != (maths.Light{})) {
			return fmt.Errorf("trace: wake foam refuses a painted, shining or emissive material")
		}
		if material != nil && material.WakeFoam != nil && material.WakeFoam.Coverage > 0 &&
			(scene.oceanSun != nil || scene.oceanSpotInscatter) {
			return fmt.Errorf("trace: wake foam cannot be combined with mean-plane or participating source approximations")
		}
		if material != nil && material.WakeFoam != nil && !material.WakeFoam.Valid() {
			return fmt.Errorf("trace: wake foam parameters are invalid or outside their authored bounds")
		}
	}
	for _, each := range scene.volumes {
		if each.gas != nil && !each.gas.IsLobe() {
			return fmt.Errorf("trace: gas %s has an asymmetry of %v, "+
				"which is no lobe: it must be between minus one and one, "+
				"and neither of them", each.gas.Name, each.gas.Phase)
		}
	}
	return nil
}

// staged is a world evaluated for one render time: placed objects, volumes, and
// fields, plus a snapshot of lamps, sky, and haze. The lamp slice is copied, so
// later additions do not enter this render. Geometry, media, and materials
// remain shared and must not change until rendering finishes.
type staged struct {
	seconds float64
	// volumeSky is a fixed full-sphere quadrature for isotropic scattering.
	// Empty keeps the legacy source term and lamp normalization.
	volumeSky []maths.Vec
	lamps     []thing.Lamp
	// tree is the standing lamps as a tree to pick among, and others
	// every other lamp's index, built once; no tree for a scene of few
	// lamps. See lamptree.go.
	tree   *lampTree
	others []int
	// discs is the suns among lamps drawn as discs, found once
	discs []film.Sun
	beams *beamstep.Beams
	sky   thing.Sky
	glow  thing.Glow
	// lighting is the sky as a table of each cell's mean, which is what
	// lights a surface, as the gpu's kernel reads it; nil with no sky
	lighting *skytable.Table
	haze     thing.Haze
	crowd    crowd
	volumes  []placedVolume
	fields   []placedField
	// fault is why a field is no hole, as fieldFault has it; nil if none
	fault error
	// clearInscatterFault is a runtime refusal for a custom lamp whose
	// direction cannot be established before camera-path samples are evaluated.
	clearInscatterFault *atomic.Uint32
	// gassy is whether any volume is a gas, worked out once a picture
	gassy bool
	// walked is a replay's steps, told as the pixel is made; nil in
	// every render but a replay
	walked *walk
	// spread encloses bounded surfaces that can reflect. Secondary rays use
	// a share of it as their local offset; nonreflecting geometry still
	// casts shadows without enlarging this scale
	spread float64
	// marching is the things a march draws, which the lift must clear
	marching marchedThings
	// oceanSun is non-nil only for the explicit CPU mean-plane direct-sun
	// approximation. Its source and boundary were validated before workers.
	oceanSun           *oceanDirectSun
	oceanSpotInscatter bool
	oceanSunFault      atomic.Uint32
	oceanSunMiss       atomic.Pointer[oceanSunPathMiss]
	oceanEventBoundary *thing.Thing
	oceanEventLimit    int
}

// lifted offsets a ray origin along the surface normal to avoid intersecting
// the surface it just left.
func (scene *staged) lifted(point, normal maths.Vec) maths.Vec {
	return point.Add(normal.Scale(scene.lifting(point)))
}

// lifting computes the ray-origin offset as lift times the larger of scene
// extent and distance from the origin. The latter handles unbounded geometry
// excluded from the scene bounds. Both can be nothing at once, at a plane
// through the origin of an unbounded scene; a ray a surface sends onward
// measures itself by its own travel too, through liftedAlong, and no
// absolute length appears anywhere, so the picture scales.
//
// Where a march draws a thing, the offset is no less than its marched
// surface is cleared by, as marchedThings.clear has it.
func (scene *staged) lifting(point maths.Vec) float64 {
	return max(lift*max(scene.spread, point.Len()),
		scene.marching.clear(point))
}

// liftedAlong is lifted with the asking ray's own travel as one more
// length to measure the lift by: the ray came from somewhere, and that
// length scales with the scene when nothing else does.
func (scene *staged) liftedAlong(point, normal, from maths.Vec) maths.Vec {
	came := point.Sub(from).Len()
	return point.Add(normal.Scale(max(scene.lifting(point), lift*came)))
}

// Spread returns a declared ray-origin reference, or the legacy reflecting
// extent when none is supplied. Nonreflecting surfaces do not enlarge that
// legacy extent. All geometry remains visible and available to shadow queries.
func Spread(world *thing.World, moment time.Duration) float64 {
	return rayOriginSpread(world, place(world, moment))
}

// rayOriginSpread shares the same declared numerical reference with GPU
// encoding and CPU staging. The zero choice retains legacy scene bounds.
func rayOriginSpread(world *thing.World, things []placedThing) float64 {
	if reference := world.RayOriginReference.Metres(); reference > 0 {
		return reference
	}
	return reflectingSpread(things)
}

// reflectingSpread includes diffuse, textured, glossy and spectral surfaces.
// Black surfaces and pure emitters are omitted regardless of their distance;
// they remain ordinary primary-ray surfaces and shadow occluders.
func reflectingSpread(things []placedThing) float64 {
	var boxes []maths.Box
	for i := range things {
		m := things[i].thing.Material
		if m != nil && m.Colour == (maths.Light{}) && m.Texture == nil && m.Gloss <= 0 && m.Radiant == nil {
			continue
		}
		boxes = append(boxes, worldBox(&things[i]))
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

// stage evaluates the world at a render time. The result is not changed
// once a picture starts, so every worker of a render shares it.
func stage(world *thing.World, moment time.Duration) *staged {
	volumes := placeVolumes(world, moment)
	// the lamps are copied before the sky is first read, so a sky that
	// changes the world when it is read changes no lamp of this picture
	lamps := append([]thing.Lamp(nil), world.Lamps...)
	var lighting *skytable.Table
	if world.Sky != nil {
		lighting = skytable.Of(world.Sky)
	}
	things := gathered(place(world, moment))
	made := &staged{
		volumeSky: world.VolumeIllumination.Directions(),
		lamps:     lamps,
		discs:     film.Disced(lamps),
		beams:     beamstep.Of(lamps),
		sky:       world.Sky,
		glow:      world.Glow,
		lighting:  lighting,
		haze:      world.Haze,
		crowd:     things,
		volumes:   volumes,
		fields:    placeFields(world, moment),
		fault:     fieldFault(world),
		gassy:     hasGas(volumes),
		spread:    rayOriginSpread(world, things.things),
		marching:  marchedBy(things.things),
		oceanSpotInscatter: world.OceanSurface != nil &&
			world.OceanSurface.SpotInscatter,
		seconds: moment.Seconds(),
	}
	if len(lamps) > lampsEvery {
		made.tree, made.others = treeOf(lamps)
	}
	if ocean := world.OceanSurface; ocean != nil && ocean.InterfaceEvents > 0 {
		made.oceanEventBoundary, made.oceanEventLimit = ocean.Boundary,
			ocean.InterfaceEvents
	}
	if ocean := world.OceanSurface; ocean != nil &&
		(ocean.MeanPlaneSource() || ocean.Boundary != nil &&
			(oceanDiffuseSkyCandidate(world, ocean.Boundary.Material) ||
				oceanClearCloudSkyCandidate(world, ocean.Boundary.Material))) {
		made.oceanSun = oceanDirectSunOf(world)
		if made.oceanSun != nil {
			for index := range made.crowd.things {
				if made.crowd.things[index].thing == made.oceanSun.boundary {
					made.oceanSun.placedBoundary = &made.crowd.things[index]
					break
				}
			}
		}
	}
	return made
}

// enableClearInscatterFault allocates a concurrent refusal flag only for a
// render that starts in, or can enter, a participating medium. The ocean
// route is the only supported participating boundary, so checking its
// material is constant work; ordinary non-scattering worlds keep the legacy
// zero-state path.
func (scene *staged) enableClearInscatterFault(seen view) {
	for _, each := range seen.inside[:seen.holding] {
		if each.clearMedium.HasScattering() {
			scene.clearInscatterFault = &atomic.Uint32{}
			return
		}
	}
	if seen.world != nil && seen.world.OceanSurface != nil &&
		seen.world.OceanSurface.Boundary != nil {
		material := seen.world.OceanSurface.Boundary.Material
		if material != nil && material.ClearMedium.HasScattering() {
			scene.clearInscatterFault = &atomic.Uint32{}
		}
	}
}

// reached holds a lamp's attenuated light and direction at a point. A zero
// direction identifies ambient illumination.
type reached struct {
	toward maths.Vec
	light  maths.Light
}

// ambient reports the lamps' zero-direction convention for illumination from
// all directions, such as a flood.
func (got reached) ambient() bool {
	return got.toward == (maths.Vec{})
}

// reaching evaluates a lamp at a point and normal. Zero contributions skip
// shadow work; ambient light is unshadowed; directed light uses transmit.
func (scene *staged) reaching(lamp thing.Lamp, point,
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
func (scene *staged) transmit(from, toward maths.Vec,
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

// span clips a local ray to a volume before limit, preferring the medium's Span
// method and otherwise using its bounds. Misses and tangencies have no
// integration interval.
func (volume placedVolume) span(ray maths.Ray, limit float64) (extent, bool) {
	var inside extent
	var crossed bool
	if exact, can := volume.med.(medium.Spanner); can {
		inside.enter, inside.leave, crossed = exact.Span(ray, 0, limit)
	} else {
		inside.enter, inside.leave, crossed = volume.bounds.Hit(ray, 0, limit)
	}
	if !crossed || inside.leave <= inside.enter {
		return extent{}, false
	}
	return inside, true
}

// intervals retains disjoint shell legs; ordinary media keep their one span.
func (volume placedVolume) intervals(ray maths.Ray, limit float64) [2]extent {
	if split, ok := volume.med.(medium.SplitSpanner); ok {
		parts := split.Spans(ray, 0, limit)
		return [2]extent{{parts[0].Enter, parts[0].Leave}, {parts[1].Enter, parts[1].Leave}}
	}
	inside, crossed := volume.span(ray, limit)
	if !crossed {
		return [2]extent{}
	}
	return [2]extent{inside}
}

// crossing stores world-space entry and exit distances and the ray transformed
// into the volume's local coordinates.
type crossing struct {
	volume       *placedVolume
	enter, leave float64
	local        maths.Ray
}

// crossings returns volume intersections before far, sorted by entry distance.
func crossings(volumes []placedVolume, ray maths.Ray, far float64) []crossing {
	return crossingsIn(nil, volumes, ray, far)
}

// crossingsIn is crossings with the result built on the end of room, so a
// caller that passes an empty slice with capacity allocates only for
// crossings beyond it. The result shares room's memory.
func crossingsIn(room []crossing, volumes []placedVolume, ray maths.Ray,
	far float64) []crossing {
	across := room
	for i := range volumes {
		volume := &volumes[i]
		if !volume.meets(ray, math.Min(far, nowhere)) {
			continue
		}
		local := volume.at.RayIn(ray)
		for _, inside := range volume.intervals(local, math.Min(far, nowhere)/volume.scale) {
			if inside.leave <= inside.enter {
				continue
			}
			across = append(across, crossing{volume: volume,
				enter: inside.enter * volume.scale,
				leave: inside.leave * volume.scale, local: local})
		}
	}
	if len(across) > 1 {
		slices.SortStableFunc(across, func(one, other crossing) int {
			// less-than only: a NaN entry compares equal to every other
			switch {
			case one.enter < other.enter:
				return -1
			case other.enter < one.enter:
				return 1
			}
			return 0
		})
	}
	return across
}
