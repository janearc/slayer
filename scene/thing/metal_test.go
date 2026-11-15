package thing

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// gold is Johnson and Christy's gold (1972), at their wavelengths
// nearest the tracer's red, green and blue: 617, 549 and 471 nm.
var gold = Metal{N: [3]float64{0.21, 0.43, 1.31}, K: [3]float64{3.272, 2.455, 1.849}}

// nearly is whether got is within tolerance of want; a NaN never is.
func nearly(got, want, tolerance float64) bool {
	return math.Abs(got-want) <= tolerance
}

// Face on, a conductor reflects ((n-1)^2 + k^2) / ((n+1)^2 + k^2): for
// Johnson and Christy's gold, worked with awk, 0.930978291 of red,
// 0.786915760 of green and 0.401478098 of blue. That is gold's colour.
// The anti-test: at grazing it is no colour at all, for every surface
// mirrors all of every colour there.
func TestGoldReflectsMostOfRedAndLeastOfBlueFaceOn(t *testing.T) {
	made := Material{Gloss: 1, Metal: &gold}
	got := made.Reflects(1)
	want := maths.RGB(0.930978291, 0.786915760, 0.401478098)
	if !nearly(got.R, want.R, 1e-9) || !nearly(got.G, want.G, 1e-9) ||
		!nearly(got.B, want.B, 1e-9) {
		t.Errorf("gold reflects %v face on, want %v", got, want)
	}
	if got := made.Reflects(0); !nearly(got.R, 1, 1e-12) ||
		!nearly(got.G, 1, 1e-12) || !nearly(got.B, 1, 1e-12) {
		t.Errorf("gold reflects %v at grazing, want all of every colour", got)
	}
}

// At grazing every surface reflects all of every colour, and from face
// on to grazing a metal's share never leaves zero to one.
func TestAConductorReflectsAllAtGrazingAndNeverMore(t *testing.T) {
	for c := range 3 {
		if got := Conductor(0, gold.N[c], gold.K[c]); !nearly(got, 1, 1e-12) {
			t.Errorf("gold's colour %d reflects %v at grazing", c, got)
		}
		for step := range 101 {
			cos := float64(step) / 100
			if got := Conductor(cos, gold.N[c], gold.K[c]); !(got >= 0 && got <= 1) {
				t.Fatalf("gold's colour %d reflects %v at cosine %v", c, got, cos)
			}
		}
	}
}

// A conductor with no k is a clear dielectric: the two forms of
// Fresnel's equations agree for glass at every angle. The anti-test: a
// conductor that does absorb is no dielectric, and face on it mirrors
// more than glass of its n does.
func TestAConductorWithNoAbsorptionIsADielectric(t *testing.T) {
	for step := range 21 {
		cos := float64(step) / 20
		if got, want := Conductor(cos, 1.5, 0), Dielectric(cos, 1.5); !nearly(got, want, 1e-12) {
			t.Errorf("at cosine %v the conductor gives %v, the dielectric %v",
				cos, got, want)
		}
	}
	if got, glass := Conductor(1, 1.5, 2), Dielectric(1, 1.5); !(got > glass+0.3) {
		t.Errorf("a conductor with a k of 2 mirrors %v face on, glass %v",
			got, glass)
	}
	// face on, glass reflects ((1.5-1)/(1.5+1))^2 = 0.04
	if got := Dielectric(1, 1.5); !nearly(got, 0.04, 1e-15) {
		t.Errorf("glass reflects %v face on, want 0.04", got)
	}
	// from inside water, past its critical angle, all of it
	if got := Dielectric(0.3, 1/1.333); got != 1 {
		t.Errorf("past the critical angle water reflects %v, want all", got)
	}
}

// A film that is not there leaves the metal bare: index one, no
// thickness. The anti-test is a film that lets nothing through, which
// shows none of the metal under it: only its own front.
func TestAFilmBetweenNothingAndEverything(t *testing.T) {
	bare := Material{Gloss: 1, Metal: &gold}
	thin := Material{Gloss: 1, Metal: &gold, Film: &Film{Index: 1}}
	dark := Material{Gloss: 1, Metal: &gold, Film: &Film{Index: 1.7,
		Thickness: 25e-6, Absorbs: [3]float64{1e9, 1e9, 1e9}}}
	for _, cos := range []float64{1, 0.7, 0.3, 0.05} {
		b, f := bare.Reflects(cos), thin.Reflects(cos)
		if !nearly(b.R, f.R, 1e-12) || !nearly(b.G, f.G, 1e-12) || !nearly(b.B, f.B, 1e-12) {
			t.Errorf("at cosine %v an empty film gives %v, the bare metal %v",
				cos, f, b)
		}
		front := Dielectric(cos, 1.7)
		d := dark.Reflects(cos)
		if !nearly(d.R, front, 1e-12) || !nearly(d.B, front, 1e-12) {
			t.Errorf("at cosine %v an opaque film gives %v, its front %v",
				cos, d, front)
		}
	}
}

// Face on, Johnson and Christy's titanium, worked with awk the same
// way: 0.608895790 of red, 0.581832774 of green, 0.550221981 of blue.
// The anti-test: it is a grey and not a colour, its red and its blue
// within six hundredths, where gold's are half apart.
func TestTitaniumIsAWarmGreyFaceOn(t *testing.T) {
	made := Material{Gloss: 1, Metal: Titanium()}
	got := made.Reflects(1)
	want := maths.RGB(0.608895790, 0.581832774, 0.550221981)
	if !nearly(got.R, want.R, 1e-9) || !nearly(got.G, want.G, 1e-9) ||
		!nearly(got.B, want.B, 1e-9) {
		t.Errorf("titanium reflects %v face on, want %v", got, want)
	}
	yellow := (&Material{Gloss: 1, Metal: Gold()}).Reflects(1)
	if !(got.R-got.B < 0.06) || !(yellow.R-yellow.B > 0.5) {
		t.Errorf("titanium's red and blue are %v apart, gold's %v",
			got.R-got.B, yellow.R-yellow.B)
	}
}

// Face on, the other three metals, worked with awk the same way from
// the readings materials holds: silver 0.986930029 of red, 0.982836296
// of green and 0.978572181 of blue; aluminium 0.913060575, 0.914118518
// and 0.919091051; beryllium 0.537446060, 0.535614617 and 0.535542035.
// Silver and aluminium are white mirrors and beryllium a grey one. The
// anti-test: none of the three is a colour, its three shares within a
// hundredth of one another.
func TestSilverAluminiumAndBerylliumFaceOn(t *testing.T) {
	metals := map[string]struct {
		made *Metal
		want maths.Light
	}{
		"silver":    {Silver(), maths.RGB(0.986930029, 0.982836296, 0.978572181)},
		"aluminium": {Aluminium(), maths.RGB(0.913060575, 0.914118518, 0.919091051)},
		"beryllium": {Beryllium(), maths.RGB(0.537446060, 0.535614617, 0.535542035)},
	}
	for name, each := range metals {
		got := (&Material{Gloss: 1, Metal: each.made}).Reflects(1)
		if !nearly(got.R, each.want.R, 1e-9) || !nearly(got.G, each.want.G, 1e-9) ||
			!nearly(got.B, each.want.B, 1e-9) {
			t.Errorf("%s reflects %v face on, want %v", name, got, each.want)
		}
		most := math.Max(got.R, math.Max(got.G, got.B))
		least := math.Min(got.R, math.Min(got.G, got.B))
		if !(most-least < 0.01) {
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
	const thick = 40e-6
	film := Kapton(thick)
	if film.Index != 1.7 || film.Thickness != thick {
		t.Errorf("Kapton is %+v", *film)
	}
	want := [3]float64{0.799315, 0.301194, 0.001114}
	for colour, absorbs := range film.Absorbs {
		if got := math.Exp(-absorbs * thick); !nearly(got, want[colour], 1e-6) {
			t.Errorf("40 microns of Kapton passes %v of colour %d, want %v",
				got, colour, want[colour])
		}
	}
	blanket := Material{Gloss: 1, Metal: Aluminium(), Film: Kapton(25e-6)}
	if got := blanket.Reflects(1); !(got.R > got.G && got.G > got.B) {
		t.Errorf("Kapton over aluminium reflects %v face on: not gold", got)
	}
	none := Material{Gloss: 1, Metal: Aluminium(), Film: Kapton(0)}
	if got := none.Reflects(1); got.R > got.G && got.G > got.B {
		t.Errorf("Kapton of no thickness over aluminium reflects %v: gold", got)
	}
}

// Rutile is as thick as it is asked to be, at Devore's index. The
// anti-test is in the same line: it is clear, and absorbs nothing in
// any colour, where Kapton's film does.
func TestRutileIsAClearFilmAtDevoresIndex(t *testing.T) {
	const thick = 5e-9
	oxide := Rutile(thick)
	if oxide.Index != 2.65 || oxide.Thickness != thick ||
		oxide.Absorbs != [3]float64{} {
		t.Errorf("rutile is %+v", *oxide)
	}
}

// Amber over aluminium: a film that keeps red and swallows blue turns
// a silver metal gold, as Kapton turns a blanket's aluminium. The
// anti-test is the same metal bare, which is not gold.
func TestAnAmberFilmGildsSilverMetal(t *testing.T) {
	aluminium := Metal{N: [3]float64{1.1911, 0.93525, 0.6444},
		K: [3]float64{7.0711, 6.3099, 5.3994}}
	bare := Material{Gloss: 1, Metal: &aluminium}
	if got := bare.Reflects(1); got.R > got.G && got.G > got.B {
		t.Errorf("bare aluminium reflects %v face on: gold", got)
	}
	blanket := Material{Gloss: 1, Metal: &aluminium, Film: &Film{Index: 1.7,
		Thickness: 25e-6, Absorbs: [3]float64{4e3, 3e4, 2e5}}}
	got := blanket.Reflects(1)
	if !(got.R > got.G && got.G > got.B) {
		t.Errorf("Kapton over aluminium reflects %v face on: not gold", got)
	}
}

// Anything but a metal reflects as it always has: the same share of all
// three colours, to the last bit, so no picture made before moves. The
// anti-test: a metal with no gloss on it mirrors nothing, whatever its
// colour.
func TestANonMetalReflectsAsMirrorsDoes(t *testing.T) {
	glass := Material{Gloss: 1, Index: 1.5}
	for _, cos := range []float64{1, 0.5, 0.1, 0} {
		want := glass.Mirrors(cos)
		if got := glass.Reflects(cos); got != maths.Grey(want) {
			t.Errorf("glass reflects %v at cosine %v, Mirrors %v", got, cos, want)
		}
	}
	plain := Material{Colour: maths.Grey(0.5), Metal: &gold}
	if got := plain.Reflects(1); got != (maths.Light{}) {
		t.Errorf("a metal with no gloss reflects %v", got)
	}
}
