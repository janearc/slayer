package manufacture

import (
	_ "embed"
	"fmt"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/materials"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
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
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Stairs *stairFlight `yaml:"stairs"`
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
	pieces := row.Parts
	if row.Stairs != nil {
		built, err := row.stairPieces(row.Stairs, assembly.object.dressed)
		if err != nil {
			//ninja:vow feint value API; Things rejects it
			return good{}, err
		}
		pieces = built
	}
	for _, piece := range pieces {
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

// stairPieces lays a booked flight out as yard parts, after checking
// what its families wear: a flight names a material and a finish for
// every family, the material must be a name the yard's palette
// carries, and the finish must be the paint the yard's look for that
// material actually wears, so a flight cannot quietly name a finish
// nobody mixed.
//
// The flight's own parts are refused by name instead of
// built when its parameters are nonsense.
func (row yardEquipment) stairPieces(flight *stairFlight,
	dressed looks,
) ([]yardPart, error) {
	for _, worn := range flight.wears() {
		material, known := yardSlots[worn.material]
		if !known {
			return nil, fmt.Errorf(
				"manufacture: stairs %s made of %q, which the yard does not carry",
				worn.family, worn.material)
		}
		if worn.finish != paintFinish {
			return nil, fmt.Errorf(
				"manufacture: stairs %s finished %q; the yard paints",
				worn.family, worn.finish)
		}
		if look, mixed := dressed[material]; !mixed || look == nil ||
			look.Metal != nil {
			return nil, fmt.Errorf(
				"manufacture: stairs %s look is not painted", worn.family)
		}
	}
	return flight.parts()
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

// yardLooks reads picked display colours from the same scenery book and
// makes each look of its material painted in that colour: the yard is
// what it is made of, with a coat of pigment over it. The made look's
// Name is set to "yard <name>" afterwards, so the geometry records,
// which hash the look's name and its eight numbers, do not move: paint
// is flat, and a plain look was flat too.
func yardLooks() looks {
	paint := looks{}
	for name, material := range yardSlots {
		kind, err := materials.ByName(name)
		if err != nil {
			panic(fmt.Sprintf("yard looks: %v", err))
		}
		coat, err := thing.Made(kind, thing.Painted(maths.RGB(
			yardColour(name, "red"), yardColour(name, "green"),
			yardColour(name, "blue"))))
		if err != nil {
			panic(fmt.Sprintf("yard looks: %v", err))
		}
		coat.Name = "yard " + name
		paint[material] = coat
	}
	return paint
}

// yardColour reads one picked reflectance channel.
func yardColour(name, channel string) float64 {
	return groundValue("yard-" + name + "-" + channel)
}
