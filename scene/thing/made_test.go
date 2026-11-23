package thing

import (
	"fmt"
	"strings"
	"testing"

	"github.com/janearc/slayer/materials"
	"github.com/janearc/slayer/maths"
)

// The tables the made tests read, beside numbers.yaml.
const (
	madeFinishesFile = "testdata/made-finishes.csv"
	madeLooksFile    = "testdata/made-looks.csv"
	madePaintsFile   = "testdata/made-paints.csv"
)

// madeLopsidedBook is a book of one finish that wears two parts of colour,
// where it needs three.
const madeLopsidedBook = "finishes:\n- name: lopsided\n  colour: [1, 2]\n"

// worn is a finish from the book, or the test stops.
func worn(t *testing.T, name string) Finish {
	t.Helper()
	finish, err := FinishOf(name)
	if err != nil {
		t.Fatalf("finish %q: %v", name, err)
	}
	return finish
}

// madeFinish is the row of the finishes table a finish is named by.
func madeFinish(t *testing.T, name string) map[string]string {
	t.Helper()
	return readSheet(t, madeFinishesFile).row(t, madeFinishesFile, name)
}

// madeColour is the parts of colour a finishes row wears, none for a row
// with no colour.
func madeColour(t *testing.T, row map[string]string) []float64 {
	t.Helper()
	if row["colour-red"] == "" {
		return nil
	}
	return []float64{cell(t, row, "colour-red"), cell(t, row, "colour-green"),
		cell(t, row, "colour-blue")}
}

// madeMaterial is a material of the database, or the test stops.
func madeMaterial(t *testing.T, name string) materials.Material {
	t.Helper()
	found, err := materials.ByName(name)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return found
}

// madeIron is iron, which has a formula and a density and no optics.
func madeIron(t *testing.T) materials.Material {
	t.Helper()
	formula, err := materials.NewFormula(map[string]float64{"Fe": unity})
	if err != nil {
		t.Fatalf("iron's formula: %v", err)
	}
	iron, err := materials.NewMaterial("iron", formula,
		picked(t, "made-iron-density"))
	if err != nil {
		t.Fatalf("iron: %v", err)
	}
	return iron
}

// The book loads with every number picked for the eight finishes, so a
// hand edit to finishes.yaml that drifts from the rows of the table is
// caught here.
func TestTheBookOfFinishesCarriesEveryNumber(t *testing.T) {
	for _, each := range readSheet(t, madeFinishesFile) {
		name := each["name"]
		row, known := finishes[name]
		if !known {
			t.Errorf("no finish %q in the book", name)
			continue
		}
		gloss, satin := cell(t, each, "gloss"), cell(t, each, "satin")
		if row.Gloss != gloss || row.Satin != satin {
			t.Errorf("%s glosses %v at satin %v, want %v at %v", name,
				row.Gloss, row.Satin, gloss, satin)
		}
		colour := madeColour(t, each)
		if len(row.Colour) != len(colour) {
			t.Errorf("%s wears %v parts of colour, want %v", name, row.Colour,
				colour)
		}
		for place, part := range colour {
			if len(row.Colour) == len(colour) && row.Colour[place] != part {
				t.Errorf("%s colour %d is %v, want %v", name, place,
					row.Colour[place], part)
			}
		}
		if row.Paint != (each["paint"] == "true") {
			t.Errorf("%s paint is %v", name, row.Paint)
		}
		index, thickness := cell(t, each, "film-index"),
			cell(t, each, "film-thickness")
		if row.FilmIndex != index || row.FilmThickness != thickness {
			t.Errorf("%s films at %v by %v, want %v by %v", name,
				row.FilmIndex, row.FilmThickness, index, thickness)
		}
		if row.Kind != "picked" {
			t.Errorf("%s is %q, want picked", name, row.Kind)
		}
	}
}

// Titanium under weather is titanium's own readings with a weary
// surface: the metal is ByName's, band for band, and the finish's satin
// and colour sit on top.
func TestTitaniumWeatheredIsTitaniumWithAWearySurface(t *testing.T) {
	titanium := madeMaterial(t, "titanium")
	look, err := Made(titanium, worn(t, "weathered"))
	if err != nil {
		t.Fatalf("titanium weathered: %v", err)
	}
	sight := titanium.Optics()
	for place, band := range [...]materials.Band{materials.Red, materials.Green,
		materials.Blue} {
		read := sight.At(band)
		if look.Metal.N[place] != read.N() || look.Metal.K[place] != read.K() {
			t.Errorf("colour %d is n %v k %v, want n %v k %v", place,
				look.Metal.N[place], look.Metal.K[place], read.N(), read.K())
		}
	}
	weathered := madeFinish(t, "weathered")
	if satin := cell(t, weathered, "satin"); look.Satin != satin {
		t.Errorf("weathered satin is %v, want %v", look.Satin, satin)
	}
	dust := maths.RGB(cell(t, weathered, "colour-red"),
		cell(t, weathered, "colour-green"), cell(t, weathered, "colour-blue"))
	if look.Colour != dust {
		t.Errorf("weathered colour is %v, want a dusty warm grey", look.Colour)
	}
	if look.Name != "titanium" {
		t.Errorf("the look is named %q, want titanium", look.Name)
	}
}

// Gold under polish mirrors as Gold itself does at the same gloss: a
// made look and the standing constructor agree, colour for colour.
func TestGoldPolishedMirrorsAsGoldItself(t *testing.T) {
	made, err := Made(madeMaterial(t, "gold"), worn(t, "polished"))
	if err != nil {
		t.Fatalf("gold polished: %v", err)
	}
	want := &Material{Name: "gold", Gloss: unity, Metal: Gold()}
	for _, cos := range metalCosines(t, madeLooksFile) {
		got, expected := made.Reflects(cos), want.Reflects(cos)
		if !metalSame(got, expected, picked(t, "made-tolerance")) {
			t.Errorf("at cosine %v gold made reflects %v, want %v",
				cos, got, expected)
		}
	}
}

// Flat is the finish that finishes nothing: over a metal it carries
// the metal's own readings, but with no gloss there is nothing
// mirrored, at any angle, so only the material's own colour is left.
func TestGoldFlatMirrorsNothingAtAnyAngle(t *testing.T) {
	look, err := Made(madeMaterial(t, "gold"), worn(t, "flat"))
	if err != nil {
		t.Fatalf("gold flat: %v", err)
	}
	if look.Gloss != nought || look.Satin != nought {
		t.Errorf("flat glosses %v at satin %v, want nought at nought",
			look.Gloss, look.Satin)
	}
	if look.Colour != maths.RGB(nought, nought, nought) {
		t.Errorf("flat carries a colour of its own: %v", look.Colour)
	}
	if look.Film != nil {
		t.Error("flat carries a film")
	}
	for _, cos := range metalCosines(t, madeLooksFile) {
		if got := look.Reflects(cos); got != maths.RGB(nought, nought, nought) {
			t.Errorf("at cosine %v flat gold reflects %v, want nothing",
				cos, got)
		}
	}
}

// Paint needs no optics: over a material with none it is exactly a
// plain coloured look, flat as the pigment left it.
func TestPaintedOverNoOpticsIsAPlainColouredLook(t *testing.T) {
	row := readSheet(t, madePaintsFile).row(t, madePaintsFile, "coat")
	coat := maths.RGB(cell(t, row, "red"), cell(t, row, "green"),
		cell(t, row, "blue"))
	look, err := Made(madeIron(t), Painted(coat))
	if err != nil {
		t.Fatalf("iron painted: %v", err)
	}
	if look.Name != "iron" || look.Colour != coat {
		t.Errorf("painted iron is %q of %v", look.Name, look.Colour)
	}
	if look.Gloss != nought || look.Satin != nought {
		t.Errorf("painted iron glosses %v at satin %v, want nought at nought",
			look.Gloss, look.Satin)
	}
	if look.Metal != nil || look.Film != nil || look.Through != nought ||
		look.Index != nought || look.Texture != nil ||
		look.Glow != maths.RGB(nought, nought, nought) {
		t.Errorf("painted iron carries more than a colour: %+v", look)
	}
}

// The oxide over titanium is a clear film of the book's index and
// thickness.
func TestOxidisedTitaniumWearsTheBooksFilm(t *testing.T) {
	look, err := Made(madeMaterial(t, "titanium"), worn(t, "oxidised"))
	if err != nil {
		t.Fatalf("titanium oxidised: %v", err)
	}
	row := finishes["oxidised"]
	if look.Film == nil {
		t.Fatal("oxidised titanium wears no film")
	}
	if look.Film.Index != row.FilmIndex || look.Film.Thickness != row.FilmThickness {
		t.Errorf("the film is %v by %v m, want %v by %v", look.Film.Index,
			look.Film.Thickness, row.FilmIndex, row.FilmThickness)
	}
	if look.Film.Absorbs != ([metalColours]float64{}) {
		t.Errorf("the oxide absorbs %v, want a clear film", look.Film.Absorbs)
	}
}

// A finish the book does not carry, and a material the database does
// not carry, are refused by name, from both sides of the bridge.
func TestUnknownNamesAreRefusedByName(t *testing.T) {
	if _, err := FinishOf("burnished"); err == nil ||
		!strings.Contains(err.Error(), `no finish "burnished"`) {
		t.Errorf("burnished: %v", err)
	}
	if _, err := FinishOf("painted"); err == nil ||
		!strings.Contains(err.Error(), "paint needs a colour") {
		t.Errorf("paint by name: %v", err)
	}
	titanium := madeMaterial(t, "titanium")
	if _, err := Made(titanium, Finish{name: "burnished"}); err == nil ||
		!strings.Contains(err.Error(), `no finish "burnished"`) {
		t.Errorf("burnished through Made: %v", err)
	}
	if _, err := materials.ByName("osmium"); err == nil ||
		!strings.Contains(err.Error(), "osmium") {
		t.Errorf("osmium: %v", err)
	}
}

// stopped is what a call stops the program with, as words, and nothing
// if it ran through.
func stopped(run func()) (said string) {
	defer madeSaid(&said)
	run()
	return ""
}

// madeSaid is what the program was stopped with, as words, put in said;
// it is called as a call is left, and does nothing if none stopped it.
func madeSaid(said *string) {
	if stop := recover(); stop != nil {
		*said = fmt.Sprint(stop)
	}
}

// madeFinishesNotYaml reads a book of finishes that is not yaml.
func madeFinishesNotYaml() { readFinishes([]byte("finishes: [")) }

// madeFinishWithTwoPartsOfColour reads a book whose one finish wears a
// colour of two parts.
func madeFinishWithTwoPartsOfColour() {
	readFinishes([]byte(madeLopsidedBook))
}

// madeMetalMaterialsLacks reads a book of metals that names one materials
// does not carry.
func madeMetalMaterialsLacks() { readMetals("gold", "osmium") }

// madeLookNotHeld asks for a look of a metal the book does not hold.
func madeLookNotHeld() { look("osmium") }

// madeBooksAsTheyAre reads the books as they are.
func madeBooksAsTheyAre() {
	readFinishes(finishesRaw)
	readMetals("gold", "titanium")
	look("gold")
}

// A material never made, the zero one, is refused whatever it would
// wear: it has no name for the look to carry.
func TestAMaterialNeverMadeIsRefused(t *testing.T) {
	var never materials.Material
	for _, finish := range []Finish{worn(t, "polished"),
		Painted(maths.RGB(unity, nought, nought))} {
		if _, err := Made(never, finish); err == nil ||
			!strings.Contains(err.Error(), "never made") {
			t.Errorf("a material never made, %q: %v", finish.name, err)
		}
	}
}

// A book that cannot be read stops the program before any look is made:
// a book of finishes that is not yaml, or whose finish wears a colour
// of two parts, by its name; a book of metals that names one materials
// does not carry, by its name; and a look asked of a metal the book
// does not hold. The anti-test is the books as they are, which read.
func TestABrokenBookStopsTheProgram(t *testing.T) {
	stops := map[string]struct {
		run  func()
		says string
	}{
		"finishes that are not yaml": {madeFinishesNotYaml, "finishes.yaml"},
		"a colour of two parts": {madeFinishWithTwoPartsOfColour,
			"lopsided wears 2 parts of colour"},
		"a metal materials does not carry": {madeMetalMaterialsLacks,
			"osmium"},
		"a look the book does not hold": {madeLookNotHeld, "osmium has no"},
	}
	for name, each := range stops {
		if said := stopped(each.run); !strings.Contains(said, each.says) {
			t.Errorf("%s: stopped with %q, want it to say %q", name, said,
				each.says)
		}
	}
	if said := stopped(madeBooksAsTheyAre); said != "" {
		t.Errorf("the books as they are stopped with %q", said)
	}
}

// A metal finish refuses a material with no optics: there is nothing to
// polish. Paint is the exception, and its test above pins that.
func TestAMetalFinishRefusesAMaterialWithNoOptics(t *testing.T) {
	_, err := Made(madeIron(t), worn(t, "polished"))
	if err == nil || !strings.Contains(err.Error(), "iron") ||
		!strings.Contains(err.Error(), `"polished"`) {
		t.Errorf("iron polished: %v", err)
	}
}
