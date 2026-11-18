package set

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/ocean"
	"github.com/janearc/slayer/scene/thing"
	yaml "gopkg.in/yaml.v3"
)

// WaveSurface is a finite sample of an explicitly authored procedural sea.
// Its extent and wave inputs are SI values; Middle is in the world's units.
type WaveSurface struct {
	material                   string
	directSun                  string
	directMoon                 string
	interfaceEvents            float64
	inscatterSource            string
	diffuseSkySamples          float64
	clearCloudSkySamples       float64
	clearCloudWaterStepMode    string
	shadingNormal              string
	motion                     string
	middle                     [3]float64
	extent, wind, fetch, depth float64
	toward, seconds, seed      float64
	samples                    float64
	source                     string
	flatMean                   *FlatMeanOceanContinuation
	surfaceWake                *SurfaceWake
	glint                      OceanGlint
}

// SurfaceWake is a chosen moving surface-height envelope attached to an
// authored craft trajectory. Origin is map east/north/up at time zero;
// heading is clockwise from north. This is not fluid simulation.
type SurfaceWake struct {
	origin                                                   [3]float64
	heading, speed, amplitude, halfWidth, length, wavelength float64
	addToSpectralShading                                     bool
	foam                                                     *SurfaceWakeFoam
}

// SurfaceWakeFoam is an authored diffuse aerated-water response. It is a
// bounded material approximation, not a fluid or cavitation model.
type SurfaceWakeFoam struct {
	coverage, band, satin float64
	reflectance           [3]float64
	patternWavelength     float64
	widthScale            float64
}

// Origin is the wake's craft reference at scene time zero in map coordinates.
func (wake SurfaceWake) Origin() [3]float64 { return wake.origin }

// HeadingRadians is the wake's heading, in radians.
func (wake SurfaceWake) HeadingRadians() float64 { return wake.heading }

// SpeedMetresPerSecond is the craft's speed on the water.
func (wake SurfaceWake) SpeedMetresPerSecond() float64 { return wake.speed }

// AmplitudeMetres is the height of the wake's waves.
func (wake SurfaceWake) AmplitudeMetres() float64 { return wake.amplitude }

// HalfWidthMetres is half the width of the wake.
func (wake SurfaceWake) HalfWidthMetres() float64 { return wake.halfWidth }

// LengthMetres is the length of the wake along its heading.
func (wake SurfaceWake) LengthMetres() float64 { return wake.length }

// WavelengthMetres is the wavelength of the wake's waves.
func (wake SurfaceWake) WavelengthMetres() float64 { return wake.wavelength }

// AddToSpectralShading reports whether the wake adds to the spectral
// shading.
func (wake SurfaceWake) AddToSpectralShading() bool { return wake.addToSpectralShading }

// Foam returns the wake's foam, and whether it has any.
func (wake SurfaceWake) Foam() (SurfaceWakeFoam, bool) {
	if wake.foam == nil {
		return SurfaceWakeFoam{}, false
	}
	return *wake.foam, true
}

// Coverage is the share of the surface the foam covers.
func (foam SurfaceWakeFoam) Coverage() float64 { return foam.coverage }

// Band is the width of the foam band.
func (foam SurfaceWakeFoam) Band() float64 { return foam.band }

// Satin is the foam's satin response.
func (foam SurfaceWakeFoam) Satin() float64 { return foam.satin }

// Reflectance is the foam's reflectance for each colour channel.
func (foam SurfaceWakeFoam) Reflectance() [3]float64 { return foam.reflectance }

// PatternWavelengthMetres is the wavelength of the foam pattern.
func (foam SurfaceWakeFoam) PatternWavelengthMetres() float64 {
	return foam.patternWavelength
}

// WidthScale is the foam's width scale, one where none is set.
func (foam SurfaceWakeFoam) WidthScale() float64 {
	if foam.widthScale == 0 {
		return 1
	}
	return foam.widthScale
}

// OceanGlint is the explicitly authored model of the sun's highlight on
// the ocean, the roughness below one height-field cell. The zero value
// says nothing and leaves the ocean's material exactly as written.
//
// "artistic" takes Roughness as the material's Satin, a look chosen by
// eye and not a measured sea. "cox-munk" derives the slope below one
// cell from the sea's wind and spectrum, for a Beckmann lobe.
type OceanGlint struct {
	model     string
	roughness float64
}

// The glint models a set may name.
const (
	// GlintArtistic is an authored roughness, chosen by eye.
	GlintArtistic = "artistic"
	// GlintCoxMunk is the slope Cox and Munk's wind fit leaves below one
	// cell of the sheet, after the slope its spectrum resolves.
	GlintCoxMunk = "cox-munk"
)

// Model is the named glint model, or empty when the set says none.
func (glint OceanGlint) Model() string { return glint.model }

// Roughness is the authored artistic roughness, from above zero to one,
// and zero for any other model.
func (glint OceanGlint) Roughness() float64 { return glint.roughness }

// FlatMeanOceanContinuation is an explicitly authored CPU far-field collar.
// MeanLevelMetres is relative to the ocean sheet's authored middle elevation;
// CollarWidthMetres is the width added on each side of the finite core.
type FlatMeanOceanContinuation struct {
	meanLevelMetres, collarWidthMetres float64
	extendsToInfinity                  bool
}

// MeanLevelMetres is the flat far-field height relative to the sheet middle.
func (made FlatMeanOceanContinuation) MeanLevelMetres() float64 { return made.meanLevelMetres }

// CollarWidthMetres is the width blended from the core to the flat mean.
func (made FlatMeanOceanContinuation) CollarWidthMetres() float64 { return made.collarWidthMetres }

// ExtendsToInfinity reports whether the authored mean plane continues beyond
// the finite sampled core and collar.
func (made FlatMeanOceanContinuation) ExtendsToInfinity() bool {
	return made.extendsToInfinity
}

// Material is the name of the world's clear interface material.
func (made WaveSurface) Material() string { return made.material }

// DirectSunMode names the explicitly selected direct-light approximation.
// Empty means the legacy untransported sunlight behavior.
func (made WaveSurface) DirectSunMode() string { return made.directSun }

// DirectMoonMode selects the staged Moon's mean-interface source path.
func (made WaveSurface) DirectMoonMode() string { return made.directMoon }

// InterfaceEvents is the explicit CPU ocean-event budget; zero is legacy.
func (made WaveSurface) InterfaceEvents() int { return int(made.interfaceEvents) }

// InscatterSource names the explicitly selected ocean source path, if any.
func (made WaveSurface) InscatterSource() string { return made.inscatterSource }

// DiffuseSkySamples is the explicit upper-hemisphere sky quadrature budget,
// zero when the ocean does not add diffuse sky to its participating medium.
func (made WaveSurface) DiffuseSkySamples() int {
	return int(made.diffuseSkySamples)
}

// ClearCloudSkySamples is the upper-hemisphere source budget for opted-in
// local cloud volumes, independent of clear-water scattering.
func (made WaveSurface) ClearCloudSkySamples() int {
	return int(made.clearCloudSkySamples)
}

// ClearCloudWaterStepMode selects the explicit mixed-medium quadrature budget.
// Empty and per-gap retain the legacy full water budget in every cloud gap.
func (made WaveSurface) ClearCloudWaterStepMode() string { return made.clearCloudWaterStepMode }

// ShadingNormalApproximation is the explicitly named ocean shading-normal
// approximation, or empty when the geometric normal remains in use.
func (made WaveSurface) ShadingNormalApproximation() string {
	return made.shadingNormal
}

// Motion is the explicitly selected evaluation clock; empty preserves the
// authored one-time wave snapshot.
func (made WaveSurface) Motion() string { return made.motion }

// Middle is the sheet's centre in map coordinates converted to world space.
func (made WaveSurface) Middle() maths.Vec { return fromMap(made.middle) }

// MapMiddle is the ocean centre in authored east/north/up map coordinates.
func (made WaveSurface) MapMiddle() [3]float64 { return made.middle }

// Extent is the square's side in metres.
func (made WaveSurface) Extent() float64 { return made.extent }

// Samples is the number of samples on each side, including both edges.
func (made WaveSurface) Samples() int { return int(made.samples) }

// Sea is the authored wind speed, fetch, depth and wind direction. Distances
// are metres, wind is metres per second, and direction is radians from east.
func (made WaveSurface) Sea() (wind, fetch, depth, toward float64) {
	return made.wind, made.fetch, made.depth, made.toward
}

// Seed is the deterministic wave synthesis seed.
func (made WaveSurface) Seed() uint64 { return uint64(made.seed) }

// Seconds is the explicit instant sampled for this static surface.
func (made WaveSurface) Seconds() float64 { return made.seconds }

// Source is the stated provenance of the wave inputs.
func (made WaveSurface) Source() string { return made.source }

// FlatMean returns the optional explicitly authored CPU continuation.
func (made WaveSurface) FlatMean() (FlatMeanOceanContinuation, bool) {
	if made.flatMean == nil {
		return FlatMeanOceanContinuation{}, false
	}
	return *made.flatMean, true
}

// Glint is the authored glint model; its zero value when none is said.
func (made WaveSurface) Glint() OceanGlint { return made.glint }

// SurfaceWake returns the optional authored wake envelope.
func (made WaveSurface) SurfaceWake() (SurfaceWake, bool) {
	if made.surfaceWake == nil {
		return SurfaceWake{}, false
	}
	return *made.surfaceWake, true
}

// waveSurface reads the world's one finite procedural wave sheet.
func (reading reader) waveSurface(_, value *yaml.Node) error {
	var made WaveSurface
	var flatMean FlatMeanOceanContinuation
	var wake SurfaceWake
	var wakeFoam SurfaceWakeFoam
	const fullTurn = 2 * math.Pi
	steps := map[string]step{
		"material":                    reading.text(&made.material),
		"direct-sun":                  reading.word(&made.directSun, "mean-plane"),
		"direct-moon":                 reading.word(&made.directMoon, "mean-plane"),
		"interface-events":            reading.number(&made.interfaceEvents, 0, thing.MaxOceanInterfaceEvents),
		"in-scatter":                  reading.word(&made.inscatterSource, "spot-lamps"),
		"diffuse-sky-samples":         reading.whole(&made.diffuseSkySamples, 1, 256),
		"clear-cloud-sky-samples":     reading.whole(&made.clearCloudSkySamples, 1, 256),
		"clear-cloud-water-step-mode": reading.word(&made.clearCloudWaterStepMode, "per-gap", "ray-spacing"),
		"shading-normal": reading.word(&made.shadingNormal,
			ocean.SmoothNormalBilinearVertexSlopes,
			ocean.SpectralSlopeNormals),
		"middle": reading.three(&made.middle, -farthest, farthest),
		"extent": reading.quantity(&made.extent, "m",
			math.SmallestNonzeroFloat64, farthest),
		"samples": reading.whole(&made.samples, 2,
			float64(ocean.MaxHeightFieldSamples)),
		"sea": reading.block(map[string]step{
			"wind":   reading.quantity(&made.wind, "m/s", 0, farthest),
			"fetch":  reading.quantity(&made.fetch, "m", 0, farthest),
			"depth":  reading.quantity(&made.depth, "m", 0, farthest),
			"toward": reading.quantity(&made.toward, "rad", 0, fullTurn),
		}),
		"seed":   reading.whole(&made.seed, 0, mostSeed),
		"time":   reading.quantity(&made.seconds, "s", 0, farthest),
		"source": reading.source(&made.source),
		"flat-mean": reading.block(map[string]step{
			"mean-level":          reading.quantity(&flatMean.meanLevelMetres, "m", -farthest, farthest),
			"collar-width":        reading.quantity(&flatMean.collarWidthMetres, "m", math.SmallestNonzeroFloat64, farthest),
			"extends-to-infinity": reading.yes(&flatMean.extendsToInfinity),
		}),
		"glint": reading.block(map[string]step{
			"model": reading.word(&made.glint.model, GlintArtistic,
				GlintCoxMunk),
			"roughness": reading.number(&made.glint.roughness,
				math.SmallestNonzeroFloat64, 1),
		}),
		"surface-wake": reading.block(map[string]step{
			"origin":                  reading.three(&wake.origin, -farthest, farthest),
			"heading":                 reading.quantity(&wake.heading, "rad", 0, fullTurn),
			"speed":                   reading.quantity(&wake.speed, "m/s", 0, 100),
			"amplitude":               reading.quantity(&wake.amplitude, "m", math.SmallestNonzeroFloat64, 1),
			"half-width":              reading.quantity(&wake.halfWidth, "m", math.SmallestNonzeroFloat64, 100),
			"length":                  reading.quantity(&wake.length, "m", math.SmallestNonzeroFloat64, farthest),
			"wavelength":              reading.quantity(&wake.wavelength, "m", math.SmallestNonzeroFloat64, farthest),
			"add-to-spectral-shading": reading.boolean(&wake.addToSpectralShading),
			"foam": reading.block(map[string]step{
				"coverage":    reading.number(&wakeFoam.coverage, 0, 1),
				"band":        reading.number(&wakeFoam.band, math.SmallestNonzeroFloat64, 1),
				"reflectance": reading.three(&wakeFoam.reflectance, 0, 1),
				"satin":       reading.number(&wakeFoam.satin, math.SmallestNonzeroFloat64, 1),
				"pattern-wavelength": reading.quantity(&wakeFoam.patternWavelength,
					"m", math.SmallestNonzeroFloat64, farthest),
				"width-scale": reading.number(&wakeFoam.widthScale,
					math.SmallestNonzeroFloat64, 1),
			}),
		}),
	}
	if reading.second {
		steps = reading.with(steps, map[string]step{
			"motion": reading.onlyTwo(reading.word(&made.motion, "scene-clock")),
		})
	}
	err := reading.keys(value, steps, nil)
	if err == nil && valueNode(value, "surface-wake") != nil &&
		valueNode(valueNode(value, "surface-wake"), "foam") != nil {
		wake.foam = &wakeFoam
		err = reading.all(valueNode(valueNode(value, "surface-wake"), "foam"),
			"surface wake foam", "coverage", "band", "reflectance", "satin")
		if wakeFoam.band <= 0 || wakeFoam.satin <= 0 {
			err = fmt.Errorf("surface-wake foam needs positive band and satin")
		}
	}
	if err == nil && valueNode(value, "glint") != nil {
		err = reading.glinted(valueNode(value, "glint"), made.glint)
	}
	if err == nil {
		err = reading.all(value, "an ocean wave surface", "material", "middle",
			"extent", "samples", "sea", "seed", "time", "source")
	}
	if err == nil {
		err = reading.all(valueNode(value, "sea"), "a sea state", "wind",
			"fetch", "depth", "toward")
	}
	if err == nil && valueNode(value, "flat-mean") != nil {
		if made.shadingNormal != "" {
			return fmt.Errorf("ocean shading-normal %q cannot be combined with flat-mean",
				made.shadingNormal)
		}
		err = reading.all(valueNode(value, "flat-mean"), "a flat-mean ocean continuation", "mean-level", "collar-width")
		if err == nil && flatMean.collarWidthMetres > made.extent {
			err = fmt.Errorf("ocean flat-mean collar-width %g m exceeds core extent %g m", flatMean.collarWidthMetres, made.extent)
		}
		if err == nil {
			made.flatMean = &flatMean
		}
	}
	if err == nil && valueNode(value, "surface-wake") != nil {
		err = reading.all(valueNode(value, "surface-wake"), "a surface wake", "origin", "heading", "speed", "amplitude", "half-width", "length", "wavelength", "add-to-spectral-shading")
		if err == nil && wake.length > made.extent*2 {
			err = reading.refuse(valueNode(value, "surface-wake"), "surface-wake length %g m exceeds twice the ocean extent %g m", wake.length, made.extent)
		}
		if err == nil && wake.wavelength > wake.length {
			err = reading.refuse(valueNode(value, "surface-wake"), "surface-wake wavelength %g m exceeds its length %g m", wake.wavelength, wake.length)
		}
		if err == nil && made.shadingNormal == ocean.SpectralSlopeNormals && !wake.addToSpectralShading {
			err = reading.refuse(valueNode(value, "surface-wake"), "spectral-slopes requires surface-wake add-to-spectral-shading: true")
		}
		if err == nil && made.shadingNormal != ocean.SpectralSlopeNormals && wake.addToSpectralShading {
			err = reading.refuse(valueNode(value, "surface-wake"), "surface-wake add-to-spectral-shading: true requires shading-normal: spectral-slopes")
		}
		if err == nil {
			made.surfaceWake = &wake
		}
	}
	if err == nil && made.diffuseSkySamples > 0 {
		if made.flatMean == nil || !made.flatMean.extendsToInfinity {
			err = reading.refuse(valueNode(value,
				"diffuse-sky-samples"),
				"diffuse-sky-samples requires flat-mean extends-to-infinity: true")
		}
	}
	if err == nil && made.clearCloudSkySamples > 0 &&
		(made.flatMean == nil || !made.flatMean.extendsToInfinity) {
		err = reading.refuse(valueNode(value, "clear-cloud-sky-samples"),
			"clear-cloud-sky-samples requires flat-mean extends-to-infinity: true")
	}
	if err == nil && made.directSun != "" && made.directMoon != "" {
		err = reading.refuse(value, "ocean direct-sun and direct-moon are mutually exclusive")
	}
	if err == nil && made.interfaceEvents != math.Trunc(made.interfaceEvents) {
		err = reading.refuse(valueNode(value, "interface-events"), "interface-events requires an integer")
	}
	if err == nil && made.interfaceEvents > 0 &&
		(made.flatMean == nil || !made.flatMean.extendsToInfinity) {
		err = reading.refuse(value, "interface-events requires flat-mean extends-to-infinity: true")
	}
	if err == nil {
		reading.set.world.ocean = &made
	}
	return err
}

// glinted checks that an ocean glint block names its model, and that an
// artistic glint, and only an artistic glint, says a roughness.
func (reading reader) glinted(block *yaml.Node, glint OceanGlint) error {
	if err := reading.all(block, "an ocean glint", "model"); err != nil {
		return err
	}
	artistic, rough := glint.model == GlintArtistic, says(block, "roughness")
	if artistic && !rough {
		return reading.refuse(block, "an artistic glint says its roughness")
	}
	if !artistic && rough {
		return reading.refuse(block, "only an artistic glint takes a "+
			"roughness; %s derives its own", glint.model)
	}
	return nil
}
