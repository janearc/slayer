package trace

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

const (
	oceanSunInvalidPath uint32 = iota + 1
	oceanSunNoBoundary
)

// oceanDirectSun is the validated source needed by the explicitly authored
// finite, horizontal mean-plane sunlight approximation.
type oceanDirectSun struct {
	boundary       *thing.Thing
	placedBoundary *placedThing
	medium         *thing.ClearMedium
	material       *thing.Material
	metresPerUnit  float64
	sunIndex       int
	fillIndex      int
	sunToward      maths.Vec
	strength       maths.Light
	skyDirections  []maths.Vec
	waterStepMode  string
}

type oceanSunPathMiss struct {
	shape                string
	worldOrigin          maths.Vec
	worldDirection       maths.Vec
	localOrigin          maths.Vec
	localDirection       maths.Vec
	localSignedGap       float64
	viewOrigin           maths.Vec
	viewDirection        maths.Vec
	viewSignedGap        float64
	viewBoundaryDistance float64
	viewBoundaryHit      bool
	viewBeforeSampleDist float64
	viewBeforeSampleHit  bool
	viewSampleLocal      maths.Vec
	viewSampleGap        float64
	viewScale            float64
	viewCrowdDistance    float64
	viewCrowdHit         bool
	viewCrowdBoundary    bool
	sample               maths.Vec
	sampleAlong          float64
	crossed              int
	slipped              int
	bounce               int
	holding              int
	onward               bool
	lastClearTIR         bool
}

// recordOceanSunPathMiss keeps the geometry of the first ray that misses the
// ocean sun path, for diagnosis. Later misses are not recorded.
func (scene *staged) recordOceanSunPathMiss(ray maths.Ray,
	sample maths.Vec, seen look) {
	ocean := scene.oceanSun
	if ocean == nil || ocean.placedBoundary == nil {
		return
	}
	local := ocean.placedBoundary.at.RayIn(ray)
	gap := math.NaN()
	if measured, ok := ocean.boundary.Shape.(interface {
		Distance(maths.Vec) float64
	}); ok {
		gap = measured.Distance(local.At)
	}
	viewLengthSquared := seen.ray.Dir.Dot(seen.ray.Dir)
	sampleAlong := math.NaN()
	if viewLengthSquared > 0 && finite(viewLengthSquared) {
		sampleAlong = sample.Sub(seen.ray.At).Dot(seen.ray.Dir) /
			viewLengthSquared
	}
	viewLocal := ocean.placedBoundary.at.In(seen.ray.At)
	viewLocalRay := ocean.placedBoundary.at.RayIn(seen.ray)
	viewGap := math.NaN()
	if measured, ok := ocean.boundary.Shape.(interface {
		Distance(maths.Vec) float64
	}); ok {
		viewGap = measured.Distance(viewLocal)
	}
	viewBoundaryDistance, viewBoundaryHit := ocean.placedBoundary.meet(
		seen.ray, 0, math.Inf(1))
	viewBeforeSampleDistance, viewBeforeSampleHit := math.NaN(), false
	viewSampleLocal, viewSampleGap := maths.Vec{}, math.NaN()
	if finite(sampleAlong) && sampleAlong > 0 &&
		finite(ocean.placedBoundary.scale) && ocean.placedBoundary.scale > 0 {
		viewBeforeSampleDistance, viewBeforeSampleHit =
			ocean.placedBoundary.meet(seen.ray, 0, sampleAlong)
		viewSampleLocal = viewLocalRay.Point(sampleAlong /
			ocean.placedBoundary.scale)
		if measured, ok := ocean.boundary.Shape.(interface {
			Distance(maths.Vec) float64
		}); ok {
			viewSampleGap = measured.Distance(viewSampleLocal)
		}
	}
	viewFirst, viewCrowdHit := scene.crowd.first(seen.ray, 0)
	viewCrowdDistance := math.NaN()
	viewCrowdBoundary := false
	if viewCrowdHit {
		viewCrowdDistance = viewFirst.dist
		viewCrowdBoundary = viewFirst.thing.thing == ocean.boundary
	}
	miss := &oceanSunPathMiss{shape: fmt.Sprintf("%T",
		ocean.boundary.Shape), worldOrigin: ray.At,
		worldDirection: ray.Dir, localOrigin: local.At,
		localDirection: local.Dir, localSignedGap: gap,
		viewOrigin: seen.ray.At, viewDirection: seen.ray.Dir,
		viewSignedGap:        viewGap,
		viewBoundaryDistance: viewBoundaryDistance,
		viewBoundaryHit:      viewBoundaryHit,
		viewBeforeSampleDist: viewBeforeSampleDistance,
		viewBeforeSampleHit:  viewBeforeSampleHit,
		viewSampleLocal:      viewSampleLocal,
		viewSampleGap:        viewSampleGap,
		viewScale:            ocean.placedBoundary.scale,
		viewCrowdDistance:    viewCrowdDistance,
		viewCrowdHit:         viewCrowdHit,
		viewCrowdBoundary:    viewCrowdBoundary,
		sample:               sample, sampleAlong: sampleAlong,
		crossed: seen.crossed, slipped: seen.slipped, bounce: seen.bounce,
		holding: seen.holding, onward: seen.onward,
		lastClearTIR: seen.lastClearTIR}
	scene.oceanSunMiss.CompareAndSwap(nil, miss)
}

// validateOceanDirectSunWorld checks that the ocean's direct sun route
// matches its Sun or Moon selection, and that no volume or field is mixed in.
func validateOceanDirectSunWorld(world *thing.World) error {
	ocean := world.OceanSurface
	if !ocean.MeanPlaneSource() {
		return nil
	}
	if ocean.DirectMoonMeanPlane && ocean.DirectSunMeanPlane ||
		ocean.DirectMoonMeanPlane != (ocean.SunRoute == thing.OceanSunRouteMoon) {
		return fmt.Errorf("trace: ocean directional source route does not match its Sun/Moon selection")
	}
	if (len(world.Volumes) != 0 && !clearCloudOnly(world)) || len(world.Fields) != 0 {
		return fmt.Errorf("trace: ocean mean-plane direct sun cannot be combined with volume or gravitational-field transport")
	}
	if world.Haze.Density > 0 {
		return fmt.Errorf("trace: ocean mean-plane direct sun cannot be combined with separate scene haze")
	}
	if ocean.Boundary == nil || ocean.Boundary.Shape == nil ||
		ocean.ExtentMetres <= 0 ||
		!finite(ocean.ExtentMetres, ocean.MetresPerUnit,
			ocean.Mean.X, ocean.Mean.Y, ocean.Mean.Z) ||
		ocean.MetresPerUnit <= 0 {
		return fmt.Errorf("trace: ocean mean-plane direct sun needs a finite boundary, positive extent and positive metres-per-unit")
	}
	switch ocean.Boundary.Shape.(type) {
	case *shape.HeightField, *arid.FlatMeanContinuation,
		*arid.InfiniteMeanContinuation, *arid.Sea:
	default:
		return fmt.Errorf("trace: ocean mean-plane direct sun requires the staged finite wave, flat-mean continuation or unbounded sea boundary")
	}
	foundBoundary := false
	for _, each := range world.Things {
		if each == ocean.Boundary {
			foundBoundary = true
		}
		if each.Material != nil && each.Material.Through > 0 &&
			each != ocean.Boundary {
			return fmt.Errorf("trace: ocean mean-plane direct sun refuses additional transparent boundaries")
		}
	}
	if !foundBoundary {
		return fmt.Errorf("trace: ocean mean-plane boundary is not present in the world")
	}
	if ocean.Boundary.Hidden() {
		return fmt.Errorf("trace: ocean mean-plane boundary cannot be hidden")
	}
	material := ocean.Boundary.Material
	if material == nil || material.Through != 1 || material.Gloss != 1 ||
		material.Metal != nil || !material.ClearMedium.Defined() ||
		material.ClearMedium.MetresPerUnit() != ocean.MetresPerUnit {
		return fmt.Errorf("trace: ocean mean-plane direct sun requires one fully clear named boundary with matching metre units")
	}
	if ocean.SunLampIndex < 0 || ocean.SunLampIndex >= len(world.Lamps) {
		return fmt.Errorf("trace: ocean mean-plane direct sun has no validated source lamp index")
	}
	switch ocean.SunRoute {
	case thing.OceanSunRouteGround:
		if ocean.FillLampIndex < 0 ||
			ocean.FillLampIndex >= len(world.Lamps) ||
			ocean.SunLampIndex == ocean.FillLampIndex {
			return fmt.Errorf("trace: ground ocean mean-plane sun has no distinct validated sun and fill indexes")
		}
	case thing.OceanSunRouteGroundlessAir:
		if ocean.FillLampIndex != -1 {
			return fmt.Errorf("trace: groundless-air ocean mean-plane sun must not carry a synthetic fill index")
		}
		if world.Sky == nil {
			return fmt.Errorf("trace: groundless-air ocean mean-plane sun requires its separately staged atmosphere sky")
		}
	case thing.OceanSunRouteMoon:
		if !ocean.DirectMoonMeanPlane || ocean.DirectSunMeanPlane ||
			ocean.FillLampIndex != -1 {
			return fmt.Errorf("trace: ocean mean-plane Moon needs an exclusive lunar source and no fill")
		}
	default:
		return fmt.Errorf("trace: ocean mean-plane direct sun has no supported staged source route")
	}
	if _, _, ok := distantLight(world.Lamps[ocean.SunLampIndex]); !ok {
		return fmt.Errorf("trace: ocean mean-plane source lamp %d is not emit.Distant", ocean.SunLampIndex)
	}
	sunToward, _, _ := distantLight(world.Lamps[ocean.SunLampIndex])
	if !finite(sunToward.X, sunToward.Y, sunToward.Z) || sunToward.Y <= 0 {
		return fmt.Errorf("trace: ocean mean-plane direct sun must point above the horizontal interface")
	}
	if ocean.SunRoute == thing.OceanSunRouteGround {
		if _, _, ok := distantLight(world.Lamps[ocean.FillLampIndex]); !ok {
			return fmt.Errorf("trace: ocean mean-plane fill lamp %d is not the supported emit.Distant fill", ocean.FillLampIndex)
		}
	}
	for index, lamp := range world.Lamps {
		if index == ocean.SunLampIndex ||
			ocean.SunRoute == thing.OceanSunRouteGround &&
				index == ocean.FillLampIndex {
			continue
		}
		switch lamp.(type) {
		case film.Flood, *film.Flood:
			// Directionless Flood lamps retain their pre-existing separate
			// behavior. They are never the tagged direct sun or the ground
			// route's distinct Distant fill.
		default:
			return fmt.Errorf("trace: ocean mean-plane direct sun refuses additional directed lamp %d of type %T", index, lamp)
		}
	}
	if material.ClearMedium.HasScattering() {
		for index := range world.Lamps {
			if index == ocean.SunLampIndex ||
				ocean.SunRoute == thing.OceanSunRouteGround &&
					index == ocean.FillLampIndex {
				continue
			}
			return fmt.Errorf("trace: ocean single scattering only integrates the tagged direct sun; lamp %d remains outside its source model", index)
		}
	}
	return nil
}

// validateOceanInitialMedia checks that the initial media agree with the
// side of the ocean boundary the camera is on.
func validateOceanInitialMedia(world *thing.World, eye maths.Subspace,
	media []InitialMedium, at time.Duration) error {
	ocean := world.OceanSurface
	if ocean == nil || ocean.Boundary == nil ||
		(!ocean.MeanPlaneSource() && !oceanDiffuseSkyCandidate(world,
			ocean.Boundary.Material)) {
		return nil
	}
	material := ocean.Boundary.Material
	localEye := ocean.Boundary.At(at).In(eye.Origin)
	depth := ocean.Boundary.Shape.Distance(localEye)
	if !finite(depth) || depth == 0 {
		return fmt.Errorf("trace: camera lies on or ambiguously near the ocean boundary")
	}
	shouldStartInWater := depth < 0
	found := false
	for _, initial := range media {
		if initial.ClearMedium.Name() != material.ClearMedium.Name() {
			return fmt.Errorf("trace: ocean in-scattering refuses an initial medium other than %q", material.ClearMedium.Name())
		}
		if initial.Index != material.Dense() || initial.Rank != material.Rank ||
			!sameClearMedium(initial.ClearMedium, material.ClearMedium) {
			return fmt.Errorf("trace: initial ocean medium does not match its refractive index, rank and optical properties")
		}
		found = true
	}
	if shouldStartInWater != found {
		if shouldStartInWater {
			return fmt.Errorf("trace: camera begins below the ocean boundary; supply its named clear medium as initial state")
		}
		return fmt.Errorf("trace: initial ocean medium was supplied for a camera outside the ocean interior")
	}
	return nil
}

// distantLight returns the direction towards a distant lamp and its strength.
// The result is false for any other lamp, or a nil pointer.
func distantLight(lamp thing.Lamp) (toward maths.Vec, strength maths.Light, ok bool) {
	switch source := lamp.(type) {
	case emit.Distant:
		toward, _, strength = source.Light(maths.Vec{}, maths.Vec{})
		return toward, strength, true
	case *emit.Distant:
		if source == nil {
			return maths.Vec{}, maths.Light{}, false
		}
		toward, _, strength = source.Light(maths.Vec{}, maths.Vec{})
		return toward, strength, true
	default:
		return maths.Vec{}, maths.Light{}, false
	}
}

// oceanDirectSunOf builds the direct sun description for the ocean boundary.
// It returns nil when the ocean has no direct sun source.
func oceanDirectSunOf(world *thing.World) *oceanDirectSun {
	ocean := world.OceanSurface
	if ocean == nil || ocean.Boundary == nil ||
		ocean.Boundary.Material == nil ||
		ocean.Boundary.Material.ClearMedium == nil {
		return nil
	}
	material := ocean.Boundary.Material
	if !ocean.MeanPlaneSource() &&
		material.ClearMedium.DiffuseSkyInscatterSamples() == 0 &&
		ocean.ClearCloudSkySamples == 0 {
		return nil
	}
	made := &oceanDirectSun{boundary: ocean.Boundary,
		waterStepMode: ocean.ClearCloudWaterStepMode,
		medium:        material.ClearMedium, material: material,
		metresPerUnit: ocean.MetresPerUnit, sunIndex: -1,
		fillIndex: ocean.FillLampIndex,
		skyDirections: equalSolidAngleSkyDirections(max(
			material.ClearMedium.DiffuseSkyInscatterSamples(),
			ocean.ClearCloudSkySamples))}
	if ocean.MeanPlaneSource() {
		if ocean.SunLampIndex < 0 || ocean.SunLampIndex >= len(world.Lamps) ||
			(ocean.SunRoute != thing.OceanSunRouteGround &&
				ocean.SunRoute != thing.OceanSunRouteGroundlessAir &&
				ocean.SunRoute != thing.OceanSunRouteMoon) {
			return nil
		}
		toward, strength, ok := distantLight(world.Lamps[ocean.SunLampIndex])
		if !ok {
			return nil
		}
		made.sunIndex = ocean.SunLampIndex
		made.sunToward, made.strength = toward, strength
	}
	return made
}

// equalSolidAngleSkyDirections samples the air-side upper hemisphere with
// equal-solid-angle strata. Sky is evaluated per air solid angle, so the
// transport later applies the Snell solid-angle Jacobian explicitly.
func equalSolidAngleSkyDirections(samples int) []maths.Vec {
	if samples <= 0 {
		return nil
	}
	ways := make([]maths.Vec, samples)
	golden := math.Pi * (3 - math.Sqrt(5))
	for index := range samples {
		up := (float64(index) + .5) / float64(samples)
		radius := math.Sqrt(1 - up*up)
		angle := float64(index) * golden
		ways[index] = maths.V(radius*math.Cos(angle), up,
			radius*math.Sin(angle))
	}
	return ways
}

// oceanSunActive reports whether the direct sun through the water applies
// to a look that reaches target in the ocean's clear medium.
func (scene *staged) oceanSunActive(seen look, target *thing.Thing) bool {
	if scene.oceanSun == nil || target == scene.oceanSun.boundary ||
		seen.holding == 0 {
		return false
	}
	active := activeClearMedium(seen)
	return active != nil && active.Name() == scene.oceanSun.medium.Name()
}

// oceanSunLight returns the direction, light and usable flag for the direct
// sun at point on the ocean surface, lifted along its normal.
func (scene *staged) oceanSunLight(point, normal maths.Vec,
	seen look) (maths.Vec, maths.Light, bool) {
	return scene.oceanSunLightFrom(scene.lifted(point, normal), seen)
}

// oceanSunAtSample evaluates the tagged direct source at an interior scattering
// sample without applying the surface ray-origin lift. Lifting an interior
// point along a surface normal would alter both its water path and its source
// extinction.
func (scene *staged) oceanSunAtSample(point maths.Vec,
	seen look) (maths.Vec, maths.Light, bool) {
	return scene.oceanSunLightFrom(point, seen)
}

// oceanSunLightFrom traces the sun through the water from the point from.
// It returns the direction, the light reaching from and a usable flag. Any
// failure is stored in oceanSunFault.
func (scene *staged) oceanSunLightFrom(from maths.Vec,
	seen look) (maths.Vec, maths.Light, bool) {
	ocean := scene.oceanSun
	if ocean == nil {
		return maths.Vec{}, maths.Light{}, false
	}
	if ocean.sunIndex < 0 {
		return maths.Vec{}, maths.Light{}, true
	}
	active := activeClearMedium(seen)
	if active == nil || !sameClearMedium(active, ocean.medium) {
		scene.oceanSunFault.Store(oceanSunInvalidPath)
		return maths.Vec{}, maths.Light{}, false
	}
	airToward := ocean.sunToward.Unit()
	if airToward.Y <= 0 || !finite(airToward.X, airToward.Y,
		airToward.Z) {
		scene.oceanSunFault.Store(oceanSunInvalidPath)
		return maths.Vec{}, maths.Light{}, false
	}
	// Trace the incoming solar direction from air into the water. The
	// refracted direction is reversed to get from the sample toward the sun.
	incomingAir := airToward.Neg()
	face := maths.V(0, 1, 0)
	incomingWater, total := refracted(incomingAir, face,
		ocean.material.Dense())
	if total || incomingWater == (maths.Vec{}) {
		scene.oceanSunFault.Store(oceanSunInvalidPath)
		return maths.Vec{}, maths.Light{}, false
	}
	towardWater := incomingWater.Neg().Unit()
	cosAir, cosWater := airToward.Y, towardWater.Y
	if cosWater <= 0 || !finite(cosWater) {
		scene.oceanSunFault.Store(oceanSunInvalidPath)
		return maths.Vec{}, maths.Light{}, false
	}
	if ocean.placedBoundary == nil {
		scene.oceanSunFault.Store(oceanSunInvalidPath)
		return maths.Vec{}, maths.Light{}, false
	}
	ray := maths.Ray{At: from, Dir: towardWater}
	waterDistance, intersects := ocean.placedBoundary.meet(ray, 0,
		math.Inf(1))
	if !intersects {
		scene.oceanSunFault.Store(oceanSunNoBoundary)
		return maths.Vec{}, maths.Light{}, false
	}
	if waterDistance <= 0 || !finite(waterDistance) {
		scene.oceanSunFault.Store(oceanSunInvalidPath)
		return maths.Vec{}, maths.Light{}, false
	}
	crossing := ray.Point(waterDistance)
	if scene.crowd.blockedExcept(maths.Ray{At: from, Dir: towardWater},
		waterDistance, ocean.boundary) ||
		scene.crowd.blockedExcept(maths.Ray{At: crossing, Dir: airToward},
			math.Inf(1), ocean.boundary) {
		return maths.Vec{}, maths.Light{}, true
	}
	reflectance := ocean.material.SplitsBetween(cosAir,
		ocean.material.Dense())
	projected := (1 - reflectance) * cosAir / cosWater
	metres := waterDistance * ocean.metresPerUnit
	extinction := ocean.medium.Extinction()
	attenuation := maths.RGB(beerClear(extinction.R, metres),
		beerClear(extinction.G, metres), beerClear(extinction.B, metres))
	light := ocean.strength.Scale(projected).Times(attenuation)
	light = light.Times(paintedClear(scene.volumes, from, towardWater,
		waterDistance))
	return towardWater, light, true
}

// oceanLamplight returns the lamp light at point, adding the direct sun
// through the water when the ocean sun path applies.
func (scene *staged) oceanLamplight(point, normal maths.Vec,
	colour maths.Light, seen look, target *thing.Thing) maths.Light {
	if !scene.oceanSunActive(seen, target) {
		return scene.lamplight(point, normal, colour)
	}
	sum := scene.lamplightExcept(point, normal, colour,
		scene.oceanSun.sunIndex)
	toward, light, usable := scene.oceanSunLight(point, normal, seen)
	if !usable || light == (maths.Light{}) {
		return sum
	}
	square := math.Max(0, normal.Dot(toward))
	if square == 0 {
		return sum
	}
	matte := colour.Times(light).Scale(square / math.Pi)
	return sum.Add(matte)
}

// oceanGlint returns the glint at point, adding the direct sun through the
// water when the ocean sun path applies.
func (scene *staged) oceanGlint(point, normal, look maths.Vec,
	material *thing.Material, seen look, target *thing.Thing) maths.Light {
	if !scene.oceanSunActive(seen, target) {
		return scene.glint(point, normal, look, material)
	}
	sum := scene.glintExcept(point, normal, look, material,
		scene.oceanSun.sunIndex)
	toward, light, usable := scene.oceanSunLight(point, normal, seen)
	if !usable || light == (maths.Light{}) {
		return sum
	}
	viewed, square := normal.Dot(look), normal.Dot(toward)
	if !glints(material) || viewed <= 0 || square <= 0 {
		return sum
	}
	half := toward.Add(look).Unit()
	lobe := highlight(material, normal.Dot(half), square, viewed)
	if lobe <= 0 {
		return sum
	}
	return sum.Add(light.Times(material.Reflects(half.Dot(look))).Scale(lobe))
}
