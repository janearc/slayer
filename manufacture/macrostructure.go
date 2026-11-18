package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// MacrostructureStyle selects an authored architectural construction rule.
type MacrostructureStyle string

const (
	TerracedArcology MacrostructureStyle = "terraced-arcology"
	TwistedSpires    MacrostructureStyle = "twisted-spires"
	InhabitedBridge  MacrostructureStyle = "inhabited-bridge"
)

// MacrostructureSpec fixes a metre-scale footprint and height. Seed changes
// massing and asymmetry within that envelope. Twist is the end-to-end twist
// of the tallest spire; the other towers take smaller shares of that angle.
// These are scenery rules, not structural engineering or occupied buildings.
type MacrostructureSpec struct {
	Style                MacrostructureStyle
	Width, Depth, Height measure.Length
	Twist                measure.Angle
	Wildness             float64 // zero preserves the original geometry; one is strongly gnarled
	LivingCover          bool    // authored giant branching growth, not an ecology simulation
}

// DefaultMacrostructure gives each family a useful kilometre-scale specimen.
// Dimensions and proportions are authored in macro-choices.yaml.
func DefaultMacrostructure(style MacrostructureStyle) MacrostructureSpec {
	prefix := string(style)
	if style != TerracedArcology && style != TwistedSpires && style != InhabitedBridge {
		return MacrostructureSpec{Style: style}
	}
	return MacrostructureSpec{Style: style, Width: measure.Metres(macroValue(prefix + "-width")),
		Depth:  measure.Metres(macroValue(prefix + "-depth")),
		Height: measure.Metres(macroValue(prefix + "-height")),
		Twist:  measure.Degrees(macroValue("spire-twist-degrees"))}
}

// Macrostructure checks dimensional inputs and returns a normal factory kind.
// Its geometry is in metres: an orbital using 1e6 metres per scene unit places
// the specimen with a uniform scale of 1e-6; no artistic unit inference occurs.
func Macrostructure(spec MacrostructureSpec) (Kind, error) {
	if spec.Style != TerracedArcology && spec.Style != TwistedSpires && spec.Style != InhabitedBridge {
		return kind{}, fmt.Errorf("manufacture: unknown macrostructure %q", spec.Style)
	}
	if math.IsNaN(spec.Wildness) || spec.Wildness < 0 || spec.Wildness > 1 {
		return kind{}, fmt.Errorf("manufacture: macrostructure wildness must be within zero and one")
	}
	for _, length := range []measure.Length{spec.Width, spec.Depth, spec.Height} {
		if !positive(length.InMetres()) || length.InMetres() < 1 || length.InMetres() > macroValue("maximum-length") {
			return kind{},
				fmt.Errorf("manufacture: macrostructure dimensions need finite metres within the recipe budget")
		}
	}
	w, d, h := spec.Width.InMetres(), spec.Depth.InMetres(), spec.Height.InMetres()
	if math.Max(w, math.Max(d, h))/math.Min(w, math.Min(d, h)) > 1000 {
		return kind{}, fmt.Errorf("manufacture: macrostructure aspect exceeds 1000")
	}
	if spec.Style == TwistedSpires && (w/d < .5 || w/d > 2 || h < math.Max(w, d)*.5) {
		return kind{},
			fmt.Errorf("manufacture: twisted cluster needs a balanced footprint and tall spires")
	}
	twist := spec.Twist.InRadians()
	if math.IsNaN(twist) || math.IsInf(twist, 0) || math.Abs(twist) > math.Pi {
		return kind{}, fmt.Errorf("manufacture: macrostructure twist must be within half a turn")
	}
	if spec.Style == InhabitedBridge && (spec.Width.InMetres() < 2*spec.Depth.InMetres() || spec.Width.InMetres() < 2*spec.Height.InMetres()) {
		return kind{},
			fmt.Errorf("manufacture: inhabited bridge needs span at least twice its depth and height")
	}
	return kind{name: "macro-" + string(spec.Style), how: macrostructureMaker{spec}}, nil
}

type macrostructureMaker struct{ spec MacrostructureSpec }

const (
	macroStone slot = iota
	macroGlass
	macroMetal
	macroPark
)

// macroLooks retains separate architectural surface roles for native batching.
func macroLooks() looks {
	return looks{macroStone: plain("macro pale structural ceramic", .55, .58, .57),
		macroGlass: &thing.Material{Name: "macro blue glazing", Colour: maths.RGB(.045,
			.12, .17), Gloss: .72},
		macroMetal: plain("macro dark structural ribs", .13, .17, .2),
		macroPark:  plain("macro elevated planted terraces", .09, .24, .12)}
}

// macroRows registers independent recipes without attaching them to a scene.
func macroRows() map[string]maker {
	rows := map[string]maker{}
	for _, style := range []MacrostructureStyle{TerracedArcology, TwistedSpires,
		InhabitedBridge} {
		rows["macro-"+string(style)] = macrostructureMaker{DefaultMacrostructure(style)}
	}
	for _, style := range []LandformStyle{MountainMassif, CalderaVolcano} {
		rows["landform-"+string(style)] = landformMaker{DefaultLandform(style)}
	}
	return rows
}

// made constructs the default architectural representation.
func (row macrostructureMaker) made(from seed) (good,
	error) {
	return row.madeAt(from, "quick")
}

// madeAt changes representation cost while preserving requested dimensions.
func (row macrostructureMaker) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	segments, err := macroSegments(detail)
	if err != nil {
		return good{}, err
	}
	assembly := instrumentAssembly{fine: cut{around: 6}}
	switch row.spec.Style {
	case TerracedArcology:
		row.arcology(&assembly, from, detail)
	case TwistedSpires:
		row.spires(&assembly, from, segments, detail)
	case InhabitedBridge:
		row.bridge(&assembly, from, segments, detail)
	}
	structural := len(assembly.object.parts)
	if row.spec.LivingCover {
		row.living(&assembly, from)
	}
	if assembly.trouble != nil {
		return good{}, assembly.trouble
	}
	made := assembly.object.in(macroLooks())
	if row.spec.Wildness > 0 || row.spec.LivingCover {
		made = row.weather(made, from, structural)
	}
	limit := int(macroValue("face-budget"))
	if row.spec.Wildness > 0 {
		limit = 100000
	}
	if made.faces() > limit {
		return good{}, errTooMany
	}
	width, depth, height := row.spec.Width.InMetres(), row.spec.Depth.InMetres(),
		row.spec.Height.InMetres()
	extent := maths.Box{Min: maths.V(-width/2, 0, -depth/2), Max: maths.V(width/2,
		height, depth/2)}
	made.extent = &extent
	return made, nil
}

// macroSegments is a bounded tessellation choice, not a size multiplier.
func macroSegments(detail string) (int, error) {
	switch detail {
	case "hull", "quick", "demo":
		return int(macroValue("segments-" + detail)), nil
	}
	return 0, fmt.Errorf("manufacture: macro recipes have no %s representation", detail)
}

// macroBox places one centred, grounded box through the shared constructor.
func macroBox(a *instrumentAssembly, w, h, d float64, at maths.Vec, material slot) {
	a.box(w, h, d, at.Sub(maths.V(w/2, 0, d/2)), material)
}

// macroRing creates a closed four-sided gallery around an actual courtyard.
func macroRing(a *instrumentAssembly, w, d, innerW, innerD, h float64, at maths.Vec,
	material slot) {
	a.add(macroLoft{width: w, depth: d, innerWidth: innerW, innerDepth: innerD,
		height: h, segments: 1},
		maths.NewSubspace(at, maths.Identity, 1), material)
}
