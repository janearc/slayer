package manufacture

import (
	_ "embed"
	"fmt"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
)

//go:embed yard-equipment.yaml
var yardRaw []byte

var yardDesigns = readYard(yardRaw)

// yardEquipment is a fixed, authored assembly, expressed in metres.
type yardEquipment struct {
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Kind string `yaml:"kind"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	About string `yaml:"about"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Minimum maths.Vec `yaml:"minimum"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Maximum maths.Vec `yaml:"maximum"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Parts []yardPart `yaml:"parts"`
}

// yardPart uses either a closed box or a closed rod, never an open card.
type yardPart struct {
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Material string `yaml:"material"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Box *yardBox `yaml:"box"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Rod *yardRod `yaml:"rod"`
}

type yardBox struct {
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	At maths.Vec `yaml:"at"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Size maths.Vec `yaml:"size"`
}
type yardRod struct {
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	From maths.Vec `yaml:"from"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	To maths.Vec `yaml:"to"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Radius float64 `yaml:"radius"`
}

// readYard uses Puzzlemouth for the immutable scenery book.
func readYard(raw []byte) []yardEquipment {
	var book struct{ Designs []yardEquipment }
	if err := yamlfile.Read("yard-equipment.yaml", raw, &book); err != nil {
		panic(err)
	}
	return book.Designs
}

// yardRows exposes ordinary kinds; public scene placement stays unchanged.
func yardRows() map[string]maker {
	rows := map[string]maker{}
	for _, design := range yardDesigns {
		rows[design.Kind] = design
	}
	return rows
}

// made keeps the usual quick specimen behind the existing maker interface.
func (row yardEquipment) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt varies round members, retaining the same working equipment.
func (row yardEquipment) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		//ninja:vow feint value API; Things rejects it
		return good{}, errGood
	}
	assembly := &instrumentAssembly{fine: cut{
		around: int(groundValue("sides-" + detail))},
		object: good{dressed: yardLooks()}}
	for _, piece := range row.Parts {
		if err := piece.addTo(assembly); err != nil {
			//ninja:vow feint value API; Things rejects it
			return good{}, err
		}
	}
	bounds := maths.SegmentBox(row.Minimum, row.Maximum, spaceZero)
	object, err := assembly.finish(bounds)
	if err != nil {
		//ninja:vow feint value API; Things rejects it
		return good{}, err
	}
	return *object, nil
}

// addTo refuses an ambiguous component instead of silently dropping it.
func (piece yardPart) addTo(assembly *instrumentAssembly) error {
	return piece.addWith(assembly, yardSlots)
}

// addWith lets a facility use structural roles instead of the yard's palette.
func (piece yardPart) addWith(assembly *instrumentAssembly,
	slots map[string]slot,
) error {
	material, known := slots[piece.Material]
	if !known || (piece.Box == nil) == (piece.Rod == nil) {
		return fmt.Errorf("manufacture: invalid yard component %q", piece.Material)
	}
	if piece.Box != nil {
		box := piece.Box
		assembly.box(box.Size.X, box.Size.Y, box.Size.Z, box.At, material)
		return nil
	}
	rod := piece.Rod
	assembly.rod(rod.From, rod.To, rod.Radius, material)
	return nil
}

// yardSlots separates paint, steel, tyres and opaque cab windows.
var yardSlots = map[string]slot{
	"ochre": body, "steel": spaceStructure, "tyre": leaf,
	"window": personalGlass, "white": spaceMirror, "timber": spaceSecondary,
}

// yardLooks reads picked display colours from the same scenery book.
func yardLooks() looks {
	paint := looks{}
	for name, material := range yardSlots {
		paint[material] = plain("yard "+name,
			yardColour(name, "red"), yardColour(name, "green"),
			yardColour(name, "blue"))
	}
	return paint
}

// yardColour reads one picked reflectance channel.
func yardColour(name, channel string) float64 {
	return groundValue("yard-" + name + "-" + channel)
}
