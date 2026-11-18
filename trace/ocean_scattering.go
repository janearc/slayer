package trace

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// oceanClearInscatter integrates first-order source from the tagged refracted
// direct sun and any separately opted-in diffuse sky along a water camera
// segment. The ordinary-ground legacy fill is not an in-scattered source.
func oceanClearInscatter(scene *staged, seen look, ray maths.Ray,
	distance float64, medium *thing.ClearMedium) maths.Light {
	return oceanClearInscatterSteps(scene, seen, ray, distance, medium, medium.InscatterSteps())
}

// oceanClearInscatterSteps changes only this segment's quadrature budget;
// the authored medium identity, coefficients and source paths remain intact.
func oceanClearInscatterSteps(scene *staged, seen look, ray maths.Ray,
	distance float64, medium *thing.ClearMedium, steps int) maths.Light {
	if steps < 1 || steps > medium.InscatterSteps() {
		markClearInscatterFault(scene, clearInscatterNumericalOverflow)
		return maths.Light{}
	}
	if !medium.HasScattering() {
		return maths.Light{}
	}
	directionLength := ray.Dir.Len()
	metresPerParameter := directionLength * medium.MetresPerUnit()
	if !(metresPerParameter > 0) || !finite(metresPerParameter) ||
		!finite(ray.Dir.X, ray.Dir.Y, ray.Dir.Z) {
		markClearInscatterFault(scene, clearInscatterNumericalOverflow)
		return maths.Light{}
	}
	if distance == 0 {
		return maths.Light{}
	}
	if math.IsNaN(distance) || distance < 0 {
		markClearInscatterFault(scene, clearInscatterNumericalOverflow)
		return maths.Light{}
	}
	pathMetres := math.Min(distance*metresPerParameter,
		medium.MaxInscatterDistanceMetres())
	if math.IsNaN(pathMetres) || !(pathMetres > 0) {
		if math.IsNaN(pathMetres) {
			markClearInscatterFault(scene, clearInscatterNumericalOverflow)
		}
		return maths.Light{}
	}
	cellMetres := pathMetres / float64(steps)
	cellParameters := cellMetres / metresPerParameter
	if !(cellParameters > 0) || !finite(cellMetres, cellParameters) {
		markClearInscatterFault(scene, clearInscatterNumericalOverflow)
		return maths.Light{}
	}
	extinction, scattering := medium.Extinction(), medium.Scattering()
	viewDirection := ray.Dir.Unit()
	var result maths.Light
	for index := 0; index < steps; index++ {
		beginMetres := float64(index) * cellMetres
		point := ray.Point((float64(index) + 0.5) * cellParameters)
		if !finite(point.X, point.Y, point.Z) {
			markClearInscatterFault(scene, clearInscatterNumericalOverflow)
			return maths.Light{}
		}
		source, usable := oceanClearSourceAtSample(scene, point, seen,
			viewDirection, medium)
		if !usable {
			if scene.oceanSunFault.Load() == 0 {
				markClearInscatterFault(scene, clearInscatterNumericalOverflow)
			}
			return maths.Light{}
		}
		if source == (maths.Light{}) {
			continue
		}
		for _, channel := range []struct {
			scatter, total, source float64
			value                  *float64
		}{{scattering.R, extinction.R, source.R, &result.R},
			{scattering.G, extinction.G, source.G, &result.G},
			{scattering.B, extinction.B, source.B, &result.B}} {
			if channel.scatter == 0 || channel.source == 0 {
				continue
			}
			outgoing := math.Exp(-channel.total * beginMetres)
			cellIntegral := clearScatterCell(channel.total, cellMetres)
			increment := channel.source * channel.scatter *
				outgoing * cellIntegral
			value := *channel.value + increment
			if !finite(outgoing, cellIntegral, increment, value) {
				markClearInscatterFault(scene, clearInscatterNumericalOverflow)
				return maths.Light{}
			}
			*channel.value = value
		}
	}
	return result
}

// oceanClearSourceAtSample combines independently evaluated directional sun
// and opt-in sky contributions. A blocked sun is a zero source, not a reason
// to skip the sky quadrature.
func oceanClearSourceAtSample(scene *staged, point maths.Vec, seen look,
	viewDirection maths.Vec, medium *thing.ClearMedium,
) (maths.Light, bool) {
	return oceanClearSourceWithPhase(scene, point, seen, viewDirection, medium,
		cloudPhaseSelection{})
}

// A local cloud may select its own phase without changing source visibility,
// medium identity or the water segment's independent scattering law.
func oceanClearSourceWithPhase(scene *staged, point maths.Vec, seen look,
	viewDirection maths.Vec, medium *thing.ClearMedium,
	phaseChoice cloudPhaseSelection,
) (maths.Light, bool) {
	towardSun, sun, usable := scene.oceanSunAtSample(point, seen)
	if !usable {
		return maths.Light{}, false
	}
	var source maths.Light
	if sun != (maths.Light{}) {
		cosine := math.Max(-1, math.Min(1,
			towardSun.Unit().Dot(viewDirection)))
		phase := phaseChoice.at(cosine, medium.Phase())
		source = sun.Scale(phase)
		if !finite(phase, source.R, source.G, source.B) {
			markClearInscatterFault(scene, clearInscatterNumericalOverflow)
			return maths.Light{}, false
		}
	}
	if medium.DiffuseSkyInscatterSamples() > 0 ||
		(scene.oceanSun != nil && len(scene.oceanSun.skyDirections) > 0) {
		skySource := oceanDiffuseSkyWithPhase(scene, point, seen,
			viewDirection, phaseChoice)
		source = source.Add(skySource)
	}
	if !finite(source.R, source.G, source.B) {
		markClearInscatterFault(scene, clearInscatterNumericalOverflow)
		return maths.Light{}, false
	}
	return source, true
}
