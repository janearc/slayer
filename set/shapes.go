package set

import (
	"math"

	"github.com/janearc/slayer/maths"
	yaml "gopkg.in/yaml.v3"
)

// A Sphere is a ball of a material, its middle at a place, so big; and,
// where the set says, carried by a motion, which then says where it is.
type Sphere struct {
	name, material, motion string
	middle                 [3]float64
	radius                 float64
}

// Name is the sphere's name.
func (made Sphere) Name() string { return made.name }

// Material is the name of the material it wears.
func (made Sphere) Material() string { return made.material }

// Middle is where its middle stands before any motion, in the world's
// space.
func (made Sphere) Middle() maths.Vec { return fromMap(made.middle) }

// Radius is its radius, in the world's units.
func (made Sphere) Radius() float64 { return made.radius }

// Motion is the name of the motion that carries it, and whether one
// does.
func (made Sphere) Motion() (string, bool) {
	return made.motion, made.motion != ""
}

// A Wisp is a ball of gas made ragged by noise: thick at its middle and
// thinning to nothing at its radius, and the colour it gives back of the
// light that crosses it.
//
// its noise has a seed that makes its wisps its own, a size for them,
// and how much of the density they may take.
type Wisp struct {
	name                         string
	colour                       Colour
	middle                       [3]float64
	radius, thick, scale, amount float64
	seed                         float64
}

// Name is the wisp's name.
func (made Wisp) Name() string { return made.name }

// Colour is what it gives back of the light.
func (made Wisp) Colour() Colour { return made.colour }

// Middle is where its middle stands, in the world's space.
func (made Wisp) Middle() maths.Vec { return fromMap(made.middle) }

// Ball is its radius, in the world's units, and its density at the
// middle.
func (made Wisp) Ball() (radius, thick float64) {
	return made.radius, made.thick
}

// Ragged is its seed, how big its wisps are, in the world's units, and
// how much of the density they may take.
func (made Wisp) Ragged() (seed uint64, scale, amount float64) {
	return uint64(made.seed), made.scale, made.amount
}

// A Fizz is the mist of bubbles too small to see, over something that
// fizzes: a column standing on its foot, thick where it is born and
// thinning as it climbs and widens, and the colour it gives back of the
// light that crosses it.
//
// the water's churn makes it ragged, and the churn climbs with the
// bubbles: a fizz drawn at two moments is two pictures. One with no
// churn, or that does not rise, is the same picture at every moment.
type Fizz struct {
	name                        string
	colour                      Colour
	foot                        [3]float64
	axis                        *[3]float64
	radius, height, thick, rise float64
	churn, seed                 float64
	clearWater                  bool
	phaseG                      *float64
}

// Name is the fizz's name.
func (made Fizz) Name() string { return made.name }

// Colour is what it gives back of the light.
func (made Fizz) Colour() Colour { return made.colour }

// Foot is where the middle of its foot stands, in the world's space.
func (made Fizz) Foot() maths.Vec { return fromMap(made.foot) }

// Axis is the authored map-space direction of local +Y, or world-up when
// omitted. The flag preserves the legacy placement exactly when absent.
func (made Fizz) Axis() (maths.Vec, bool) {
	if made.axis == nil {
		return maths.Up, false
	}
	return fromMap(*made.axis).Unit(), true
}

// Column is its radius at the foot and how high it rises, in the
// world's units, and its density at the middle of the foot.
func (made Fizz) Column() (radius, height, thick float64) {
	return made.radius, made.height, made.thick
}

// Rise is how fast it climbs, in the world's units a second.
func (made Fizz) Rise() float64 { return made.rise }

// Churned is its seed, and how much of the density the churn may take.
func (made Fizz) Churned() (seed uint64, churn float64) {
	return uint64(made.seed), made.churn
}

// ClearWater opts this local fizz into CPU clear-water/cloud composition.
func (made Fizz) ClearWater() bool { return made.clearWater }

// PhaseAsymmetry is an explicitly authored HG phase choice. Explicit zero
// is isotropic; omission preserves inherited water-phase behavior.
func (made Fizz) PhaseAsymmetry() (float64, bool) { return said(made.phaseG) }

// VaporCavitation is a bounded propeller-tip density field. Its controls are
// explicit visual inputs, not a prediction of Alfa blade loading.
type VaporCavitation struct {
	name, source                                   string
	blades                                         int
	fragmentScale, fragmentContrast, fragmentSeed  float64
	colour                                         Colour
	origin, axis                                   [3]float64
	tipRadius, rpm, slipstream, length             float64
	coreRadius, density, pressureDeficit, recovery float64
	visualOnset, clearWater                        bool
	phaseG                                         *float64
}

// Name is the field's name.
func (made VaporCavitation) Name() string { return made.name }

// Blades is the explicitly authored propeller blade count.
func (made VaporCavitation) Blades() int { return made.blades }

// PhaseAsymmetry is the optional independent HG cloud phase.
func (made VaporCavitation) PhaseAsymmetry() (float64, bool) { return said(made.phaseG) }

// Fragments gives the deterministic advected density breakup controls.
func (made VaporCavitation) Fragments() (seed uint64, scale, contrast float64) {
	return uint64(made.fragmentSeed), made.fragmentScale, made.fragmentContrast
}

// Colour is what the field gives back of the light.
func (made VaporCavitation) Colour() Colour { return made.colour }

// Origin is its map-space origin, in the world's units.
func (made VaporCavitation) Origin() maths.Vec { return fromMap(made.origin) }

// Axis is its downstream map-space direction.
func (made VaporCavitation) Axis() maths.Vec { return fromMap(made.axis) }

// Controls returns tip radius, shaft speed, slipstream speed, and maximum
// downstream length in the world's units and seconds.
func (made VaporCavitation) Controls() (tipRadius, rpm, slipstream, length float64) {
	return made.tipRadius, made.rpm, made.slipstream, made.length
}

// Core returns the filament core radius and its density per metre.
func (made VaporCavitation) Core() (radius, density float64) {
	return made.coreRadius, made.density
}

// PressureRecovery returns the authored visual pressure deficit and recovery
// gradient, which cap the field's downstream persistence.
func (made VaporCavitation) PressureRecovery() (deficitPa, recoveryPaPerM float64) {
	return made.pressureDeficit, made.recovery
}

// VisualOnset is whether this authored approximation is present.
func (made VaporCavitation) VisualOnset() bool { return made.visualOnset }

// ClearWater opts the local field into CPU clear-water composition.
func (made VaporCavitation) ClearWater() bool { return made.clearWater }

// spheres reads each sphere, by its name.
func (reading reader) spheres(_, value *yaml.Node) error {
	return reading.each(value, func(name, block *yaml.Node) error {
		made := Sphere{name: name.Value}
		var source string
		err := reading.keys(block, map[string]step{
			"material": reading.text(&made.material),
			"middle":   reading.three(&made.middle, -farthest, farthest),
			"radius": reading.number(&made.radius,
				math.SmallestNonzeroFloat64, farthest),
			"motion": reading.text(&made.motion),
			"source": reading.source(&source),
		}, nil)
		if err == nil {
			err = reading.all(block, "a sphere", "material", "middle",
				"radius", "source")
		}
		reading.set.world.spheres = append(reading.set.world.spheres, made)
		return err
	})
}

// wisps reads each wisp, by its name.
func (reading reader) wisps(_, value *yaml.Node) error {
	return reading.each(value, func(name, block *yaml.Node) error {
		made := Wisp{name: name.Value}
		var source string
		over := math.SmallestNonzeroFloat64
		err := reading.keys(block, map[string]step{
			"colour": reading.colour(&made.colour),
			"middle": reading.three(&made.middle, -farthest, farthest),
			"radius": reading.number(&made.radius, over, farthest),
			"thick":  reading.number(&made.thick, over, brightest),
			"seed":   reading.whole(&made.seed, 0, mostSeed),
			"scale":  reading.number(&made.scale, over, farthest),
			"amount": reading.number(&made.amount, over, 1),
			"source": reading.source(&source),
		}, nil)
		if err == nil {
			err = reading.all(block, "a wisp", "colour", "middle", "radius",
				"thick", "seed", "scale", "amount", "source")
		}
		reading.set.world.wisps = append(reading.set.world.wisps, made)
		return err
	})
}

// fizzes reads each fizz, by its name.
func (reading reader) fizzes(_, value *yaml.Node) error {
	return reading.each(value, func(name, block *yaml.Node) error {
		made := Fizz{name: name.Value}
		var source string
		var thickName, thickValue *yaml.Node
		over := math.SmallestNonzeroFloat64
		err := reading.keys(block, map[string]step{
			"colour": reading.colour(&made.colour),
			"foot":   reading.three(&made.foot, -farthest, farthest),
			"axis":   reading.optionalTriple(&made.axis, -1, 1),
			"radius": reading.number(&made.radius, over, farthest),
			"height": reading.number(&made.height, over, farthest),
			"thick": func(name, value *yaml.Node) error {
				thickName, thickValue = name, value
				return nil
			},
			"rise":        reading.number(&made.rise, 0, farthest),
			"churn":       reading.number(&made.churn, 0, 1),
			"seed":        reading.whole(&made.seed, 0, mostSeed),
			"clear-water": reading.yes(&made.clearWater),
			"phase-g":     reading.maybe(&made.phaseG, math.Nextafter(-1, 0), math.Nextafter(1, 0)),
			"source":      reading.source(&source),
		}, nil)
		if err == nil {
			err = reading.all(block, "a fizz", "colour", "foot", "radius",
				"height", "thick", "rise", "churn", "seed", "source")
		}
		if err == nil {
			least := over
			if made.clearWater {
				least = 0
			}
			err = reading.number(&made.thick, least, brightest)(thickName, thickValue)
		}
		if err == nil && made.axis != nil && fromMap(*made.axis).Len() < 1e-9 {
			err = reading.refuse(valueNode(block, "axis"), "fizz axis needs a non-zero direction")
		}
		if err == nil && made.phaseG != nil && !made.clearWater {
			err = reading.refuse(valueNode(block, "phase-g"), "phase-g requires clear-water: true")
		}
		reading.set.world.fizzes = append(reading.set.world.fizzes, made)
		return err
	})
}

// vaporCavities reads reusable, bounded, authored propeller-vapor fields.
func (reading reader) vaporCavities(_, value *yaml.Node) error {
	return reading.each(value, func(name, block *yaml.Node) error {
		made := VaporCavitation{name: name.Value}
		var source string
		err := reading.keys(block, map[string]step{
			"colour":              reading.colour(&made.colour),
			"origin":              reading.three(&made.origin, -farthest, farthest),
			"axis":                reading.three(&made.axis, -1, 1),
			"blades":              reading.integer(&made.blades, 1, 16),
			"tip-radius":          reading.number(&made.tipRadius, 0.001, 5),
			"rpm":                 reading.number(&made.rpm, 0.001, 600),
			"slipstream-mps":      reading.number(&made.slipstream, 0.001, 100),
			"length-m":            reading.number(&made.length, 0.001, 200),
			"core-radius-m":       reading.number(&made.coreRadius, 0.001, 2),
			"density-per-m":       reading.number(&made.density, 0, 20),
			"fragment-scale-m":    reading.number(&made.fragmentScale, .001, 20),
			"fragment-contrast":   reading.number(&made.fragmentContrast, 0, 1),
			"fragment-seed":       reading.whole(&made.fragmentSeed, 0, mostSeed),
			"pressure-deficit-pa": reading.number(&made.pressureDeficit, 0.001, 1e6),
			"recovery-pa-per-m":   reading.number(&made.recovery, 0.001, 1e6),
			"visual-onset":        reading.yes(&made.visualOnset),
			"clear-water":         reading.yes(&made.clearWater),
			"phase-g":             reading.maybe(&made.phaseG, math.Nextafter(-1, 0), math.Nextafter(1, 0)),
			"source":              reading.source(&source),
		}, nil)
		if err == nil {
			err = reading.all(block, "a vapor cavity field", "colour", "origin", "axis", "blades",
				"tip-radius", "rpm", "slipstream-mps", "length-m", "core-radius-m",
				"density-per-m", "pressure-deficit-pa", "recovery-pa-per-m",
				"fragment-scale-m", "fragment-contrast", "fragment-seed",
				"visual-onset", "clear-water", "source")
		}
		made.source = source
		if err == nil && made.phaseG != nil && !made.clearWater {
			err = reading.refuse(valueNode(block, "phase-g"), "phase-g requires clear-water: true")
		}
		reading.set.world.vaporCavities = append(reading.set.world.vaporCavities, made)
		return err
	})
}
