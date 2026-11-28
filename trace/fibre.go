package trace

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/materials/fibre"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

func (scene *staged) prepareFibres(band maths.Band, inside bool) error {
	found := false
	for _, placed := range scene.crowd.things {
		material := placed.thing.Material
		if material == nil || material.Fibre == nil {
			continue
		}
		found = true
		shaft, meshed := placed.thing.Shape.(*mesh.Mesh)
		if !meshed || shaft == nil || !shaft.HasFibres() || !material.Fibre.Defined() ||
			!finite(placed.scale) || placed.scale <= 0 || placed.thing.Clear() {
			return fmt.Errorf("trace: fibre material %q requires a closed shaft-frame mesh with positive scale and shadow casting", material.Name)
		}
		if !standaloneFibre(material) {
			return fmt.Errorf("trace: fibre material %q combines incompatible surface optics", material.Name)
		}
	}
	if !found {
		return nil
	}
	if band.Nanometres != 0 {
		return fmt.Errorf("trace: fibre materials require the visible RGB renderer")
	}
	if inside || scene.oceanSun != nil || scene.oceanSpotInscatter {
		return fmt.Errorf("trace: fibre materials require the ordinary air-side lighting path")
	}
	for _, lamp := range scene.lamps {
		switch lamp := lamp.(type) {
		case film.Sun, *film.Sun, film.Spot, *film.Spot,
			film.Laser, *film.Laser, emit.Lit, *emit.Lit, emit.Distant, *emit.Distant,
			emit.Volume, *emit.Volume, emit.Beams, *emit.Beams, *emit.Beam:
		case film.Flood:
			scene.fibreFloods = append(scene.fibreFloods, lamp)
		case *film.Flood:
			scene.fibreFloods = append(scene.fibreFloods, *lamp)
		default:
			return fmt.Errorf("trace: fibre lighting does not support lamp %T", lamp)
		}
	}
	return nil
}

func standaloneFibre(material *thing.Material) bool {
	return material.Colour == (maths.Light{}) && material.Texture == nil &&
		material.Shine == nil && material.Glow == (maths.Light{}) &&
		material.Radiant == nil && material.Rough == nil && !material.IndependentEmission &&
		material.Limb == 0 && material.Gloss == 0 && material.Through == 0 &&
		material.Index == 0 && material.ClearMedium == nil && material.Metal == nil &&
		material.Film == nil && material.Satin == 0 && material.SlopeVariance == 0 &&
		material.Crinkle == nil && material.Finish == nil && material.WakeFoam == nil &&
		!material.Subsurface.Enabled() && !material.ThinTissue.Enabled()
}

type fibreSample struct {
	hit     hit
	skin    *mesh.Mesh
	shaft   mesh.FibreHit
	point   maths.Vec
	tangent maths.Vec
	across  maths.Vec // Projected view direction, or a radial axis at an axial hit.
	side    maths.Vec
	event   fibre.Event
}

func fibreSurface(scene *staged, seen look, met hit, model *fibre.Model) frame.Pixel {
	sample, ok := fibreAt(met, seen.ray.Dir.Neg(), model)
	var sum maths.Light
	if ok {
		sum = sample.lamps(scene).Add(sample.environment(scene))
	}
	if scene.walked != nil {
		scene.walked.tell("the ray meets a fibre", "material", met.named(),
			"along", met.dist, "scattered", sum)
	}
	hazed := scene.air(seen.ray.Dir, seen.come+met.dist, sum)
	return pixelOf(hazed, 1, float32(met.dist))
}

func fibreAt(met hit, view maths.Vec, model *fibre.Model) (fibreSample, bool) {
	if !met.on.meshed {
		return fibreSample{}, false
	}
	skin := met.thing.thing.Shape.(*mesh.Mesh)
	form, ok := skin.FibreAt(met.on.met)
	if !ok {
		return fibreSample{}, false
	}
	space := met.thing.at
	tangent := space.Orientation.Out(form.Tangent).Unit()
	radial := space.Orientation.Out(form.Radial).Unit()
	rise := view.Dot(tangent)
	projection := view.Sub(tangent.Scale(rise))
	if projection.Len() < 1e-12 {
		projection = radial
	}
	across := projection.Unit()
	side := tangent.Cross(across).Unit()
	offset := max(-1, min(1, tangent.Dot(radial.Cross(across))))
	event, err := model.At(fibre.View{
		Elevation: math.Asin(max(-1, min(1, rise))), Offset: offset,
	})
	if err != nil {
		return fibreSample{}, false
	}
	return fibreSample{
		hit: met, skin: skin, shaft: form, point: space.Out(form.Point),
		tangent: tangent, across: across, side: side, event: event,
	}, true
}

func (sample fibreSample) angles(direction maths.Vec) fibre.Light {
	return fibre.Light{
		Elevation: math.Asin(max(-1, min(1, sample.tangent.Dot(direction)))),
		Azimuth:   math.Atan2(sample.side.Dot(direction), sample.across.Dot(direction)),
	}
}

func (sample fibreSample) direction(angles fibre.Light) maths.Vec {
	elevation, flat := math.Sincos(angles.Elevation)
	side, across := math.Sincos(angles.Azimuth)
	return sample.tangent.Scale(elevation).
		Add(sample.across.Scale(flat * across)).
		Add(sample.side.Scale(flat * side)).Unit()
}

func (sample fibreSample) lamps(scene *staged) maths.Light {
	var sum maths.Light
	for _, lamp := range scene.lamps {
		switch lamp := lamp.(type) {
		case film.Flood, *film.Flood:
			// The hemispherical source is integrated with the sky below.
			continue
		case film.Sun:
			sum = sum.Add(sample.sun(scene, lamp))
			continue
		case *film.Sun:
			sum = sum.Add(sample.sun(scene, *lamp))
			continue
		}
		way, far, light := lamp.Light(sample.point, maths.Vec{})
		sum = sum.Add(sample.direct(scene, way, far, light))
	}
	return sum
}

func (sample fibreSample) sun(scene *staged, lamp film.Sun) maths.Light {
	var sum maths.Light
	directions := lamp.Spread()
	light := lamp.Strength.Scale(1 / float64(len(directions)))
	for _, way := range directions {
		sum = sum.Add(sample.direct(scene, way, math.Inf(1), light))
	}
	return sum
}

func (sample fibreSample) direct(scene *staged, way maths.Vec, far float64,
	light maths.Light,
) maths.Light {
	if light == (maths.Light{}) || way == (maths.Vec{}) {
		return maths.Light{}
	}
	paths, err := sample.event.Evaluate(sample.angles(way))
	if err != nil {
		return maths.Light{}
	}
	scattered := paths.R.Add(paths.TT).Add(paths.TRT).Add(paths.Higher)
	if scattered == (maths.Light{}) {
		return maths.Light{}
	}
	return light.Times(scattered).Times(sample.visibility(scene, way, far))
}

// A Flood's plane irradiance is that of an even radiance hemisphere.
// Use that original source distribution, with shadows for the fibre.
func fibreFloodRadiance(flood film.Flood, direction maths.Vec) maths.Light {
	up := flood.Up
	if up == (maths.Vec{}) {
		up = maths.Up
	}
	if direction.Dot(up) <= 0 {
		return maths.Light{}
	}
	return flood.Strength.Scale(1 / math.Pi)
}

const fibreEnvironmentSamples = 8

func (sample fibreSample) environment(scene *staged) maths.Light {
	if scene.lighting == nil && len(scene.fibreFloods) == 0 {
		return maths.Light{}
	}
	var sum maths.Light
	rotation := placeShare(sample.point)
	// Irrational rotations decorrelate the three sampling dimensions.
	// Use midpoint strata for the longitudinal polar coordinate.
	for i := range fibreEnvironmentSamples {
		index := float64(i)
		values := [3]float64{
			(index + .5) / fibreEnvironmentSamples,
			wrapUnit(rotation + (index+.5)*.6180339887498949),
			wrapUnit(rotation + (index+.5)*.4142135623730950),
		}
		directions, _ := sample.event.Sample(values)
		for _, path := range directions {
			if path.Weight == (maths.Light{}) {
				continue
			}
			way := sample.direction(path.Light)
			var radiance maths.Light
			if scene.lighting != nil {
				radiance = scene.lighting.Light(way)
			}
			for _, flood := range scene.fibreFloods {
				radiance = radiance.Add(fibreFloodRadiance(flood, way))
			}
			if radiance != (maths.Light{}) {
				reached := sample.visibility(scene, way, math.Inf(1))
				sum = sum.Add(radiance.Times(path.Weight).Times(reached))
			}
		}
	}
	return sum.Scale(1.0 / fibreEnvironmentSamples)
}

func wrapUnit(value float64) float64 {
	share := value - math.Floor(value)
	if share == 0 {
		return .5 // The inverse CDF takes open-interval samples.
	}
	return share
}

func (sample fibreSample) visibility(scene *staged, way maths.Vec, far float64) maths.Light {
	placed := sample.hit.thing
	space := placed.at
	normal := space.Orientation.Out(sample.shaft.Normal).Unit()
	inside := way.Dot(normal) < 0
	if inside {
		normal = normal.Neg()
	}
	// Use a coordinate-roundoff offset, not the scene-scale surface lift:
	// the latter can step across a whole hair and its near neighbours.
	// The mesh point is interpolated on the hit face before placement.
	reach := max(1, math.Abs(sample.point.X), math.Abs(sample.point.Y), math.Abs(sample.point.Z))
	step := 16 * (math.Nextafter(reach, math.Inf(1)) - reach)
	offset := normal.Scale(step)
	from := sample.point.Add(offset)
	far -= way.Dot(offset)
	if far <= 0 {
		return maths.Grey(1)
	}
	ray := maths.Ray{At: from, Dir: way}
	if scene.crowd.blockedExcept(ray, far, placed.thing) ||
		sample.skin.FibreBlocks(space.RayIn(ray), far/placed.scale, sample.shaft.Shaft, inside) {
		return maths.Light{}
	}
	return scene.volumeTransmission(from, way, far)
}
