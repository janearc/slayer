package trace

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// oceanDiffuseSkyAtSample estimates the incident single-scatter source at one
// water sample. It integrates the staged air-side sky over equal-solid-angle
// directions, maps each through the authored horizontal mean plane, and
// includes water-path Beer attenuation. Wave-resolved refraction is not
// modeled.
func oceanDiffuseSkyAtSample(scene *staged, point maths.Vec,
	seen look, viewDirection maths.Vec,
) maths.Light {
	return oceanDiffuseSkyWithPhase(scene, point, seen, viewDirection,
		cloudPhaseSelection{})
}

// oceanDiffuseSkyWithPhase is oceanDiffuseSkyAtSample with the cloud phase
// taken from phaseChoice.
func oceanDiffuseSkyWithPhase(scene *staged, point maths.Vec,
	seen look, viewDirection maths.Vec, phaseChoice cloudPhaseSelection,
) maths.Light {
	ocean := scene.oceanSun
	if ocean == nil || len(ocean.skyDirections) == 0 ||
		scene.lighting == nil {
		return maths.Light{}
	}
	viewDirection = viewDirection.Unit()
	if viewDirection == (maths.Vec{}) ||
		!finite(viewDirection.X, viewDirection.Y, viewDirection.Z) {
		markClearInscatterFault(scene, clearInscatterNumericalOverflow)
		return maths.Light{}
	}
	var sum maths.Light
	weight := 2 * math.Pi / float64(len(ocean.skyDirections))
	for _, airToward := range ocean.skyDirections {
		towardWater, incident, usable := scene.oceanSkyLightFrom(point,
			seen, airToward)
		if !usable {
			return maths.Light{}
		}
		if incident == (maths.Light{}) {
			continue
		}
		cosine := math.Max(-1, math.Min(1, towardWater.Dot(viewDirection)))
		phase := phaseChoice.at(cosine, ocean.medium.Phase())
		increment := incident.Scale(phase * weight)
		sum = sum.Add(increment)
		if !finite(phase, sum.R, sum.G, sum.B) {
			markClearInscatterFault(scene, clearInscatterNumericalOverflow)
			return maths.Light{}
		}
	}
	return sum
}

// oceanSkyLightFrom returns the source direction and transmitted radiance
// from one air-side sky direction. The product of refractive radiance scaling
// and Snell's solid-angle Jacobian is T*cosAir/cosWater. Sampling is in dΩ_air,
// hence no additional n² factor belongs in the quadrature weight.
func (scene *staged) oceanSkyLightFrom(from maths.Vec, seen look,
	airToward maths.Vec,
) (towardWater maths.Vec, incident maths.Light, usable bool) {
	ocean := scene.oceanSun
	if ocean == nil || ocean.placedBoundary == nil ||
		!finite(from.X, from.Y, from.Z, airToward.X, airToward.Y,
			airToward.Z) {
		scene.oceanSunFault.Store(oceanSunInvalidPath)
		return maths.Vec{}, maths.Light{}, false
	}
	active := activeClearMedium(seen)
	if active == nil || !sameClearMedium(active, ocean.medium) {
		scene.oceanSunFault.Store(oceanSunInvalidPath)
		return maths.Vec{}, maths.Light{}, false
	}
	airToward = airToward.Unit()
	cosAir := airToward.Y
	if !(cosAir > 0) || !finite(cosAir) {
		scene.oceanSunFault.Store(oceanSunInvalidPath)
		return maths.Vec{}, maths.Light{}, false
	}
	incomingWater, total := refracted(airToward.Neg(), maths.Up,
		ocean.material.Dense())
	if total || incomingWater == (maths.Vec{}) {
		scene.oceanSunFault.Store(oceanSunInvalidPath)
		return maths.Vec{}, maths.Light{}, false
	}
	towardWater = incomingWater.Neg().Unit()
	cosWater := towardWater.Y
	if !(cosWater > 0) || !finite(cosWater) {
		scene.oceanSunFault.Store(oceanSunInvalidPath)
		return maths.Vec{}, maths.Light{}, false
	}
	ray := maths.Ray{At: from, Dir: towardWater}
	waterDistance, intersects := ocean.placedBoundary.meet(ray, 0,
		math.Inf(1))
	if !intersects {
		scene.recordOceanSunPathMiss(ray, from, seen)
		scene.oceanSunFault.Store(oceanSunNoBoundary)
		return maths.Vec{}, maths.Light{}, false
	}
	if !(waterDistance > 0) || !finite(waterDistance) {
		scene.oceanSunFault.Store(oceanSunInvalidPath)
		return maths.Vec{}, maths.Light{}, false
	}
	crossing := ray.Point(waterDistance)
	if !finite(crossing.X, crossing.Y, crossing.Z) {
		scene.oceanSunFault.Store(oceanSunInvalidPath)
		return maths.Vec{}, maths.Light{}, false
	}
	if scene.crowd.blockedExcept(ray, waterDistance, ocean.boundary) ||
		scene.crowd.blockedExcept(maths.Ray{At: crossing,
			Dir: airToward}, math.Inf(1), ocean.boundary) {
		return towardWater, maths.Light{}, true
	}
	radiance := scene.lighting.Light(airToward)
	if radiance == (maths.Light{}) {
		return towardWater, maths.Light{}, true
	}
	transmitted := 1 - ocean.material.SplitsBetween(cosAir,
		ocean.material.Dense())
	// Snell's relation gives dΩ_water/dΩ_air =
	// (n_air/n_water)^2*cosAir/cosWater. Radiance crossing the interface
	// scales by (n_water/n_air)^2, so only the cosine ratio remains.
	factor := transmitted * cosAir / cosWater
	metres := waterDistance * ocean.metresPerUnit
	extinction := ocean.medium.Extinction()
	attenuation := maths.RGB(beerClear(extinction.R, metres),
		beerClear(extinction.G, metres), beerClear(extinction.B, metres))
	incident = radiance.Times(attenuation).Scale(factor)
	incident = incident.Times(paintedClear(scene.volumes, from,
		towardWater, waterDistance))
	if !finite(factor, metres, incident.R, incident.G, incident.B) {
		scene.oceanSunFault.Store(oceanSunInvalidPath)
		return maths.Vec{}, maths.Light{}, false
	}
	return towardWater, incident, true
}
