package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/research"
)

// spaceZero is defined as the additive identity.
const spaceZero = 0

// spaceOne is defined as the multiplicative identity.
const spaceOne = 1

// spaceHalf is defined as half a unit.
const spaceHalf = 0.5

// octagonSides is defined by an octagon.
const octagonSides = 8

// coordinatePair is defined by the radius and height of a profile point.
const coordinatePair = 2

// These material identifiers are defined after the existing body and leaf.
const (
	spaceMirror    slot = leaf + spaceOne
	spaceStructure      = spaceMirror + spaceOne
	spaceFoil           = spaceStructure + spaceOne
	spaceSecondary      = spaceFoil + spaceOne
)

// centimetresPerMetre is defined by the SI centi prefix.
const centimetresPerMetre = 100

// SpacecraftDesigns names the supported visual construction templates.
func SpacecraftDesigns() []string {
	return []string{"spitzer", "hubble", "webb"}
}

// Spacecraft scales a visual telescope template to an explicit aperture.
// It constructs scenery, with no thermal, tracking or calibrated optical model.
func Spacecraft(design string, aperture measure.Length) (*Kind, error) {
	known := false
	for _, name := range SpacecraftDesigns() {
		known = known || name == design
	}
	if !known || !positive(aperture.InMetres()) {
		return nil, fmt.Errorf(
			"manufacture: invalid spacecraft design or aperture")
	}
	row := spacecraft{design: design, aperture: aperture}
	bounds := row.extent(row.dimensions())
	if !finite(bounds.Min.Len() + bounds.Max.Len()) {
		return nil, fmt.Errorf(
			"manufacture: non-finite spacecraft envelope")
	}
	return &kind{name: "space-observatory", how: row}, nil
}

type spacecraft struct {
	design   string
	aperture measure.Length
}

type spacecraftSize struct {
	aperture, length, radius, span, shieldLength float64
}

// dimensions combines the declared aperture with its template proportions.
func (row spacecraft) dimensions() spacecraftSize {
	aperture := row.aperture.InMetres()
	if row.design == "webb" {
		reference := research.Value("webb-primary-diameter-metres")
		scale := aperture / reference
		radius := aperture * spacecraftValue("webb-bus-radius")
		width := scale * research.Value("webb-shield-width-metres")
		length := scale * research.Value("webb-shield-length-metres")
		return spacecraftSize{aperture: aperture, length: aperture,
			radius: radius, span: width, shieldLength: length}
	}
	return spacecraftSize{aperture: aperture,
		length: aperture * spacecraftValue(row.design+"-length"),
		radius: aperture * spacecraftValue(row.design+"-radius"),
		span:   aperture * spacecraftValue(row.design+"-span")}
}

// spacecraftRows supplies conveniences with source-backed default apertures.
func spacecraftRows() map[string]maker {
	spitzer := research.Value("spitzer-primary-diameter-centimetres") /
		centimetresPerMetre
	return map[string]maker{
		"spitzer": spacecraft{design: "spitzer",
			aperture: measure.Metres(spitzer)},
		"hubble": spacecraft{design: "hubble", aperture: measure.Metres(
			research.Value("hubble-primary-diameter-metres"))},
		"webb": spacecraft{design: "webb", aperture: measure.Metres(
			research.Value("webb-primary-diameter-metres"))},
	}
}

// made retains quick as the default representation.
func (row spacecraft) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt implements the existing value-returning maker interface. Empty
// specimens are refused by Things, so its error result cannot render.
func (row spacecraft) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, errGood
	}
	assembly := &instrumentAssembly{fine: cut{
		around: int(spacecraftValue("sides-" + detail))},
		object: good{dressed: spacecraftLooks(row.design)}}
	size := row.dimensions()
	if row.design == "webb" {
		row.exposed(assembly, size)
	} else {
		row.enclosed(assembly, size)
	}
	object, err := assembly.finish(row.extent(size))
	if err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	return *object, nil
}

// extent uses structural dimensions, independently of tessellation.
func (row spacecraft) extent(size spacecraftSize) maths.Box {
	aperture := size.aperture
	if row.design == "webb" {
		lift := spacecraftValue("webb-bus-lift")
		height := spacecraftValue("webb-bus-height")
		top := row.shieldTop(size) + aperture*(lift+height) +
			size.length
		return maths.SegmentBox(
			maths.V(-size.span*spaceHalf, spaceZero,
				-size.shieldLength*spaceHalf),
			maths.V(size.span*spaceHalf, top,
				size.shieldLength*spaceHalf), spaceZero)
	}
	bus := aperture * spacecraftValue("bus-height")
	width := math.Max(size.radius, size.span*spaceHalf)
	top := math.Max(bus+size.length, size.span)
	back := -size.radius
	if row.design == "spitzer" {
		width = math.Max(size.radius, aperture*
			spacecraftValue("spitzer-panel-width")*spaceHalf)
		offset := spacecraftValue("spitzer-panel-offset")
		thickness := spacecraftValue("panel-thickness")
		back = math.Min(back, -aperture*(offset+thickness*spaceHalf))
	}
	return maths.SegmentBox(maths.V(-width, spaceZero, back),
		maths.V(width, top, size.radius), spaceZero)
}
