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
	spacePanel     slot = spaceSecondary + spaceOne
	spaceGold           = spacePanel + spaceOne
)

// centimetresPerMetre is defined by the SI centi prefix.
const centimetresPerMetre = 100

// SpacecraftDesigns names the supported visual construction templates.
func SpacecraftDesigns() []string {
	return []string{"spitzer", "hubble", "webb", "prima", "roman"}
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
	bounds := row.reach(row.dimensions())
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
	if row.design == "roman" {
		return spacecraftSize{aperture: aperture,
			length: research.Value("roman-deployed-length-metres"),
			radius: research.Value("roman-deployed-width-metres") * spaceHalf,
			span:   research.Value("roman-deployed-width-metres")}
	}
	if row.design == "webb" {
		reference := research.Value("webb-primary-diameter-metres")
		scale := aperture / reference
		radius := aperture * spacecraftValue("webb-bus-radius")
		width := scale * research.Value("webb-shield-width-metres")
		length := scale * research.Value("webb-shield-length-metres")
		return spacecraftSize{aperture: aperture, length: aperture,
			radius: radius, span: width, shieldLength: length}
	}
	if row.design == "prima" {
		return spacecraftSize{aperture: aperture, length: aperture * primaValue("tube-length"),
			radius:       aperture * primaValue("tube-radius"),
			span:         aperture * primaValue("shield-span"),
			shieldLength: aperture * primaValue("shield-length")}
	}
	return spacecraftSize{aperture: aperture,
		length: aperture * spacecraftValue(row.design+"-length"),
		radius: aperture * spacecraftValue(row.design+"-radius"),
		span:   aperture * spacecraftValue(row.design+"-span")}
}

// spacecraftRows builds each telescope through rowOf, so a maturity
// report knows whose measured aperture each is.
func spacecraftRows() map[string]maker {
	return map[string]maker{
		"spitzer": rowOf(func() spacecraft {
			return spacecraft{design: "spitzer", aperture: measure.Metres(
				research.Value("spitzer-primary-diameter-centimetres") /
					centimetresPerMetre)}
		}, "spitzer", "space-observatory"),
		"hubble": rowOf(func() spacecraft {
			return spacecraft{design: "hubble", aperture: measure.Metres(
				research.Value("hubble-primary-diameter-metres"))}
		}, "hubble"),
		"webb": rowOf(func() spacecraft {
			return spacecraft{design: "webb", aperture: measure.Metres(
				research.Value("webb-primary-diameter-metres"))}
		}, "webb"),
		"prima": rowOf(func() spacecraft {
			return spacecraft{design: "prima", aperture: measure.Metres(
				research.Value("prima-primary-diameter-metres"))}
		}, "prima"),
		"roman": rowOf(func() spacecraft {
			return spacecraft{design: "roman", aperture: measure.Metres(
				research.Value("roman-primary-diameter-metres"))}
		}, "roman"),
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
		err := webbSurfaces(webbSurfacesRaw, assembly.object.dressed)
		if err != nil {
			return good{}, err
		}
		row.exposed(assembly, size)
	} else if row.design == "prima" {
		row.prima(assembly, size)
	} else if row.design == "roman" {
		row.roman(assembly, size)
	} else {
		row.enclosed(assembly, size)
	}
	if err := row.shareArrays(assembly); err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	object, err := assembly.finish(row.reach(size))
	if err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	return *object, nil
}

// shareArrays stores the identical cells and covers of the craft's arrays
// once each. Prima carries no array, so it is left as it is.
func (row spacecraft) shareArrays(assembly *instrumentAssembly) error {
	if row.design == "prima" {
		return nil
	}
	return assembly.shareSolar(leaf, romanSolarString, solarCover)
}

// extent is the box of the spacecraft, from its structural dimensions and
// not by making it.
func (row spacecraft) extent(from seed) (maths.Box, error) {
	return row.reach(row.dimensions()), nil
}

// reach uses structural dimensions, independently of tessellation.
func (row spacecraft) reach(size spacecraftSize) maths.Box {
	aperture := size.aperture
	if row.design == "roman" {
		return maths.SegmentBox(
			maths.V(-size.span*spaceHalf,
				-aperture*spacecraftValue("roman-envelope-below"),
				-size.length*spaceHalf),
			maths.V(size.span*spaceHalf,
				aperture*spacecraftValue("roman-envelope-above"),
				size.length*spaceHalf), spaceZero)
	}
	if row.design == "webb" {
		// above the shield: the tower's rise and the telescope on it.
		// below it, on the sun's side: the gap, the bus, and the antenna
		// hung under the bus, taken as deep as it is wide
		top := row.shieldTop(size) +
			aperture*spacecraftValue("webb-tower-rise") + size.length
		under := -aperture * (spacecraftValue("webb-bus-drop") +
			spacecraftValue("webb-bus-height") +
			spacecraftValue("webb-antenna-drop") +
			spacecraftValue("webb-antenna-radius"))
		return maths.SegmentBox(
			maths.V(-size.span*spaceHalf, under,
				-size.shieldLength*spaceHalf),
			maths.V(size.span*spaceHalf, top,
				size.shieldLength*spaceHalf), spaceZero)
	}
	if row.design == "prima" {
		return maths.SegmentBox(
			maths.V(-size.span*spaceHalf, -size.shieldLength*spaceHalf,
				-size.shieldLength*spaceHalf),
			maths.V(size.span*spaceHalf, size.shieldLength*spaceHalf,
				size.shieldLength*spaceHalf), spaceZero)
	}
	bus := aperture * spacecraftValue("bus-height")
	width := math.Max(size.radius+aperture*spacecraftValue("detail-relief"), size.span*spaceHalf)
	top := math.Max(bus+size.length, size.span)
	back := -size.radius
	front := size.radius + aperture*spacecraftValue("detail-relief")
	if row.design == "hubble" {
		top += size.radius / spaceHalf
		front = size.radius*spacecraftValue("hubble-door-out") +
			aperture*spacecraftValue("detail-relief")
	}
	if row.design == "spitzer" {
		panel := spitzerPanels(aperture)
		width = math.Max(width, panel.width+panel.thickness)
		top = math.Max(top, panel.height+panel.extension)
		offset := spacecraftValue("spitzer-panel-offset")
		back = math.Min(back, -aperture*offset-panel.width-panel.thickness)
	}
	back = math.Min(back, -front)
	return maths.SegmentBox(maths.V(-width, spaceZero, back),
		maths.V(width, top, front), spaceZero)
}
