package manufacture

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
)

// retailSeeds is how many seeds the frontage is checked over.
const retailSeeds = 60

// retailEdgeSlack is how far a placed corner may stand past a frontage's
// ends, in metres, over the bleed a unit's shell carries.
const retailEdgeSlack = 1e-9

// retailMostLamps is the most lamps a unit may have at any level: the
// tracer samples one lamp a ray, so a street of units has to keep each
// to a few.
const retailMostLamps = 8

// retailUnitWidth is the width of a named unit at its least, in metres.
func retailUnitWidth(name string) float64 {
	wide, _ := retailBookRead.Units[name].size()
	return wide
}

// retailBroken is the data with one thing changed by change, read again.
func retailBroken(t *testing.T, change func(*retailBook)) error {
	t.Helper()
	book, err := readRetailBook(retailUnitsData)
	if err != nil {
		t.Fatal("the shipped data is not whole:", err)
	}
	change(&book)
	return book.check()
}

// The data read as shipped is whole, and has the units the brief asks
// for.
func TestRetailDataIsWhole(t *testing.T) {
	if retailBookError != nil {
		t.Fatal(retailBookError)
	}
	for _, name := range []string{"ramen", "dive", "shuttered", "vending",
		"konbini", "izakaya", "arcade", "pharmacy"} {
		if _, known := retailBookRead.Units[name]; !known {
			t.Errorf("no unit %q in the data", name)
		}
	}
}

// The kinds are in the table under their names.
func TestFrontageKindsAreRegistered(t *testing.T) {
	for _, name := range []string{"frontage", "frontage-one", "frontage-ramen",
		"frontage-dive", "retail-block"} {
		if _, err := KindNamed(name); err != nil {
			t.Errorf("kind %s: %v", name, err)
		}
	}
}

// A run has as many units as the data says, and no two of a kind stand
// side by side, over many seeds.
func TestFrontageNeighboursDiffer(t *testing.T) {
	for number := range uint64(retailSeeds) {
		names, err := FrontageUnits(number)
		if err != nil {
			t.Fatal(err)
		}
		if len(names) != retailBookRead.Run.Units {
			t.Errorf("seed %d: %d units, want %d", number, len(names),
				retailBookRead.Run.Units)
		}
		for index := 1; index < len(names); index++ {
			if names[index] == names[index-1] {
				t.Errorf("seed %d: two %s side by side", number, names[index])
			}
		}
	}
}

// The same seed draws the same run, and the same faces.
func TestFrontageIsTheSameForASeed(t *testing.T) {
	first, err := frontageRow{}.madeAt(Sow(17), "demo")
	if err != nil {
		t.Fatal(err)
	}
	second, err := frontageRow{}.madeAt(Sow(17), "demo")
	if err != nil {
		t.Fatal(err)
	}
	if first.faces() != second.faces() || len(first.lights) != len(second.lights) {
		t.Error("one seed made two different frontages")
	}
}

// More detail is more faces: a lit opening, then its counter and door,
// then its stools and bowls.
func TestFrontageDetailGrows(t *testing.T) {
	rows := map[string]frontageRow{"frontage-ramen": {only: "ramen"},
		"frontage-dive": {only: "dive"}, "frontage": {}}
	for _, name := range retailBookRead.unitNames() {
		rows["frontage-"+name] = frontageRow{only: name}
	}
	for name, row := range rows {
		hull, errHull := row.madeAt(Sow(17), "hull")
		quick, errQuick := row.madeAt(Sow(17), "quick")
		demo, errDemo := row.madeAt(Sow(17), "demo")
		if err := errors.Join(errHull, errQuick, errDemo); err != nil {
			t.Fatal(name, err)
		}
		if !(hull.faces() < quick.faces() && quick.faces() < demo.faces()) {
			t.Errorf("%s: faces %d, %d, %d are not growing", name,
				hull.faces(), quick.faces(), demo.faces())
		}
	}
}

// Every unit that has a room throws light, so that the pavement is lit
// by what is inside, and none has more lamps than the budget, at any
// level, over many seeds.
func TestFrontageUnitsAreLit(t *testing.T) {
	for _, name := range retailBookRead.unitNames() {
		for number := range uint64(retailSeeds) {
			made, err := frontageRow{only: name}.madeAt(Sow(number), "demo")
			if err != nil {
				t.Fatal(name, err)
			}
			if len(made.lights) < 1 || len(made.lights) > retailMostLamps {
				t.Errorf("unit %s seed %d has %d lamps", name, number,
					len(made.lights))
			}
		}
	}
}

// Nothing a unit draws, stickers and props included, stands past its own
// width: a prop never leans into its neighbour, however wide the seed
// draws the unit.
func TestFrontageUnitsStayInTheirBays(t *testing.T) {
	for _, name := range retailBookRead.unitNames() {
		for number := range uint64(retailSeeds) {
			builder, err := frontageRow{only: name}.built(Sow(number), "demo")
			if err != nil {
				t.Fatal(name, err)
			}
			for material, skin := range builder.skins {
				for _, one := range skin {
					for _, corner := range one {
						if corner.X < -retailBleed-retailEdgeSlack ||
							corner.X > builder.left+retailBleed+retailEdgeSlack {
							t.Fatalf("unit %s seed %d: %s at x %g, past %g",
								name, number, material, corner.X, builder.left)
						}
					}
				}
			}
		}
	}
}

// A sign slot lies in the unit it belongs to, and only a unit with a
// sign has one, in order along the front.
func TestFrontageSlotsLieInTheirUnits(t *testing.T) {
	for number := range uint64(retailSeeds) {
		names, err := FrontageUnits(number)
		if err != nil {
			t.Fatal(err)
		}
		slots, err := FrontageSlots(number)
		if err != nil {
			t.Fatal(err)
		}
		builder, err := frontageRow{}.built(Sow(number), "hull")
		if err != nil {
			t.Fatal(err)
		}
		if builder.left < float64(len(names))*cityValue("bay-pitch") {
			t.Fatalf("seed %d: a front of %g m for %d units", number,
				builder.left, len(names))
		}
		next := 0
		for _, name := range names {
			if retailBookRead.Units[name].Sign == nil {
				continue
			}
			if next >= len(slots) || slots[next].Unit != name {
				t.Fatalf("seed %d: no slot for %s", number, name)
			}
			slot := slots[next]
			start := slot.Place.Origin.X
			if !(slot.Width > 0 && slot.Height > 0 && start >= 0 &&
				start+slot.Width <= builder.left+retailEdgeSlack) {
				t.Errorf("seed %d: the %s slot is outside the front", number,
					name)
			}
			next++
		}
		if next != len(slots) {
			t.Errorf("seed %d: %d slots, %d signs", number, len(slots), next)
		}
	}
}

// A level that is not drawn is refused, and so is the clumps level the
// city has and a frontage does not.
func TestFrontageRefusesAnUnknownLevel(t *testing.T) {
	for _, detail := range []string{"clumps", "street", "demo "} {
		if _, err := (frontageRow{}).madeAt(Sow(1), detail); err == nil {
			t.Errorf("level %q was made", detail)
		}
	}
}

// A seed that was not sown makes nothing.
func TestFrontageRefusesAnUnsownSeed(t *testing.T) {
	if _, err := (frontageRow{}).madeAt(seed{}, "quick"); !errors.Is(err,
		errGood) {
		t.Errorf("an unsown seed: %v", err)
	}
}

// A unit type the data does not have is refused, and the refusal says
// which there are.
func TestFrontageRefusesAnUnknownUnit(t *testing.T) {
	_, err := frontageRow{only: "sushi"}.units(Sow(1))
	if err == nil || !strings.Contains(err.Error(), "ramen") {
		t.Errorf("an unknown unit: %v", err)
	}
}

// A unit that may be drawn at more than one width is drawn at more than
// one, over many seeds, and one that may not is always its own.
func TestRetailUnitsVaryInWidth(t *testing.T) {
	for _, name := range retailBookRead.unitNames() {
		unit := retailBookRead.Units[name]
		seen := map[float64]bool{}
		for number := range uint64(retailSeeds) {
			builder, err := frontageRow{only: name}.built(Sow(number), "hull")
			if err != nil {
				t.Fatal(name, err)
			}
			seen[builder.left] = true
		}
		want := unit.mostBays() - unit.Bays + 1
		if len(seen) != want {
			t.Errorf("unit %s: %d widths over %d seeds, want %d", name,
				len(seen), retailSeeds, want)
		}
	}
}

// Twenty ramen counters from twenty seeds are twenty places: no two are
// the same in width, colours, options and faces together.
func TestRetailRamenCountersDiffer(t *testing.T) {
	seen := map[string]bool{}
	for number := range uint64(retailSeeds) {
		builder, err := frontageRow{only: "ramen"}.built(Sow(number), "demo")
		if err != nil {
			t.Fatal(err)
		}
		faces := 0
		for _, skin := range builder.skins {
			faces += len(skin)
		}
		seen[fmt.Sprint(builder.left, builder.picks, faces,
			len(builder.lamps))] = true
	}
	if len(seen) < retailSeeds/2 {
		t.Errorf("%d ramen counters from %d seeds", len(seen), retailSeeds)
	}
}

// A palette's pick is one of its materials, and every palette is picked
// from, over many seeds.
func TestRetailPalettesPickTheirOwn(t *testing.T) {
	for number := range uint64(retailSeeds) {
		builder, err := frontageRow{only: "ramen"}.built(Sow(number), "hull")
		if err != nil {
			t.Fatal(err)
		}
		for name, list := range retailBookRead.Palettes {
			if !slices.Contains(list, builder.picks[name]) {
				t.Errorf("seed %d: %q is not in palette %s", number,
					builder.picks[name], name)
			}
		}
	}
}

// An option the unit is too narrow for is not offered: a vending unit of
// one bay has no third machine.
func TestRetailOptionsAskForTheirBays(t *testing.T) {
	builder := &retailBuilder{book: retailBookRead,
		width: retailUnitWidth("vending")}
	unit := retailBookRead.Units["vending"]
	for number := range uint64(retailSeeds) {
		for _, kit := range builder.kits(0, unit, Sow(number)) {
			for _, piece := range kit.Parts {
				if strings.Contains(piece.Says, "third machine") {
					t.Fatalf("seed %d: a narrow unit has a third machine",
						number)
				}
			}
		}
	}
}

// A part carried along or filling moves as it should, and the data it
// came from is left as it was.
func TestRetailPartsSlide(t *testing.T) {
	box := retailPart{Material: "wood", Box: []float64{1, 0, 0, 2, 1, 1},
		Along: retailRight}
	if moved := box.slid(3); moved.Box[0] != 4 || moved.Box[retailBoxRight] != 5 {
		t.Errorf("a part carried right: %v", moved.Box)
	}
	if box.Box[0] != 1 {
		t.Error("carrying a part changed the data")
	}
	fill := retailPart{Material: "wood", Box: []float64{1, 0, 0, 2, 1, 1},
		Along: retailFill}
	if grown := fill.slid(3); grown.Box[retailBoxRight] != 5 {
		t.Errorf("a part filling: %v", grown.Box)
	}
	repeat := retailPart{Material: "wood", Box: []float64{1, 0, 0, 2, 1, 1},
		Count: 4, Step: []float64{1, 0, 0}, Along: retailFill}
	if grown := repeat.slid(3.5); grown.Count != 7 {
		t.Errorf("a part repeating: %d places", grown.Count)
	}
	if repeat.Count != 4 {
		t.Error("filling a part changed the data")
	}
	still := retailPart{Material: "wood", Box: []float64{1, 0, 0, 2, 1, 1}}
	if same := still.slid(3); same.Box[0] != 1 {
		t.Error("a part that stays moved")
	}
}

// A slab is a closed piece the mesh takes, of twelve faces, and a slab
// with no thickness is refused by the data and not drawn.
func TestRetailSlabIsClosed(t *testing.T) {
	skin := retailSlab(0, 1, 0, 0, 1, -0.4, 0.04)
	if len(skin) != 12 {
		t.Errorf("a slab of %d faces", len(skin))
	}
	if _, err := skin.meshed(); err != nil {
		t.Errorf("the mesh refused a slab: %v", err)
	}
	err := retailBroken(t, func(book *retailBook) {
		module := book.Modules["awning"]
		module.Parts = append(slices.Clone(module.Parts), retailPart{
			Material: "steel", Slab: []float64{0, 1, 0, 0, 1, -0.4, 0}})
		book.Modules["awning"] = module
	})
	if err == nil {
		t.Error("a slab of no thickness was taken")
	}
}

// Glass is clear: a pane lets the room through.
func TestRetailGlassIsClear(t *testing.T) {
	look := retailBookRead.look("glass")
	if look.Through != 1 || look.Index == 0 {
		t.Errorf("glass lets %g through at index %g", look.Through,
			look.Index)
	}
}

// A block is the building and shops at its foot: more parts than the
// building, lights from the shops, and its front stands out past the
// building's by the podium.
func TestRetailBlockStandsShopsOnABuilding(t *testing.T) {
	made, err := retailBlockRow{}.madeAt(Sow(5), "quick")
	if err != nil {
		t.Fatal(err)
	}
	if len(made.lights) < 1 {
		t.Error("a block has no shop lights")
	}
	building, err := retailBlockRow{}.building(Sow(5), "quick")
	if err != nil {
		t.Fatal(err)
	}
	podium := retailBookRead.Grounds.Podium
	got := made.Bounds().Max.Z - building.Bounds().Max.Z
	if got < podium-retailEdgeSlack {
		t.Errorf("shops stand %g m out, want %g", got, podium)
	}
}

// A block is the same for a seed.
func TestRetailBlockIsTheSameForASeed(t *testing.T) {
	first, err := retailBlockRow{}.madeAt(Sow(9), "hull")
	if err != nil {
		t.Fatal(err)
	}
	second, err := retailBlockRow{}.madeAt(Sow(9), "hull")
	if err != nil {
		t.Fatal(err)
	}
	if first.faces() != second.faces() || len(first.lights) != len(second.lights) {
		t.Error("one seed made two different blocks")
	}
}

// A block's units fit along its building, and no two of a type stand
// side by side.
func TestRetailBlockUnitsFit(t *testing.T) {
	for number := range uint64(retailSeeds) {
		room := 40.0
		placed, used, err := retailBlockRow{}.placed(Sow(number), 1, room)
		if err != nil {
			t.Fatal(err)
		}
		if used > room+retailEdgeSlack || len(placed) < 1 {
			t.Errorf("seed %d: %d units use %g of %g m", number, len(placed),
				used, room)
		}
	}
}

// A unit of the pool is drawn once and shared: the same key is the same
// unit, and its meshes are held, not drawn again.
func TestRetailPrototypesAreShared(t *testing.T) {
	first, err := retailPrototypeOf("ramen", 3, 1)
	if err != nil {
		t.Fatal(err)
	}
	second, err := retailPrototypeOf("ramen", 3, 1)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Error("a unit of the pool was drawn twice")
	}
	other, err := retailPrototypeOf("ramen", 4, 1)
	if err != nil {
		t.Fatal(err)
	}
	if other == first {
		t.Error("two variants were the one unit")
	}
}

// A block refuses a level it does not have, and a seed that was not
// sown.
func TestRetailBlockRefusesWhatItCannotMake(t *testing.T) {
	if _, err := (retailBlockRow{}).madeAt(Sow(1), "clumps"); err == nil {
		t.Error("a block was made at a level it does not have")
	}
	if _, err := (retailBlockRow{}).madeAt(seed{}, "quick"); !errors.Is(err,
		errGood) {
		t.Errorf("an unsown seed: %v", err)
	}
}

// Data that does not hold together makes no frontage: a look nobody
// made, a part outside its bays, a light with no luminance, one type of
// unit, a run of none, a palette with nothing in it, a module nobody
// made, a podium too shallow for a room.
func TestRetailDataRefusesWhatIsBroken(t *testing.T) {
	breaks := map[string]func(*retailBook){
		"a wall of no look": func(book *retailBook) {
			unit := book.Units["ramen"]
			unit.Wall = "nothing"
			book.Units["ramen"] = unit
		},
		"a wall of no palette": func(book *retailBook) {
			unit := book.Units["ramen"]
			unit.Wall = "@nothing"
			book.Units["ramen"] = unit
		},
		"a part past the bays": func(book *retailBook) {
			unit := book.Units["dive"]
			unit.Kit.Parts = append(slices.Clone(unit.Kit.Parts),
				retailPart{Material: "wood", Box: []float64{0, 0, 0, 9, 1, 1}})
			book.Units["dive"] = unit
		},
		"a part past the widest bays": func(book *retailBook) {
			unit := book.Units["ramen"]
			unit.Kit.Parts = append(slices.Clone(unit.Kit.Parts), retailPart{
				Material: "wood", Box: []float64{7, 0, 0, 7.3, 1, 1},
				Along: retailRight})
			book.Units["ramen"] = unit
		},
		"a light of no luminance": func(book *retailBook) {
			unit := book.Units["konbini"]
			unit.Kit.Lights = []retailLight{{At: []float64{1, 1, -1},
				Heat: 5000}}
			book.Units["konbini"] = unit
		},
		"one type of unit": func(book *retailBook) {
			book.Units = map[string]retailUnit{"ramen": book.Units["ramen"]}
		},
		"a run of none": func(book *retailBook) { book.Run.Units = 0 },
		"a prop nobody made": func(book *retailBook) {
			unit := book.Units["dive"]
			unit.Kit.Props = []retailPropUse{{Prop: "piano",
				X: []float64{0, 1}, Z: []float64{1, 2}, Chance: 1}}
			book.Units["dive"] = unit
		},
		"a palette with nothing in it": func(book *retailBook) {
			book.Palettes["awning"] = nil
		},
		"a palette of a look nobody made": func(book *retailBook) {
			book.Palettes["awning"] = []string{"nothing"}
		},
		"a module nobody made": func(book *retailBook) {
			unit := book.Units["ramen"]
			unit.Kit.Uses = []retailUse{{Module: "porch"}}
			book.Units["ramen"] = unit
		},
		"a way to move nobody has": func(book *retailBook) {
			unit := book.Units["ramen"]
			unit.Kit.Parts = append(slices.Clone(unit.Kit.Parts), retailPart{
				Material: "wood", Box: []float64{1, 0, 0, 2, 1, 1},
				Along: "up"})
			book.Units["ramen"] = unit
		},
		"an option with no variants": func(book *retailBook) {
			unit := book.Units["ramen"]
			unit.Options = append(slices.Clone(unit.Options),
				retailOption{Chance: 0.5})
			book.Units["ramen"] = unit
		},
		"a variant of no weight": func(book *retailBook) {
			unit := book.Units["ramen"]
			unit.Options = append(slices.Clone(unit.Options), retailOption{
				Chance: 0.5, Variants: []retailVariant{{}}})
			book.Units["ramen"] = unit
		},
		"a podium too shallow": func(book *retailBook) {
			book.Grounds.Podium = 1
		},
		"a podium with no buildings": func(book *retailBook) {
			book.Grounds.Buildings = nil
		},
		"a prism of two sides": func(book *retailBook) {
			unit := book.Units["dive"]
			unit.Kit.Parts = append(slices.Clone(unit.Kit.Parts), retailPart{
				Material: "wood", Prism: []float64{1, 1, -1, 0.1, 0.1},
				Sides: 2})
			book.Units["dive"] = unit
		},
	}
	for name, change := range breaks {
		if retailBroken(t, change) == nil {
			t.Errorf("data with %s was taken", name)
		}
	}
}

// Unknown words in the data are refused, not ignored.
func TestRetailDataRefusesAnUnknownField(t *testing.T) {
	raw := append([]byte("verandas: 3\n"), retailUnitsData...)
	if _, err := readRetailBook(raw); err == nil {
		t.Error("a field the data does not have was taken")
	}
}
