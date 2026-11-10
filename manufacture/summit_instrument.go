package manufacture

import (
	_ "embed"
	"fmt"
	"math"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/research"
)

//go:embed summit-instruments.yaml
var summitRaw []byte

var summitDesigns = readSummits(summitRaw)

// summitInstrument is the private YAML wire for an authored scenery family.
// Published apertures name the research ledger; other lengths are metres.
type summitInstrument struct {
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Kind string `yaml:"kind"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Family string `yaml:"family"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Aperture string `yaml:"aperture"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	FocalRatio string `yaml:"focal-ratio"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	PickedAperture float64 `yaml:"picked-aperture"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Housing string `yaml:"housing"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Mount string `yaml:"mount"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Diameter float64 `yaml:"diameter"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Height float64 `yaml:"height"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	HeightReference string `yaml:"height-reference"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Seat float64 `yaml:"seat"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Length float64 `yaml:"length"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Minimum maths.Vec `yaml:"minimum"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Maximum maths.Vec `yaml:"maximum"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Array []maths.Vec `yaml:"array"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	ArrayCount string `yaml:"array-count"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Ancillary []yardPart `yaml:"ancillary"`
}

// readSummits decodes the private family book through Puzzlemouth.
func readSummits(raw []byte) []summitInstrument {
	var book struct{ Designs []summitInstrument }
	if err := yamlfile.Read("summit-instruments.yaml", raw, &book); err != nil {
		panic(err)
	}
	return book.Designs
}

// summitRows gives each site family a normal seeded-factory name.
func summitRows() map[string]maker {
	rows := map[string]maker{}
	for _, design := range summitDesigns {
		rows[design.Kind] = design
	}
	return rows
}

// summitValue reads shared picked structure and display choices.
func summitValue(name string) float64 { return groundValue("summit-" + name) }

// aperture keeps the explicitly picked Mees proxy distinct from source roles.
func (row summitInstrument) aperture() float64 {
	if row.Aperture != "" {
		return research.Value(row.Aperture)
	}
	return row.PickedAperture
}

// height keeps the published Gemini enclosure height in its source ledger.
func (row summitInstrument) height() float64 {
	if row.HeightReference != "" {
		return research.Value(row.HeightReference)
	}
	return row.Height
}

// made supplies the ordinary quick specimen.
func (row summitInstrument) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt keeps one physical site and pointing across circular tessellation.
func (row summitInstrument) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		//ninja:vow feint value API; Things rejects it
		return good{}, errGood
	}
	if len(row.Array) != spaceZero {
		return row.array(detail)
	}
	assembly := &instrumentAssembly{fine: cut{
		around: int(groundValue("sides-" + detail))},
		object: good{dressed: row.looks()}}
	if err := row.assemble(assembly); err != nil {
		//ninja:vow feint value API; Things rejects it
		return good{}, err
	}
	bounds := maths.SegmentBox(row.Minimum, row.Maximum, spaceZero)
	object, err := assembly.finish(bounds)
	if err != nil {
		//ninja:vow feint value API; Things rejects it
		return good{}, err
	}
	return *object, nil
}

// frame holds the telescope's local optical axis, independently of the house.
func (row summitInstrument) frame() maths.Subspace {
	angle := summitValue("altitude-deg")*maths.Degree - math.Pi*spaceHalf
	return tiltedFrame(maths.YAxis.Scale(row.Seat), angle)
}

// ground adapts existing closed mirror, truss and fork primitives.
func (row summitInstrument) ground() groundObservatory {
	pose := row.frame()
	return groundObservatory{diameter: row.Diameter, height: row.height(),
		aperture: row.aperture(), pointing: &pose}
}

// assemble refuses unknown families instead of returning an empty building.
func (row summitInstrument) assemble(assembly *instrumentAssembly) error {
	row.enclosure(assembly)
	row.mount(assembly)
	if err := row.optics(assembly); err != nil {
		return err
	}
	for _, piece := range row.Ancillary {
		if err := piece.addWith(assembly, summitSlots); err != nil {
			return err
		}
	}
	return assembly.trouble
}

// summitSlots names architectural roles without borrowing the yard colours.
var summitSlots = map[string]slot{
	"shell": body, "trim": summitRoof, "support": spaceStructure,
}

// optics selects distinct visual constructions, not calibrated prescriptions.
func (row summitInstrument) optics(assembly *instrumentAssembly) error {
	switch row.Family {
	case "open", "closed", "camera", "ranging":
		row.reflector(assembly)
	case "off-axis":
		row.offAxis(assembly)
	case "radio":
		row.radio(assembly)
	case "twin":
		row.twin(assembly)
	case "refractor":
		row.solarTube(assembly)
	default:
		return fmt.Errorf("manufacture: unknown summit family %q", row.Family)
	}
	return nil
}
