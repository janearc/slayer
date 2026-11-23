package thing

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The tables the metal tests read, beside numbers.yaml. A row of each is
// asked for by its name column.
const (
	metalReadingsFile    = "testdata/metal-readings.csv"
	metalFaceOnFile      = "testdata/metal-face-on.csv"
	metalAbsorbsFile     = "testdata/metal-absorbs.csv"
	metalLooksFile       = "testdata/metal-looks.csv"
	metalMirrorLooksFile = "testdata/metal-mirror-looks.csv"
)

// metalColours is how many colours a metal's readings are in.
const metalColours = 3 // defined: red, green and blue

// nearly is whether got is within tolerance of want; a NaN never is.
func nearly(got, want, tolerance float64) bool {
	return math.Abs(got-want) <= tolerance
}

// metalRow is the row of a table beside these tests with a name.
func metalRow(t *testing.T, file, name string) map[string]string {
	t.Helper()
	return readSheet(t, file).row(t, file, name)
}

// metalThree is a row's number for red, green and blue, from the columns
// the prefix names: n-red, n-green and n-blue for the prefix n.
func metalThree(t *testing.T, row map[string]string,
	prefix string) [metalColours]float64 {
	t.Helper()
	return [metalColours]float64{cell(t, row, prefix+"-red"),
		cell(t, row, prefix+"-green"), cell(t, row, prefix+"-blue")}
}

// metalRead is the metal a row of the readings table names.
func metalRead(t *testing.T, name string) Metal {
	t.Helper()
	row := metalRow(t, metalReadingsFile, name)
	return Metal{N: metalThree(t, row, "n"), K: metalThree(t, row, "k")}
}

// metalFaceOn is the share of red, green and blue a row of the face-on
// table names.
func metalFaceOn(t *testing.T, name string) maths.Light {
	t.Helper()
	row := metalRow(t, metalFaceOnFile, name)
	return maths.RGB(cell(t, row, "red"), cell(t, row, "green"),
		cell(t, row, "blue"))
}

// metalAbsorbs is how much per metre of red, green and blue a row of the
// absorption table swallows.
func metalAbsorbs(t *testing.T, name string) [metalColours]float64 {
	t.Helper()
	return metalThree(t, metalRow(t, metalAbsorbsFile, name), "absorbs")
}

// metalCosines is the cosines of the looks a table holds, in the order
// written.
func metalCosines(t *testing.T, file string) []float64 {
	t.Helper()
	var cosines []float64
	for _, row := range readSheet(t, file) {
		cosines = append(cosines, cell(t, row, "cosine"))
	}
	return cosines
}

// metalSweep is the cosines from nought to one in as many even steps as
// the picked number of the name says.
func metalSweep(t *testing.T, name string) []float64 {
	t.Helper()
	steps := pickedCount(t, name)
	cosines := make([]float64, steps+unity)
	for step := range cosines {
		cosines[step] = float64(step) / float64(steps)
	}
	return cosines
}

// metalSame is whether two lights are within tolerance in every colour.
func metalSame(got, want maths.Light, tolerance float64) bool {
	return nearly(got.R, want.R, tolerance) &&
		nearly(got.G, want.G, tolerance) && nearly(got.B, want.B, tolerance)
}

// metalFinite is whether every colour of a light is a number.
func metalFinite(light maths.Light) bool {
	for _, share := range []float64{light.R, light.G, light.B} {
		if math.IsNaN(share) || math.IsInf(share, nought) {
			return false
		}
	}
	return true
}

// metalUnder is a glossy surface of a metal under a film.
func metalUnder(metal *Metal, film Film) Material {
	return Material{Gloss: unity, Metal: metal, Film: &film}
}

// Face on, a conductor reflects ((n-1)^2 + k^2) / ((n+1)^2 + k^2): for
// Johnson and Christy's gold, worked with awk, 0.930978291 of red,
// 0.786915760 of green and 0.401478098 of blue. That is gold's colour.
// The anti-test: at grazing it is no colour at all, for every surface
// mirrors all of every colour there.
func TestGoldReflectsMostOfRedAndLeastOfBlueFaceOn(t *testing.T) {
	gold := metalRead(t, "gold-rounded")
	made := Material{Gloss: unity, Metal: &gold}
	got, want := made.Reflects(unity), metalFaceOn(t, "gold-rounded")
	if !metalSame(got, want, picked(t, "metal-awk-tolerance")) {
		t.Errorf("gold reflects %v face on, want %v", got, want)
	}
	grazing := made.Reflects(nought)
	if !metalSame(grazing, maths.Grey(unity),
		picked(t, "metal-grazing-tolerance")) {
		t.Errorf("gold reflects %v at grazing, want all of every colour",
			grazing)
	}
}

// Gold, as the package makes it from materials' readings, reflects what
// its own doc says face on, to the doc's two places, and is a colour: red
// over green over blue. The anti-test: silver, made the same way, is
// not, its red and blue within a hundredth.
func TestGoldFromMaterialsIsAsItsDocSays(t *testing.T) {
	made := Material{Gloss: unity, Metal: Gold()}
	got := made.Reflects(unity)
	if !metalSame(got, metalFaceOn(t, "gold-doc"), picked(t, "metal-doc-tolerance")) {
		t.Errorf("Gold reflects %v face on, its doc says %v", got,
			metalFaceOn(t, "gold-doc"))
	}
	if !(got.R > got.G && got.G > got.B) {
		t.Errorf("Gold reflects %v face on: not red over green over blue", got)
	}
	silver := (&Material{Gloss: unity, Metal: Silver()}).Reflects(unity)
	if !(silver.R-silver.B < picked(t, "metal-white-spread")) {
		t.Errorf("Silver reflects %v face on: a colour", silver)
	}
}

// At grazing every surface reflects all of every colour, and from face
// on to grazing a metal's share never leaves zero to one.
func TestAConductorReflectsAllAtGrazingAndNeverMore(t *testing.T) {
	gold := metalRead(t, "gold-rounded")
	for colour := range gold.N {
		got := Conductor(nought, gold.N[colour], gold.K[colour])
		if !nearly(got, unity, picked(t, "metal-grazing-tolerance")) {
			t.Errorf("gold's colour %d reflects %v at grazing", colour, got)
		}
		for _, cos := range metalSweep(t, "metal-sweep-steps") {
			got := Conductor(cos, gold.N[colour], gold.K[colour])
			if !(got >= nought && got <= unity) {
				t.Fatalf("gold's colour %d reflects %v at cosine %v", colour,
					got, cos)
			}
		}
	}
}

// A conductor with no k is a clear dielectric: the two forms of
// Fresnel's equations agree for glass at every angle. The anti-test: a
// conductor that does absorb is no dielectric, and face on it mirrors
// more than glass of its n does.
func TestAConductorWithNoAbsorptionIsADielectric(t *testing.T) {
	glass := picked(t, "metal-glass-index")
	for _, cos := range metalSweep(t, "metal-dielectric-steps") {
		got, want := Conductor(cos, glass, nought), dielectric(cos, glass)
		if !nearly(got, want, picked(t, "metal-dielectric-tolerance")) {
			t.Errorf("at cosine %v the conductor gives %v, the dielectric %v",
				cos, got, want)
		}
	}
	absorbing := Conductor(unity, glass, picked(t, "metal-absorbing-k"))
	plain := dielectric(unity, glass)
	if !(absorbing > plain+picked(t, "metal-absorbing-margin")) {
		t.Errorf("a conductor with a k of 2 mirrors %v face on, glass %v",
			absorbing, plain)
	}
	faceOn := picked(t, "metal-glass-face-on")
	if !nearly(plain, faceOn, picked(t, "metal-glass-face-on-tolerance")) {
		t.Errorf("glass reflects %v face on, want %v", plain, faceOn)
	}
	water := picked(t, "metal-water-index")
	got := dielectric(picked(t, "metal-critical-cosine"), unity/water)
	if got != unity {
		t.Errorf("past the critical angle water reflects %v, want all", got)
	}
}

// A film that is not there leaves the metal bare: index one, no
// thickness. The anti-test is a film that lets nothing through, which
// shows none of the metal under it: only its own front.
func TestAFilmBetweenNothingAndEverything(t *testing.T) {
	gold := metalRead(t, "gold-rounded")
	bare := Material{Gloss: unity, Metal: &gold}
	thin := metalUnder(&gold, Film{Index: unity})
	index := picked(t, "metal-dark-index")
	opaque := picked(t, "metal-opaque-absorption")
	dark := metalUnder(&gold, Film{Index: index,
		Thickness: picked(t, "metal-dark-thickness"),
		Absorbs:   [metalColours]float64{opaque, opaque, opaque}})
	tolerance := picked(t, "metal-film-tolerance")
	for _, cos := range metalCosines(t, metalLooksFile) {
		bareLight, thinLight := bare.Reflects(cos), thin.Reflects(cos)
		if !metalSame(thinLight, bareLight, tolerance) {
			t.Errorf("at cosine %v an empty film gives %v, the bare metal %v",
				cos, thinLight, bareLight)
		}
		front, shown := dielectric(cos, index), dark.Reflects(cos)
		if !nearly(shown.R, front, tolerance) ||
			!nearly(shown.B, front, tolerance) {
			t.Errorf("at cosine %v an opaque film gives %v, its front %v",
				cos, shown, front)
		}
	}
}

// Face on, Johnson and Christy's titanium, worked with awk the same
// way: 0.608895790 of red, 0.581832774 of green, 0.550221981 of blue.
// The anti-test: it is a grey and not a colour, its red and its blue
// within six hundredths, where gold's are half apart.
func TestTitaniumIsAWarmGreyFaceOn(t *testing.T) {
	made := Material{Gloss: unity, Metal: Titanium()}
	got, want := made.Reflects(unity), metalFaceOn(t, "titanium")
	if !metalSame(got, want, picked(t, "metal-awk-tolerance")) {
		t.Errorf("titanium reflects %v face on, want %v", got, want)
	}
	yellow := (&Material{Gloss: unity, Metal: Gold()}).Reflects(unity)
	if !(got.R-got.B < picked(t, "metal-grey-spread")) ||
		!(yellow.R-yellow.B > picked(t, "metal-colour-spread")) {
		t.Errorf("titanium's red and blue are %v apart, gold's %v",
			got.R-got.B, yellow.R-yellow.B)
	}
}

// Face on, the other two metals, worked with awk the same way from
// the readings materials holds: silver 0.986930029 of red, 0.982836296
// of green and 0.978572181 of blue; aluminium 0.913060575, 0.914118518
// and 0.919091051. Silver and aluminium are white mirrors. The
// anti-test: neither is a colour, its three shares within a hundredth of
// one another.
func TestSilverAndAluminiumFaceOn(t *testing.T) {
	metals := map[string]*Metal{"silver": Silver(),
		"aluminium": Aluminium()}
	for name, each := range metals {
		got := (&Material{Gloss: unity, Metal: each}).Reflects(unity)
		want := metalFaceOn(t, name)
		if !metalSame(got, want, picked(t, "metal-awk-tolerance")) {
			t.Errorf("%s reflects %v face on, want %v", name, got, want)
		}
		most := math.Max(got.R, math.Max(got.G, got.B))
		least := math.Min(got.R, math.Min(got.G, got.B))
		if !(most-least < picked(t, "metal-white-spread")) {
			t.Errorf("%s's shares are %v apart: a colour", name, most-least)
		}
	}
}

// Kapton is amber: a film of it 40 microns thick passes, by e to the
// minus its absorption times its thickness, 0.799315 of red, 0.301194
// of green and 0.001114 of blue, worked with awk; it is as thick as it
// is asked to be, at an index of 1.7. Over aluminium it reflects most of
// red and least of blue: a blanket's gold. The anti-test: Kapton of no
// thickness swallows nothing, and the aluminium under it is no gold.
func TestKaptonPassesRedAndSwallowsBlue(t *testing.T) {
	thick := picked(t, "metal-kapton-thickness")
	film := Kapton(thick)
	if film.Index != picked(t, "metal-kapton-index") || film.Thickness != thick {
		t.Errorf("Kapton is %+v", *film)
	}
	passes := metalFaceOn(t, "kapton-passes")
	want := [metalColours]float64{passes.R, passes.G, passes.B}
	for colour, absorbs := range film.Absorbs {
		got := math.Exp(-absorbs * thick)
		if !nearly(got, want[colour], picked(t, "metal-kapton-tolerance")) {
			t.Errorf("40 microns of Kapton passes %v of colour %d, want %v",
				got, colour, want[colour])
		}
	}
	blanketThick := picked(t, "metal-blanket-thickness")
	blanket := metalUnder(Aluminium(), *Kapton(blanketThick))
	if got := blanket.Reflects(unity); !(got.R > got.G && got.G > got.B) {
		t.Errorf("Kapton over aluminium reflects %v face on: not gold", got)
	}
	none := metalUnder(Aluminium(), *Kapton(nought))
	if got := none.Reflects(unity); got.R > got.G && got.G > got.B {
		t.Errorf("Kapton of no thickness over aluminium reflects %v: gold", got)
	}
}

// Rutile is as thick as it is asked to be, at Devore's index. The
// anti-test is in the same line: it is clear, and absorbs nothing in
// any colour, where Kapton's film does.
func TestRutileIsAClearFilmAtDevoresIndex(t *testing.T) {
	thick := picked(t, "metal-rutile-thickness")
	oxide := Rutile(thick)
	if oxide.Index != picked(t, "metal-rutile-index") ||
		oxide.Thickness != thick || oxide.Absorbs != [metalColours]float64{} {
		t.Errorf("rutile is %+v", *oxide)
	}
}

// Amber over aluminium: a film that keeps red and swallows blue turns
// a silver metal gold, as Kapton turns a blanket's aluminium. The
// anti-test is the same metal bare, which is not gold.
func TestAnAmberFilmGildsSilverMetal(t *testing.T) {
	aluminium := metalRead(t, "aluminium-rounded")
	bare := Material{Gloss: unity, Metal: &aluminium}
	if got := bare.Reflects(unity); got.R > got.G && got.G > got.B {
		t.Errorf("bare aluminium reflects %v face on: gold", got)
	}
	blanket := metalUnder(&aluminium, Film{
		Index:     picked(t, "metal-kapton-index"),
		Thickness: picked(t, "metal-blanket-thickness"),
		Absorbs:   metalAbsorbs(t, "amber")})
	got := blanket.Reflects(unity)
	if !(got.R > got.G && got.G > got.B) {
		t.Errorf("Kapton over aluminium reflects %v face on: not gold", got)
	}
}

// Anything but a metal reflects as it always has: the same share of all
// three colours, to the last bit, so a picture of one does not move. The
// anti-test: a metal with no gloss on it mirrors nothing, whatever its
// colour.
func TestANonMetalReflectsAsMirrorsDoes(t *testing.T) {
	glass := Material{Gloss: unity, Index: picked(t, "metal-glass-index")}
	for _, cos := range metalCosines(t, metalMirrorLooksFile) {
		want := glass.Mirrors(cos)
		if got := glass.Reflects(cos); got != maths.Grey(want) {
			t.Errorf("glass reflects %v at cosine %v, Mirrors %v", got, cos,
				want)
		}
	}
	gold := metalRead(t, "gold-rounded")
	plain := Material{Colour: maths.Grey(picked(t, "metal-plain-grey")),
		Metal: &gold}
	if got := plain.Reflects(unity); got != maths.RGB(nought, nought, nought) {
		t.Errorf("a metal with no gloss reflects %v", got)
	}
}

// A metal with no index at all, n and k nought, reflects all of every
// colour at every look, and never a value that is not a number: face on,
// Conductor's equations divide nothing by nothing there. The anti-test is
// gold, which does not reflect all of its blue face on.
func TestAMetalWithNoIndexReflectsAll(t *testing.T) {
	bare := Material{Gloss: unity, Metal: &Metal{}}
	for _, cos := range append(metalCosines(t, metalLooksFile), nought) {
		if got := bare.Reflects(cos); got != maths.Grey(unity) {
			t.Errorf("a metal with no index reflects %v at cosine %v", got, cos)
		}
	}
	if got := Conductor(unity, nought, nought); got != unity {
		t.Errorf("Conductor of no index face on is %v, want all", got)
	}
	gold := Material{Gloss: unity, Metal: Gold()}
	if got := gold.Reflects(unity); !(got.B < unity) {
		t.Errorf("gold reflects %v of blue face on, as a bare metal does", got.B)
	}
}

// A film of no index, or of one below nought, is glass: it reflects what
// a film of glass's index does, bit for bit, and always a number. The
// anti-test: a film of another index does not reflect as glass does.
func TestAFilmOfNoIndexIsGlass(t *testing.T) {
	gold := metalRead(t, "gold-rounded")
	thick := picked(t, "metal-dark-thickness")
	glass := metalUnder(&gold, Film{Thickness: thick,
		Index: picked(t, "metal-glass-index")})
	for _, index := range []float64{nought, picked(t, "metal-negative-index")} {
		made := metalUnder(&gold, Film{Index: index, Thickness: thick})
		for _, cos := range append(metalCosines(t, metalLooksFile), nought) {
			got, want := made.Reflects(cos), glass.Reflects(cos)
			if got != want || !metalFinite(got) {
				t.Errorf("a film of index %v reflects %v at cosine %v, "+
					"glass %v", index, got, cos, want)
			}
		}
	}
	other := metalUnder(&gold, Film{Thickness: thick,
		Index: picked(t, "metal-dark-index")})
	if other.Reflects(unity) == glass.Reflects(unity) {
		t.Error("a film of another index reflects as glass does")
	}
}

// A metal with no index under a film that absorbs nothing reflects all of
// every colour at every look, grazing too, where the film's front reflects
// all and the bounces between film and metal would sum by dividing
// nothing by nothing. The anti-test: the same under an amber film does not.
func TestABareMetalUnderAClearFilmReflectsAll(t *testing.T) {
	glass := picked(t, "metal-glass-index")
	thick := picked(t, "metal-dark-thickness")
	lossless := metalUnder(&Metal{}, Film{Index: glass, Thickness: thick})
	tolerance := picked(t, "metal-film-tolerance")
	for _, cos := range append(metalCosines(t, metalLooksFile), nought) {
		got := lossless.Reflects(cos)
		if !metalSame(got, maths.Grey(unity), tolerance) {
			t.Errorf("a clear film over a bare metal reflects %v at cosine %v",
				got, cos)
		}
	}
	dim := metalUnder(&Metal{}, Film{Index: glass, Thickness: thick,
		Absorbs: metalAbsorbs(t, "amber")})
	if got := dim.Reflects(unity); !(got.B < unity-tolerance) {
		t.Errorf("an amber film over a bare metal reflects %v of blue", got.B)
	}
}
