package thing

import (
	"fmt"
	"strings"
	"testing"

	"github.com/janearc/slayer/materials"
	"github.com/janearc/slayer/maths"
)

// worn is a finish from the book, or the test stops.
func worn(t *testing.T, name string) Finish {
	t.Helper()
	finish, err := FinishOf(name)
	if err != nil {
		t.Fatalf("finish %q: %v", name, err)
	}
	return finish
}

// The book loads with every number picked for the eight finishes, so a
// hand edit to finishes.yaml that drifts from these is caught here.
func TestTheBookOfFinishesCarriesEveryNumber(t *testing.T) {
	for _, each := range []struct {
		name                     string
		gloss, satin             float64
		colour                   []float64
		paint                    bool
		filmIndex, filmThickness float64
	}{
		{"polished", 1, 0, nil, false, 0, 0},
		{"satin", 1, 0.25, nil, false, 0, 0},
		{"brushed", 0.85, 0.5, nil, false, 0, 0},
		{"weathered", 0.65, 0.45, []float64{0.45, 0.42, 0.38}, false, 0, 0},
		{"frosted", 0.3, 0.9, nil, false, 0, 0},
		{"flat", 0, 0, nil, false, 0, 0},
		{"painted", 0, 0, nil, true, 0, 0},
		{"oxidised", 0.85, 0.15, nil, false, 2.65, 5e-9},
	} {
		row, known := finishes[each.name]
		if !known {
			t.Errorf("no finish %q in the book", each.name)
			continue
		}
		if row.Gloss != each.gloss || row.Satin != each.satin {
			t.Errorf("%s glosses %v at satin %v, want %v at %v",
				each.name, row.Gloss, row.Satin, each.gloss, each.satin)
		}
		if len(row.Colour) != len(each.colour) {
			t.Errorf("%s wears %v parts of colour, want %v",
				each.name, row.Colour, each.colour)
		} else {
			for at, part := range each.colour {
				if row.Colour[at] != part {
					t.Errorf("%s colour %d is %v, want %v",
						each.name, at, row.Colour[at], part)
				}
			}
		}
		if row.Paint != each.paint {
			t.Errorf("%s paint is %v", each.name, row.Paint)
		}
		if row.FilmIndex != each.filmIndex || row.FilmThickness != each.filmThickness {
			t.Errorf("%s films at %v by %v, want %v by %v", each.name,
				row.FilmIndex, row.FilmThickness, each.filmIndex, each.filmThickness)
		}
		if row.Kind != "picked" {
			t.Errorf("%s is %q, want picked", each.name, row.Kind)
		}
	}
}

// Titanium under weather is titanium's own readings with a weary
// surface: the metal is ByName's, band for band, and the finish's satin
// and colour sit on top.
func TestTitaniumWeatheredIsTitaniumWithAWearySurface(t *testing.T) {
	ti, err := materials.ByName("titanium")
	if err != nil {
		t.Fatalf("titanium: %v", err)
	}
	look, err := Made(ti, worn(t, "weathered"))
	if err != nil {
		t.Fatalf("titanium weathered: %v", err)
	}
	sight := ti.Optics()
	for at, band := range [...]materials.Band{materials.Red, materials.Green,
		materials.Blue} {
		read := sight.At(band)
		if look.Metal.N[at] != read.N() || look.Metal.K[at] != read.K() {
			t.Errorf("colour %d is n %v k %v, want n %v k %v", at,
				look.Metal.N[at], look.Metal.K[at], read.N(), read.K())
		}
	}
	if look.Satin != 0.45 {
		t.Errorf("weathered satin is %v, want 0.45", look.Satin)
	}
	if look.Colour != maths.RGB(0.45, 0.42, 0.38) {
		t.Errorf("weathered colour is %v, want a dusty warm grey", look.Colour)
	}
	if look.Name != "titanium" {
		t.Errorf("the look is named %q, want titanium", look.Name)
	}
}

// Gold under polish mirrors as Gold itself does at the same gloss: a
// made look and the standing constructor agree, colour for colour.
func TestGoldPolishedMirrorsAsGoldItself(t *testing.T) {
	au, err := materials.ByName("gold")
	if err != nil {
		t.Fatalf("gold: %v", err)
	}
	made, err := Made(au, worn(t, "polished"))
	if err != nil {
		t.Fatalf("gold polished: %v", err)
	}
	want := &Material{Name: "gold", Gloss: 1, Metal: Gold()}
	for _, cos := range []float64{1, 0.9, 0.7, 0.4, 0.1, 0.02} {
		got, expected := made.Reflects(cos), want.Reflects(cos)
		if !nearly(got.R, expected.R, 1e-12) || !nearly(got.G, expected.G, 1e-12) ||
			!nearly(got.B, expected.B, 1e-12) {
			t.Errorf("at cosine %v gold made reflects %v, want %v",
				cos, got, expected)
		}
	}
}

// Flat is the finish that finishes nothing: over a metal it carries
// the metal's own readings, but with no gloss there is nothing
// mirrored, at any angle, so only the material's own colour is left.
func TestGoldFlatMirrorsNothingAtAnyAngle(t *testing.T) {
	au, err := materials.ByName("gold")
	if err != nil {
		t.Fatalf("gold: %v", err)
	}
	look, err := Made(au, worn(t, "flat"))
	if err != nil {
		t.Fatalf("gold flat: %v", err)
	}
	if look.Gloss != 0 || look.Satin != 0 {
		t.Errorf("flat glosses %v at satin %v, want nought at nought",
			look.Gloss, look.Satin)
	}
	if look.Colour != (maths.Light{}) {
		t.Errorf("flat carries a colour of its own: %v", look.Colour)
	}
	if look.Film != nil {
		t.Error("flat carries a film")
	}
	for _, cos := range []float64{1, 0.9, 0.7, 0.4, 0.1, 0.02} {
		if got := look.Reflects(cos); got != (maths.Light{}) {
			t.Errorf("at cosine %v flat gold reflects %v, want nothing",
				cos, got)
		}
	}
}

// Paint needs no optics: over a material with none it is exactly a
// plain coloured look, flat as the pigment left it.
func TestPaintedOverNoOpticsIsAPlainColouredLook(t *testing.T) {
	fe, err := materials.NewFormula(map[string]float64{"Fe": 1})
	if err != nil {
		t.Fatalf("iron's formula: %v", err)
	}
	iron, err := materials.NewMaterial("iron", fe, 7.87)
	if err != nil {
		t.Fatalf("iron: %v", err)
	}
	coat := maths.RGB(0.63, 0.12, 0.1)
	look, err := Made(iron, Painted(coat))
	if err != nil {
		t.Fatalf("iron painted: %v", err)
	}
	if look.Name != "iron" || look.Colour != coat {
		t.Errorf("painted iron is %q of %v", look.Name, look.Colour)
	}
	if look.Gloss != 0 || look.Satin != 0 {
		t.Errorf("painted iron glosses %v at satin %v, want nought at nought",
			look.Gloss, look.Satin)
	}
	if look.Metal != nil || look.Film != nil || look.Through != 0 ||
		look.Index != 0 || look.Texture != nil || look.Glow != (maths.Light{}) {
		t.Errorf("painted iron carries more than a colour: %+v", look)
	}
}

// The oxide over titanium is a clear film of the book's index and
// thickness.
func TestOxidisedTitaniumWearsTheBooksFilm(t *testing.T) {
	ti, err := materials.ByName("titanium")
	if err != nil {
		t.Fatalf("titanium: %v", err)
	}
	look, err := Made(ti, worn(t, "oxidised"))
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
	if look.Film.Absorbs != ([3]float64{}) {
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
	ti, err := materials.ByName("titanium")
	if err != nil {
		t.Fatalf("titanium: %v", err)
	}
	if _, err := Made(ti, Finish{name: "burnished"}); err == nil ||
		!strings.Contains(err.Error(), `no finish "burnished"`) {
		t.Errorf("burnished through Made: %v", err)
	}
	// aluminium was the database's refusal until 2026-11-11, when it was
	// decided its blue n be corrected to the published row and the
	// database carried it; osmium carries the refusal now, a name still
	// not carried
	if _, err := materials.ByName("osmium"); err == nil ||
		!strings.Contains(err.Error(), "osmium") {
		t.Errorf("osmium: %v", err)
	}
}

// stopped is what a call stops the program with, as words, and nothing
// if it ran through.
func stopped(run func()) (said string) {
	defer func() {
		if r := recover(); r != nil {
			said = fmt.Sprint(r)
		}
	}()
	run()
	return ""
}

// A material never made, the zero one, is refused whatever it would
// wear: it has no name for the look to carry.
func TestAMaterialNeverMadeIsRefused(t *testing.T) {
	var never materials.Material
	for _, finish := range []Finish{worn(t, "polished"),
		Painted(maths.RGB(1, 0, 0))} {
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
		"finishes that are not yaml": {func() {
			readFinishes([]byte("finishes: ["))
		}, "finishes.yaml"},
		"a colour of two parts": {func() {
			readFinishes([]byte("finishes:\n- name: lopsided\n  colour: [1, 2]\n"))
		}, "lopsided wears 2 parts of colour"},
		"a metal materials does not carry": {func() {
			readMetals("gold", "osmium")
		}, "osmium"},
		"a look the book does not hold": {func() {
			look("osmium")
		}, "osmium has no"},
	}
	for name, each := range stops {
		if said := stopped(each.run); !strings.Contains(said, each.says) {
			t.Errorf("%s: stopped with %q, want it to say %q", name, said,
				each.says)
		}
	}
	if said := stopped(func() {
		readFinishes(finishesRaw)
		readMetals("gold", "titanium")
		look("gold")
	}); said != "" {
		t.Errorf("the books as they are stopped with %q", said)
	}
}

// A metal finish refuses a material with no optics: there is nothing to
// polish. Paint is the exception, and its test above pins that.
func TestAMetalFinishRefusesAMaterialWithNoOptics(t *testing.T) {
	fe, err := materials.NewFormula(map[string]float64{"Fe": 1})
	if err != nil {
		t.Fatalf("iron's formula: %v", err)
	}
	iron, err := materials.NewMaterial("iron", fe, 7.87)
	if err != nil {
		t.Fatalf("iron: %v", err)
	}
	_, err = Made(iron, worn(t, "polished"))
	if err == nil || !strings.Contains(err.Error(), "iron") ||
		!strings.Contains(err.Error(), `"polished"`) {
		t.Errorf("iron polished: %v", err)
	}
}
