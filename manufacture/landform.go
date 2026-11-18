package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/thing"
)

// LandformStyle selects a radial authored terrain construction rule.
type LandformStyle string

const (
	MountainMassif LandformStyle = "mountain-massif"
	CalderaVolcano LandformStyle = "caldera-volcano"
)

// LandformSpec fixes the physical radius and summit height above y=0.
// A caldera has an explicit rim radius and depth; a mountain has neither.
// These features are scenery mathematics, not geological simulations.
type LandformSpec struct {
	Style                                     LandformStyle
	Radius, Height, CraterRadius, CraterDepth measure.Length
	Ruggedness                                float64 // zero preserves the original profile; one adds seeded radial relief
}

// DefaultLandform returns a multi-kilometre terrain feature in metres.
func DefaultLandform(style LandformStyle) LandformSpec {
	spec := LandformSpec{Style: style}
	if style != MountainMassif && style != CalderaVolcano {
		return spec
	}
	spec.Radius = measure.Metres(macroValue(string(style) + "-radius"))
	spec.Height = measure.Metres(macroValue(string(style) + "-height"))
	if style == CalderaVolcano {
		spec.CraterRadius = measure.Metres(macroValue("caldera-radius"))
		spec.CraterDepth = measure.Metres(macroValue("caldera-depth"))
	}
	return spec
}

// Landform validates metre dimensions and returns a regular factory recipe.
func Landform(spec LandformSpec) (Kind, error) {
	if spec.Style != MountainMassif && spec.Style != CalderaVolcano {
		return kind{}, fmt.Errorf("manufacture: unknown landform %q", spec.Style)
	}
	if math.IsNaN(spec.Ruggedness) || spec.Ruggedness < 0 || spec.Ruggedness > 1 {
		return kind{}, fmt.Errorf("manufacture: landform ruggedness must be within zero and one")
	}
	for _, length := range []measure.Length{spec.Radius, spec.Height} {
		if !positive(length.InMetres()) || length.InMetres() < 1 || length.InMetres() > macroValue("maximum-length") {
			return kind{}, fmt.Errorf("manufacture: landform dimensions outside finite metre budget")
		}
	}
	radius, depth := spec.CraterRadius.InMetres(), spec.CraterDepth.InMetres()
	if spec.Style == CalderaVolcano {
		if !positive(radius) || !positive(depth) || radius >= spec.Radius.InMetres()*.7 || depth >= spec.Height.InMetres()*.8 {
			return kind{},
				fmt.Errorf("manufacture: caldera needs a rim within its flank and a floor above the base")
		}
	} else if radius != 0 || depth != 0 {
		return kind{}, fmt.Errorf("manufacture: a mountain has no caldera dimensions")
	}
	return kind{name: "landform-" + string(spec.Style), how: landformMaker{spec}}, nil
}

// LandformElevation evaluates the same seeded analytic profile as geometry.
// Signed east/south coordinates are metre offsets; outside the radius is zero.
func LandformElevation(spec LandformSpec, from seed, east,
	south measure.Length) (measure.Length, error) {
	if _, err := Landform(spec); err != nil {
		return measure.Length{}, err
	}
	if !from.made() {
		return measure.Length{}, errGood
	}
	x, z := east.InMetres(), south.InMetres()
	if math.IsNaN(x) || math.IsNaN(z) || math.IsInf(x, 0) || math.IsInf(z, 0) {
		return measure.Length{},
			fmt.Errorf("manufacture: landform sample needs finite metre offsets")
	}
	height := (landformMaker{spec}).elevation(math.Hypot(x, z), math.Atan2(z, x), from)
	return measure.Metres(height), nil
}

type landformMaker struct{ spec LandformSpec }

// made constructs the normal terrain representation.
func (row landformMaker) made(from seed) (good, error) { return row.madeAt(from, "quick") }

// elevation keeps the base, summit and caldera rim exact across all seeds.
func (row landformMaker) elevation(radius, angle float64, from seed) float64 {
	spec := row.spec
	if radius >= spec.Radius.InMetres() {
		return 0
	}
	h := spec.Height.InMetres()
	if spec.Style == CalderaVolcano && radius <= spec.CraterRadius.InMetres() {
		share := radius / spec.CraterRadius.InMetres()
		return h - spec.CraterDepth.InMetres() + spec.CraterDepth.InMetres()*share*share + spec.Ruggedness*spec.CraterDepth.InMetres()*.06*math.Sin(18*math.Pi*share)*math.Sin(math.Pi*share)
	}
	low := 0.0
	if spec.Style == CalderaVolcano {
		low = spec.CraterRadius.InMetres()
	}
	u := (radius - low) / (spec.Radius.InMetres() - low)
	phase := 2 * math.Pi * from.draw(3, 0, 0)
	lobes := 3 + int(from.draw(3, 0, 1)*4)
	rough := 1 - macroValue("landform-roughness-share")*u*(1-u)*(1+math.Cos(float64(lobes)*angle+phase))
	relief := 1 - spec.Ruggedness*.22*math.Sin(math.Pi*u)*(0.5+0.25*math.Sin(37*angle+u*11+phase)+0.25*math.Sin(83*angle-u*29+phase))
	return h * math.Pow(1-u, macroValue("landform-exponent")) * rough * relief
}

// madeAt creates closed top/bottom geometry and batches surface roles.
func (row landformMaker) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	steps, err := macroSegments(detail)
	if err != nil {
		return good{}, err
	}
	if row.spec.Ruggedness > 0 {
		steps = max(steps, 36)
	}
	skins := row.surfaces(from, steps)
	made := good{dressed: looks{
		0: plain("landform stratified rock", .27, .24, .2),
		1: plain("landform summit snow", .78, .83, .87),
		2: &thing.Material{Name: "landform authored caldera lava", Colour: maths.RGB(.18,
			.04, .015), Glow: maths.RGB(.9, .16, .025)},
	}}
	if row.spec.Ruggedness > 0 {
		made.dressed[0].Texture = landformStrata{from.number, row.spec.Radius.InMetres(), row.spec.Height.InMetres()}
	}
	for material := slot(0); material < 3; material++ {
		skin := skins[material]
		if len(skin) > 0 {
			made.parts = append(made.parts, part{shape: macroSurface{skin},
				space: maths.Local(), of: material})
		}
	}
	if made.faces() > int(macroValue("face-budget")) {
		return good{}, errTooMany
	}
	radius, height := row.spec.Radius.InMetres(), row.spec.Height.InMetres()
	extent := maths.Box{Min: maths.V(-radius, 0, -radius), Max: maths.V(radius, height, radius)}
	made.extent = &extent
	return made, nil
}

// macroSurface retains generated mathematical surfaces, never imported assets.
type macroSurface struct{ triangles surface }

// skin returns immutable generated triangles through the standard form path.
func (form macroSurface) skin(cut) surface { return form.triangles }

// surfaces tessellates radial profiles, including the exact caldera rim.
func (row landformMaker) surfaces(from seed, steps int) map[slot]surface {
	spec := row.spec
	around := steps * 4
	radii := []float64{0}
	start := 0.0
	if spec.Style == CalderaVolcano {
		start = spec.CraterRadius.InMetres()
		radii = append(radii, start/2, start)
	}
	for i := 1; i <= steps; i++ {
		radii = append(radii, start+(spec.Radius.InMetres()-start)*float64(i)/float64(steps))
	}
	ring := func(radius float64) []maths.Vec {
		points := make([]maths.Vec, around)
		for i := range points {
			angle := 2 * math.Pi * float64(i) / float64(around)
			points[i] = maths.V(radius*math.Cos(angle), row.elevation(radius, angle, from),
				radius*math.Sin(angle))
		}
		return points
	}
	var all surface
	centre := maths.V(0, row.elevation(0, 0, from), 0)
	previous := ring(radii[1])
	for i, p := range previous {
		all = append(all, face{centre, previous[(i+1)%around], p})
	}
	for _, radius := range radii[2:] {
		next := ring(radius)
		for i, a := range previous {
			j := (i + 1) % around
			all = append(all, face{a, previous[j], next[i]}, face{previous[j], next[j], next[i]})
		}
		previous = next
	}
	for i, p := range previous {
		all = append(all, face{maths.Vec{}, p, previous[(i+1)%around]})
	}
	skins := map[slot]surface{}
	for _, triangle := range all {
		centre := triangle[0].Add(triangle[1]).Add(triangle[2]).Scale(1.0 / 3)
		material := slot(0)
		if spec.Style == MountainMassif && centre.Y > spec.Height.InMetres()*.68 {
			material = 1
		}
		if spec.Style == CalderaVolcano && math.Hypot(centre.X,
			centre.Z) < spec.CraterRadius.InMetres()*.36 && centre.Y > 0 {
			material = 2
		}
		skins[material] = append(skins[material], triangle)
	}
	return skins
}

// landformStrata adds stratified, fractured mineral bands at physical scales.
type landformStrata struct {
	seed           uint64
	radius, height float64
}

// At returns the mineral colour at p, banded by a sine of the height and
// warped by noise.
func (paint landformStrata) At(p maths.Vec) maths.Light {
	x, y, z := p.X/paint.radius, p.Y/paint.height, p.Z/paint.radius
	fault := noise.Gradient(paint.seed, x*18, y*25, z*18)
	grit := noise.Gradient(paint.seed+1, x*140, y*110, z*140)
	bands := .5 + .5*math.Sin(y*110+fault*6)
	return maths.RGB(.12+.20*bands+.03*grit, .115+.15*bands+.025*grit, .11+.11*bands+.02*grit)
}
