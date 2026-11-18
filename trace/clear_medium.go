package trace

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/gas"
	seaocean "github.com/janearc/slayer/scene/ocean"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// ValidateClearState checks the clear-interface world and an explicitly
// supplied camera medium stack without starting a render. Backends use this
// before allocating renderer state so they reject the same ambiguous or
// unsupported optical combinations as the CPU tracer.
func ValidateClearState(world *thing.World, media []InitialMedium) error {
	_, err := PrepareClearState(world, media)
	return err
}

// PrepareClearState validates the clear-interface world and an explicitly
// supplied camera stack, and reports whether any named extinction is present.
// Backends can use the result to select an optional packed transport layout
// without scanning the world a second time.
func PrepareClearState(world *thing.World,
	media []InitialMedium) (bool, error) {
	if world == nil {
		return false, errNoWorld
	}
	hasNamed, err := validateClearWorld(world)
	if err != nil {
		return false, err
	}
	if len(media) == 0 {
		return hasNamed, nil
	}
	if _, _, err := initialMedia(media); err != nil {
		return false, err
	}
	if err := validateInitialClearWorld(world, media); err != nil {
		return false, err
	}
	return hasNamed, nil
}

// maxClearInscatterLamps keeps source work per camera segment bounded. The
// existing lamp selector evaluates every lamp up to lampsEvery and therefore
// has no more than this many exact lamp queries when scattering is enabled.
const maxClearInscatterLamps = lampsEvery

const (
	clearInscatterAmbientUnsupported uint32 = iota + 1
	clearInscatterNumericalOverflow
)

type clearBoundaryKey struct {
	index float64
	rank  int
}

type namedClearBoundary struct {
	medium *thing.ClearMedium
	index  float64
	rank   int
}

// validateClearWorld checks optical identity before any worker starts. A name
// is one immutable optical medium and therefore has one coefficient set,
// distance unit, refractive index and overlap rank in a world.
func validateClearWorld(world *thing.World) (bool, error) {
	if err := validateOceanInterfaceEvents(world); err != nil {
		return false, err
	}
	anyOptical := false
	for index := range world.Things {
		material := world.Things[index].Material
		if material != nil && material.WakeFoam != nil && material.WakeFoam.Coverage != 0 {
			if !material.WakeFoam.Valid() || material.Texture != nil ||
				material.Shine != nil || material.Glow != (maths.Light{}) {
				return false, fmt.Errorf("trace: wake foam requires valid parameters and an unpainted, non-emissive base")
			}
			ocean := world.OceanSurface
			if material.ClearMedium.HasScattering() || (ocean != nil &&
				(ocean.MeanPlaneSource() || ocean.SpotInscatter || ocean.ClearCloudSkySamples > 0)) {
				return false, fmt.Errorf("trace: wake foam cannot be combined with mean-plane or participating source approximations")
			}
		}
		if material != nil && material.ClearMedium.Defined() {
			anyOptical = true
		}
	}
	if !anyOptical {
		return false, nil
	}
	named := map[string]namedClearBoundary{}
	optical := map[clearBoundaryKey]bool{}
	plain := map[clearBoundaryKey]bool{}
	identities := map[clearBoundaryKey]string{}
	for index := range world.Things {
		material := world.Things[index].Material
		if material == nil {
			continue
		}
		key := clearBoundaryKey{index: material.Dense(), rank: material.Rank}
		medium := material.ClearMedium
		if medium.HasScattering() &&
			!oceanScatteringCandidate(world, material) &&
			!oceanSpotScatteringCandidate(world, material) {
			return false, fmt.Errorf("trace: clear-medium in-scattering cannot yet follow light through a transparent boundary")
		}
		if !medium.Defined() {
			if material.Through > 0 {
				plain[key] = true
			}
			continue
		}
		if material.Through <= 0 {
			return false, fmt.Errorf("trace: clear optical medium %q requires a transparent boundary", medium.Name())
		}
		if material.Metal != nil {
			return false, fmt.Errorf("trace: clear optical medium %q cannot be attached to a metal boundary", medium.Name())
		}
		if !finite(material.Dense()) || material.Dense() < 1e-9 || material.Dense() > 1e9 {
			return false, fmt.Errorf("trace: clear optical boundary %q has an invalid refractive index %g", medium.Name(), material.Dense())
		}
		optical[key] = true
		if prior, exists := identities[key]; exists && prior != medium.Name() {
			return false, fmt.Errorf("trace: clear boundaries at index %g rank %d use different medium IDs %q and %q", key.index, key.rank, prior, medium.Name())
		}
		identities[key] = medium.Name()
		definition := namedClearBoundary{medium: medium,
			index: material.Dense(), rank: material.Rank}
		if prior, exists := named[medium.Name()]; exists &&
			(!sameClearMedium(prior.medium, medium) ||
				prior.index != definition.index || prior.rank != definition.rank) {
			return false, fmt.Errorf("trace: clear medium ID %q has mismatched optical properties, index, rank or units", medium.Name())
		}
		named[medium.Name()] = definition
	}
	for key := range optical {
		if plain[key] {
			return false, fmt.Errorf("trace: clear boundaries at index %g rank %d mix named and untagged interiors", key.index, key.rank)
		}
	}
	if anyOptical && len(world.Volumes) > 0 && !clearCloudOnly(world) {
		return false, fmt.Errorf("trace: named clear-medium transport can only be combined with explicitly opted-in clear-cloud volumes")
	}
	if anyOptical && clearCloudNeedsSource(world) {
		if err := validateClearCloudSourceWorld(world); err != nil {
			return false, err
		}
	}
	if anyOptical && len(world.Fields) > 0 {
		return false, fmt.Errorf("trace: named clear-medium transport cannot be combined with gravitational fields yet")
	}
	return true, nil
}

// validateInitialClearWorld checks each initial clear medium against the
// world's transport rules and the identities of its boundaries.
func validateInitialClearWorld(world *thing.World,
	media []InitialMedium) error {
	for _, initial := range media {
		key := clearBoundaryKey{index: initial.Index, rank: initial.Rank}
		if !initial.ClearMedium.Defined() {
			for index := range world.Things {
				material := world.Things[index].Material
				if material != nil && material.Through > 0 &&
					material.ClearMedium.Defined() &&
					material.Dense() == key.index && material.Rank == key.rank {
					return fmt.Errorf("trace: initial clear medium at index %g rank %d omits named optical identity %q", key.index, key.rank, material.ClearMedium.Name())
				}
			}
			continue
		}
		if initial.ClearMedium.HasScattering() {
			if err := validateClearInscatterWorld(world,
				initial.ClearMedium); err != nil {
				return err
			}
		}
		if len(world.Volumes) > 0 && !clearCloudOnly(world) {
			return fmt.Errorf("trace: initial named clear-medium transport can only be combined with explicitly opted-in clear-cloud volumes")
		}
		if len(world.Fields) > 0 {
			return fmt.Errorf("trace: initial named clear-medium transport cannot be combined with gravitational fields yet")
		}
		for index := range world.Things {
			material := world.Things[index].Material
			if material == nil || material.Through <= 0 {
				continue
			}
			if material.ClearMedium.Name() == initial.ClearMedium.Name() &&
				(!sameClearMedium(material.ClearMedium, initial.ClearMedium) ||
					material.Dense() != key.index || material.Rank != key.rank) {
				return fmt.Errorf("trace: initial medium ID %q mismatches its boundary properties", initial.ClearMedium.Name())
			}
			if material.Dense() == key.index && material.Rank == key.rank &&
				!sameClearMedium(material.ClearMedium, initial.ClearMedium) {
				return fmt.Errorf("trace: initial medium at index %g rank %d does not match clear boundary identity", key.index, key.rank)
			}
		}
	}
	return nil
}

// clearCloudOnly allows the narrow CPU composition path for explicitly
// wrapped, finite local clouds. Every other volume combination remains refused.
func clearCloudOnly(world *thing.World) bool {
	if world == nil || len(world.Volumes) == 0 {
		return false
	}
	for index := range world.Volumes {
		volume := world.Volumes[index]
		if volume.Hidden() {
			continue
		}
		if _, ok := volume.Medium.(interface{ ClearMediumOverlay() }); !ok {
			return false
		}
	}
	return true
}

// HasClearCloudOverlay reports an active, explicitly wrapped cloud-only
// composition. Hidden-only worlds do not request mixed cloud transport.
// Internal clearCloudOnly eligibility deliberately permits those empty active
// worlds, so named clear media remain usable when all volumes are hidden.
func HasClearCloudOverlay(world *thing.World) bool {
	if !clearCloudOnly(world) {
		return false
	}
	for _, volume := range world.Volumes {
		if !volume.Hidden() {
			return true
		}
	}
	return false
}

// clearCloudNeedsSource reports whether a visible cloud volume has a colour
// and so needs a sky source.
func clearCloudNeedsSource(world *thing.World) bool {
	if !clearCloudOnly(world) {
		return false
	}
	for _, volume := range world.Volumes {
		if volume.Hidden() {
			continue
		}
		if volume.Material == nil || volume.Material.Colour != (maths.Light{}) {
			return true
		}
	}
	return false
}

// validateClearCloudSourceWorld checks the ocean boundary, sky and extent
// that a clear-cloud sky source needs.
func validateClearCloudSourceWorld(world *thing.World) error {
	ocean := world.OceanSurface
	if ocean == nil || ocean.Boundary == nil || ocean.Boundary.Material == nil ||
		(ocean.ClearCloudSkySamples < 1 && !ocean.DirectMoonMeanPlane) || world.Sky == nil ||
		!oceanClearCloudSkyCandidate(world, ocean.Boundary.Material) {
		return fmt.Errorf("trace: scattering clear-cloud volumes require an explicit ocean clear-cloud-sky-samples source")
	}
	if !ocean.InfiniteMeanPlane {
		field, ok := ocean.Boundary.Shape.(*shape.HeightField)
		if !ocean.DirectMoonMeanPlane || !ok {
			return fmt.Errorf("trace: clear-cloud source requires an infinite continuation or finite calm lunar interface")
		}
		heights, _, _, _, _, _ := field.Pack()
		for _, height := range heights {
			if height != heights[0] {
				return fmt.Errorf("trace: finite clear-cloud lunar interface must be calm and flat")
			}
		}
	} else if _, ok := ocean.Boundary.Shape.(*seaocean.InfiniteMeanContinuation); !ok {
		return fmt.Errorf("trace: clear-cloud sky source requires an actual infinite mean-plane continuation shape")
	}
	if ocean.ExtentMetres <= 0 || ocean.MetresPerUnit <= 0 ||
		!finite(ocean.ExtentMetres, ocean.MetresPerUnit) {
		return fmt.Errorf("trace: clear-cloud sky source requires positive finite ocean extent and metres-per-unit")
	}
	found := false
	for _, each := range world.Things {
		if each == ocean.Boundary {
			found = true
			if each.Hidden() {
				return fmt.Errorf("trace: clear-cloud sky source boundary cannot be hidden")
			}
			continue
		}
		if each.Material != nil && each.Material.Through > 0 {
			return fmt.Errorf("trace: clear-cloud sky source refuses another transparent boundary")
		}
	}
	if !found {
		return fmt.Errorf("trace: clear-cloud sky source boundary must be present in the world")
	}
	if world.Haze.Density > 0 || len(world.Fields) > 0 {
		return fmt.Errorf("trace: clear-cloud sky source refuses separate haze or gravitational fields")
	}
	return nil
}

// validateClearInscatterWorld admits only the explicit source-path
// approximation used by clearInscatter: direct lamps evaluated at each sample,
// with opaque visibility, but no attenuation or refraction through a clear
// interface on the lamp-to-sample path. Refuse other incident media rather
// than silently treating them as air.
func validateClearInscatterWorld(world *thing.World,
	medium *thing.ClearMedium) error {
	if world != nil && world.OceanSurface != nil &&
		world.OceanSurface.Boundary != nil &&
		oceanDiffuseSkyCandidate(world,
			world.OceanSurface.Boundary.Material) {
		return validateOceanScatteringWorld(world, medium)
	}
	if world != nil && world.OceanSurface != nil &&
		world.OceanSurface.Boundary != nil &&
		oceanSpotScatteringCandidate(world,
			world.OceanSurface.Boundary.Material) {
		return validateOceanSpotInscatterWorld(world, medium)
	}
	if world != nil && world.OceanSurface != nil &&
		world.OceanSurface.Boundary != nil &&
		oceanScatteringCandidate(world,
			world.OceanSurface.Boundary.Material) {
		return validateOceanScatteringWorld(world, medium)
	}
	if len(world.Lamps) > maxClearInscatterLamps {
		return fmt.Errorf("trace: clear-medium in-scattering supports at most %d lamps, got %d", maxClearInscatterLamps, len(world.Lamps))
	}
	if world.Sky != nil {
		return fmt.Errorf("trace: clear-medium in-scattering does not yet integrate sky illumination")
	}
	for _, lamp := range world.Lamps {
		switch lamp.(type) {
		case film.Flood, *film.Flood:
			return fmt.Errorf("trace: clear-medium in-scattering refuses a directionless ambient lamp")
		}
	}
	if world.Haze.Density > 0 {
		return fmt.Errorf("trace: clear-medium in-scattering cannot be combined with separate scene haze")
	}
	if len(world.Volumes) > 0 {
		return fmt.Errorf("trace: clear-medium in-scattering cannot be combined with volume transport yet")
	}
	if len(world.Fields) > 0 {
		return fmt.Errorf("trace: clear-medium in-scattering cannot be combined with gravitational fields yet")
	}
	for index := range world.Things {
		material := world.Things[index].Material
		if material != nil && material.Through > 0 {
			return fmt.Errorf("trace: clear-medium in-scattering cannot yet follow light through a transparent boundary")
		}
	}
	return nil
}

// oceanDiffuseSkyCandidate reports whether material is the ocean boundary
// material and takes diffuse sky in-scattering samples.
func oceanDiffuseSkyCandidate(world *thing.World,
	material *thing.Material) bool {
	return world != nil && material != nil && world.OceanSurface != nil &&
		world.OceanSurface.Boundary != nil &&
		world.OceanSurface.Boundary.Material == material &&
		material.ClearMedium != nil &&
		material.ClearMedium.DiffuseSkyInscatterSamples() > 0
}

// oceanClearCloudSkyCandidate reports whether material is the ocean boundary
// material and has a clear-cloud sky source or a direct moon mean plane.
func oceanClearCloudSkyCandidate(world *thing.World,
	material *thing.Material) bool {
	return world != nil && material != nil && world.OceanSurface != nil &&
		world.OceanSurface.Boundary != nil &&
		world.OceanSurface.Boundary.Material == material &&
		material.ClearMedium != nil &&
		(world.OceanSurface.ClearCloudSkySamples > 0 ||
			world.OceanSurface.DirectMoonMeanPlane)
}

// oceanSpotScatteringCandidate reports whether material is the ocean boundary
// material, spot in-scattering is on, and the medium scatters.
func oceanSpotScatteringCandidate(world *thing.World,
	material *thing.Material) bool {
	return world != nil && material != nil && world.OceanSurface != nil &&
		world.OceanSurface.SpotInscatter &&
		world.OceanSurface.Boundary != nil &&
		world.OceanSurface.Boundary.Material == material &&
		material.ClearMedium != nil && material.ClearMedium.HasScattering()
}

// validateOceanSpotInscatterWorld checks the lamps, boundary and volumes
// that ocean spot in-scattering needs.
func validateOceanSpotInscatterWorld(world *thing.World,
	medium *thing.ClearMedium) error {
	ocean := world.OceanSurface
	if ocean == nil || ocean.Boundary == nil ||
		!oceanSpotScatteringCandidate(world, ocean.Boundary.Material) ||
		!sameClearMedium(ocean.Boundary.Material.ClearMedium, medium) {
		return fmt.Errorf("trace: ocean spot in-scattering requires the named participating medium on its explicitly opted-in boundary")
	}
	if len(world.Lamps) > maxClearInscatterLamps {
		return fmt.Errorf("trace: ocean spot in-scattering supports at most %d total lamps, got %d", maxClearInscatterLamps, len(world.Lamps))
	}
	spots := 0
	for _, lamp := range world.Lamps {
		switch source := lamp.(type) {
		case film.Spot:
			spots++
			if !oceanSpotPointsDown(source, ocean.Mean.Y) {
				return fmt.Errorf("trace: ocean spot in-scattering requires selected spots below the mean surface and aimed downward")
			}
		case *film.Spot:
			if source == nil {
				return fmt.Errorf("trace: ocean spot in-scattering refuses a nil spot")
			}
			spots++
			if !oceanSpotPointsDown(*source, ocean.Mean.Y) {
				return fmt.Errorf("trace: ocean spot in-scattering requires selected spots below the mean surface and aimed downward")
			}
		case film.Flood, *film.Flood:
			// Flood remains a surface fill, not a directional scatter source.
		default:
			return fmt.Errorf("trace: ocean spot in-scattering refuses non-spot source %T", lamp)
		}
	}
	if spots == 0 {
		return fmt.Errorf("trace: ocean spot in-scattering requires at least one submerged spot lamp")
	}
	for index := range world.Things {
		one := world.Things[index]
		if one == ocean.Boundary || one.Material == nil || one.Material.Through <= 0 {
			continue
		}
		return fmt.Errorf("trace: ocean spot in-scattering does not support another transparent boundary")
	}
	if world.Haze.Density > 0 || len(world.Volumes) > 0 || len(world.Fields) > 0 {
		return fmt.Errorf("trace: ocean spot in-scattering cannot be combined with haze, volumes or gravitational fields")
	}
	return nil
}

// oceanSpotPointsDown reports whether a spot sits below the mean surface and
// aims down, with its whole cone inside the lower half-space.
func oceanSpotPointsDown(source film.Spot, meanY float64) bool {
	if !(source.At.Y < meanY) || !(source.Aim.Y < 0) ||
		!finite(source.Cone) || source.Cone < 0 || source.Cone > math.Pi/2 {
		return false
	}
	axis := source.Aim.Unit()
	cosine := -axis.Y
	cosine = math.Max(-1, math.Min(1, cosine))
	return math.Acos(cosine)+source.Cone <= math.Pi/2
}

// oceanScatteringCandidate identifies the one explicit source-path route that
// may integrate participating clear medium: scattering attached to the exact
// ocean boundary whose mean-plane direct sun is enabled. Full route, source,
// and scene validation remains in the ocean and render preflight.
func oceanScatteringCandidate(world *thing.World,
	material *thing.Material) bool {
	if world == nil || material == nil || world.OceanSurface == nil ||
		world.OceanSurface.Boundary == nil ||
		world.OceanSurface.Boundary.Material != material ||
		material.ClearMedium == nil ||
		!material.ClearMedium.HasScattering() {
		return false
	}
	return world.OceanSurface.MeanPlaneSource() ||
		material.ClearMedium.DiffuseSkyInscatterSamples() > 0
}

// validateOceanScatteringWorld admits only the tagged direct sun and/or the
// explicitly authored sky quadrature as participating sources. Other lamps
// remain separate surface illumination and are not sampled as in-scattering.
func validateOceanScatteringWorld(world *thing.World,
	medium *thing.ClearMedium) error {
	ocean := world.OceanSurface
	if ocean == nil || ocean.Boundary == nil ||
		!oceanScatteringCandidate(world, ocean.Boundary.Material) ||
		!sameClearMedium(ocean.Boundary.Material.ClearMedium, medium) {
		return fmt.Errorf("trace: ocean single scattering requires the named participating medium on its explicitly tagged direct-sun boundary")
	}
	if err := validateOceanScatteringBoundary(world, ocean.Boundary,
		ocean.ExtentMetres, ocean.MetresPerUnit, medium); err != nil {
		return err
	}
	if ocean.MeanPlaneSource() {
		if err := validateOceanDirectSunWorld(world); err != nil {
			return err
		}
	} else {
		if world.Sky == nil {
			return fmt.Errorf("trace: diffuse-sky in-scattering requires a staged sky")
		}
		if (len(world.Volumes) != 0 && !clearCloudOnly(world)) ||
			len(world.Fields) != 0 ||
			world.Haze.Density > 0 {
			return fmt.Errorf("trace: ocean diffuse-sky in-scattering refuses unmarked volumes, fields or haze transport")
		}
		for _, each := range world.Things {
			if each != ocean.Boundary && each.Material != nil &&
				each.Material.Through > 0 {
				return fmt.Errorf("trace: ocean diffuse-sky in-scattering refuses another transparent boundary")
			}
		}
		if !ocean.InfiniteMeanPlane {
			return fmt.Errorf("trace: diffuse-sky in-scattering requires the explicit infinite mean-plane ocean continuation")
		}
		if _, ok := ocean.Boundary.Shape.(*seaocean.InfiniteMeanContinuation); !ok {
			return fmt.Errorf("trace: diffuse-sky in-scattering requires an actual infinite mean-plane continuation shape")
		}
	}
	if medium.DiffuseSkyInscatterSamples() > 0 {
		if world.Sky == nil {
			return fmt.Errorf("trace: diffuse-sky in-scattering requires a staged sky")
		}
		if !world.OceanSurface.InfiniteMeanPlane {
			return fmt.Errorf("trace: diffuse-sky in-scattering requires the explicit infinite mean-plane ocean continuation")
		}
		if _, ok := world.OceanSurface.Boundary.Shape.(*seaocean.InfiniteMeanContinuation); !ok {
			return fmt.Errorf("trace: diffuse-sky in-scattering requires an actual infinite mean-plane continuation shape")
		}
	}
	return nil
}

// validateOceanScatteringBoundary checks the boundary, extent and units that
// ocean in-scattering needs, and that the boundary is fully clear.
func validateOceanScatteringBoundary(world *thing.World,
	boundary *thing.Thing, extent, metresPerUnit float64,
	medium *thing.ClearMedium) error {
	if boundary == nil || boundary.Shape == nil ||
		!finite(extent, metresPerUnit, world.OceanSurface.Mean.X,
			world.OceanSurface.Mean.Y, world.OceanSurface.Mean.Z) ||
		extent <= 0 || metresPerUnit <= 0 {
		return fmt.Errorf("trace: ocean in-scattering needs a boundary, positive finite extent and positive finite metres-per-unit")
	}
	found := false
	for _, each := range world.Things {
		if each == boundary {
			found = true
		}
		if each != boundary && each.Material != nil &&
			each.Material.Through > 0 {
			return fmt.Errorf("trace: ocean in-scattering refuses another transparent boundary")
		}
	}
	if !found || boundary.Hidden() {
		return fmt.Errorf("trace: ocean in-scattering boundary must be present and visible in the world")
	}
	switch boundary.Shape.(type) {
	case *shape.HeightField, *seaocean.FlatMeanContinuation,
		*seaocean.InfiniteMeanContinuation:
	default:
		return fmt.Errorf("trace: ocean in-scattering requires a supported ocean heightfield boundary")
	}
	material := boundary.Material
	if material == nil || material.Through != 1 || material.Gloss != 1 ||
		material.Metal != nil || material.ClearMedium == nil ||
		!material.ClearMedium.Defined() ||
		material.ClearMedium.MetresPerUnit() != metresPerUnit ||
		!sameClearMedium(material.ClearMedium, medium) {
		return fmt.Errorf("trace: ocean in-scattering needs a fully clear boundary with matching named medium and metre units")
	}
	return nil
}

// sameClearMedium reports whether two clear media have equal values. Two nil
// media are the same; a nil and a non-nil medium are not.
func sameClearMedium(one, other *thing.ClearMedium) bool {
	if one == nil || other == nil {
		return one == nil && other == nil
	}
	return *one == *other
}

// attenuateClearSegment dims pixel by the medium's extinction over the part
// of the ray up to parameter, using the Beer-Lambert law per channel.
func attenuateClearSegment(pixel frame.Pixel, medium *thing.ClearMedium,
	ray maths.Ray, parameter float64) frame.Pixel {
	if !medium.Defined() {
		return pixel
	}
	extinction := medium.Extinction()
	metres := parameter * ray.Dir.Len() * medium.MetresPerUnit()
	red := beerClear(extinction.R, metres)
	green := beerClear(extinction.G, metres)
	blue := beerClear(extinction.B, metres)
	pixel.R = float32(float64(pixel.R) * red)
	pixel.G = float32(float64(pixel.G) * green)
	pixel.B = float32(float64(pixel.B) * blue)
	mostTransmission := math.Max(red, math.Max(green, blue))
	pixel.Cover = float32(1 - (1-float64(pixel.Cover))*mostTransmission)
	return pixel
}

// clearMediumSegment applies the existing exact extinction to the background,
// then adds the bounded first-order source integral from the active medium.
func clearMediumSegment(scene *staged, seen look, pixel frame.Pixel,
	ray maths.Ray, distance float64) frame.Pixel {
	medium := activeClearMedium(seen)
	if !medium.HasScattering() {
		return attenuateClearSegment(pixel, medium, ray, distance)
	}
	kept := attenuateClearSegment(pixel, medium, ray, distance)
	var scattered maths.Light
	if scene.oceanSun != nil &&
		sameClearMedium(scene.oceanSun.medium, medium) {
		scattered = oceanClearInscatter(scene, seen, ray, distance,
			medium)
	} else if scene.oceanSpotInscatter {
		scattered = clearInscatterSources(scene, ray, distance, medium, true)
	} else {
		scattered = clearInscatter(scene, ray, distance, medium)
	}
	if !finite(scattered.R, scattered.G, scattered.B) {
		markClearInscatterFault(scene, clearInscatterNumericalOverflow)
		return kept
	}
	values := [3]float64{float64(kept.R) + scattered.R,
		float64(kept.G) + scattered.G, float64(kept.B) + scattered.B}
	for _, value := range values {
		if !finite(value) || math.Abs(value) > float64(math.MaxFloat32) {
			markClearInscatterFault(scene, clearInscatterNumericalOverflow)
			return kept
		}
	}
	kept.R, kept.G, kept.B = float32(values[0]), float32(values[1]),
		float32(values[2])
	return kept
}

// markClearInscatterFault records fault on the scene if it has a fault slot.
// The first fault recorded is kept.
func markClearInscatterFault(scene *staged, fault uint32) {
	if scene != nil && scene.clearInscatterFault != nil {
		scene.clearInscatterFault.CompareAndSwap(0, fault)
	}
}

// clearInscatter integrates a constant-coefficient source along the finite
// camera segment. Lamp irradiance and phase are sampled at each cell midpoint;
// outgoing extinction is integrated analytically within each cell. Incoming
// attenuation/refraction through clear boundaries is intentionally absent and
// is guarded by validateClearInscatterWorld.
func clearInscatter(scene *staged, ray maths.Ray, distance float64,
	medium *thing.ClearMedium) maths.Light {
	return clearInscatterSources(scene, ray, distance, medium, false)
}

// clearInscatterSources integrates in-scattering along the camera segment.
// Only spot lamps contribute when spotsOnly is true; otherwise all lamps do.
func clearInscatterSources(scene *staged, ray maths.Ray, distance float64,
	medium *thing.ClearMedium, spotsOnly bool) maths.Light {
	if !medium.HasScattering() {
		return maths.Light{}
	}
	directionLength := ray.Dir.Len()
	metresPerParameter := directionLength * medium.MetresPerUnit()
	if !(metresPerParameter > 0) || math.IsNaN(metresPerParameter) ||
		math.IsInf(metresPerParameter, 0) {
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
	if len(scene.lamps) == 0 {
		return maths.Light{}
	}
	pathLimit := medium.MaxInscatterDistanceMetres()
	segmentMetres := math.Min(distance*metresPerParameter, pathLimit)
	if math.IsNaN(segmentMetres) || !(segmentMetres > 0) {
		if math.IsNaN(segmentMetres) {
			markClearInscatterFault(scene, clearInscatterNumericalOverflow)
		}
		return maths.Light{}
	}
	steps := medium.InscatterSteps()
	cellMetres := segmentMetres / float64(steps)
	cellParameters := cellMetres / metresPerParameter
	if !(cellParameters > 0) || !finite(cellParameters) {
		markClearInscatterFault(scene, clearInscatterNumericalOverflow)
		return maths.Light{}
	}
	scattering, extinction := medium.Scattering(), medium.Extinction()
	phaseModel := gas.Gas{Phase: medium.Phase()}
	var result maths.Light
	var room [lampsEvery]lit
	for index := 0; index < steps; index++ {
		beginMetres := float64(index) * cellMetres
		point := ray.Point((float64(index) + 0.5) * cellParameters)
		if !finite(point.X, point.Y, point.Z) {
			markClearInscatterFault(scene, clearInscatterNumericalOverflow)
			return maths.Light{}
		}
		var incident maths.Light
		for _, lamp := range scene.asked(room[:], point,
			ray.Dir.Neg(), byLight) {
			if spotsOnly && !isSpotLamp(scene.lamps[lamp.index]) {
				continue
			}
			got := scene.shadowed(lamp, point)
			if got == (maths.Light{}) {
				continue
			}
			if !finite(got.R, got.G, got.B) {
				markClearInscatterFault(scene, clearInscatterNumericalOverflow)
				return maths.Light{}
			}
			if lamp.ambient() {
				markClearInscatterFault(scene, clearInscatterAmbientUnsupported)
				continue
			}
			cosine := lamp.toward.Unit().Dot(ray.Dir.Unit())
			cosine = math.Max(-1, math.Min(1, cosine))
			phase := phaseModel.PhaseAt(cosine)
			source := got.Scale(phase)
			if !finite(phase, source.R, source.G, source.B) {
				markClearInscatterFault(scene, clearInscatterNumericalOverflow)
				return maths.Light{}
			}
			incident = incident.Add(source)
			if !finite(incident.R, incident.G, incident.B) {
				markClearInscatterFault(scene, clearInscatterNumericalOverflow)
				return maths.Light{}
			}
		}
		for _, channel := range []struct {
			scatter, total, incident float64
			value                    *float64
		}{{scattering.R, extinction.R, incident.R, &result.R},
			{scattering.G, extinction.G, incident.G, &result.G},
			{scattering.B, extinction.B, incident.B, &result.B}} {
			if channel.scatter == 0 {
				continue
			}
			outgoing := math.Exp(-channel.total * beginMetres)
			cellIntegral := clearScatterCell(channel.total, cellMetres)
			sourceFactor := channel.scatter * cellIntegral * outgoing
			if !finite(outgoing, cellIntegral, sourceFactor) {
				markClearInscatterFault(scene, clearInscatterNumericalOverflow)
				return maths.Light{}
			}
			increment := channel.incident * sourceFactor
			value := *channel.value + increment
			if !finite(increment, value) {
				markClearInscatterFault(scene, clearInscatterNumericalOverflow)
				return maths.Light{}
			}
			*channel.value = value
		}
	}
	return result
}

// isSpotLamp reports whether lamp is a spot, by value or by pointer.
func isSpotLamp(lamp thing.Lamp) bool {
	switch lamp.(type) {
	case film.Spot, *film.Spot:
		return true
	default:
		return false
	}
}

// clearScatterCell is the exact integral of exp(-sigma_t*s) over one cell.
func clearScatterCell(extinction, metres float64) float64 {
	if extinction == 0 {
		return metres
	}
	return -math.Expm1(-extinction*metres) / extinction
}

// beerClear returns the transmitted fraction of light over metres through a
// medium with the given extinction.
func beerClear(extinction, metres float64) float64 {
	if extinction == 0 {
		return 1
	}
	return math.Exp(-extinction * metres)
}
