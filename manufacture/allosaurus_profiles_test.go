package manufacture

import (
	"math"
	"strings"
	"testing"
)

func TestAllosaurusProfilesMatchIndependentWalkingWitness(t *testing.T) {
	book, err := ReadTheropodProfiles("allosaurs.yaml", allosaurusProfilesRaw)
	if err != nil {
		t.Fatal(err)
	}
	for seconds, want := range allosaurusBaselineWitness {
		row, err := book.NewAllosaurus(seconds)
		if err != nil {
			t.Fatal(err)
		}
		made, err := row.how.made(sow(7))
		if err != nil {
			t.Fatal(err)
		}
		if got := birdAllosaurusWitness(made.native); got != want {
			t.Errorf("time %g: %s != %s", seconds, got, want)
		}
	}
	if _, err := book.NewTheropod("absent", 0); err == nil {
		t.Fatal("accepted absent species")
	}
	if _, err := (*TheropodProfiles)(nil).NewAllosaurus(0); err == nil {
		t.Fatal("accepted nil book")
	}
	for _, seconds := range []float64{math.NaN(), math.Inf(1), 100001} {
		if _, err := book.NewAllosaurus(seconds); err == nil {
			t.Fatalf("accepted time %g", seconds)
		}
	}
}

func TestAllosaurusProfilesValidateBodyData(t *testing.T) {
	base := string(allosaurusProfilesRaw)
	for _, tc := range []struct{ name, from, to string }{
		{"null scale", "scale: 1", "scale: null"},
		{"missing scale", "    scale: 1\n", ""},
		{"bad scaffold", "scaffold: allosaurus-study", "scaffold: universal-dinosaur"},
		{"missing station", "      - [-4.2, 1.56, 0.02, 0.02, 1]\n", ""},
		{"station nan", "[-4.2, 1.56, 0.02, 0.02, 1]", "[-4.2, .nan, 0.02, 0.02, 1]"},
		{"station null", "[-4.2, 1.56, 0.02, 0.02, 1]", "[-4.2, null, 0.02, 0.02, 1]"},
		{"collapsed station", "[-4.2, 1.56, 0.02, 0.02, 1]", "[-4.2, 1.56, 1e-300, 1e-300, 1]"},
		{"nearly coincident stations", "[-3.7, 1.61, 0.09, 0.08, 1]", "[-4.1999999, 1.61, 0.09, 0.08, 1]"},
		{"tiny scale", "scale: 1", "scale: 1e-300"},
		{"inverted stations", "[-4.2, 1.56, 0.02, 0.02, 1]", "[4.2, 1.56, 0.02, 0.02, 1]"},
		{"bad colour", "hide: [0.245, 0.265, 0.185]", "hide: [1.5, 0.265, 0.185]"},
		{"merge", "kind: allosaurus", "<<: {kind: allosaurus}"},
		{"unknown field", "kind: allosaurus", "unknown: field\n    kind: allosaurus"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(base, tc.from) {
				t.Fatal("fixture absent")
			}
			raw := strings.Replace(base, tc.from, tc.to, 1)
			if _, err := ReadTheropodProfiles("allosaurs.yaml", []byte(raw)); err == nil {
				t.Fatal("accepted invalid profile")
			}
		})
	}
	for name, raw := range map[string]string{"second document": base + "\n---\nprofiles: []\n", "malformed trailing": base + "\n---\nprofiles: [\n"} {
		t.Run(name, func(t *testing.T) {
			if _, err := ReadTheropodProfiles("allosaurs.yaml", []byte(raw)); err == nil {
				t.Fatal("accepted extra document")
			}
		})
	}
}

func TestAllosaurusProfileRefusesInterpolatedInvertedRadii(t *testing.T) {
	text := string(allosaurusProfilesRaw)
	for _, pair := range [][2]string{
		{"[-2.2, 1.72, 0.33, 0.25, 1]", "[-2.2, 1.72, 2, 2, 1]"},
		{"[-1.5, 1.79, 0.47, 0.34, 1]", "[-1.5, 1.79, 0.000001, 0.000001, 1]"},
		{"[-0.8, 1.85, 0.60, 0.45, 1]", "[-0.8, 1.85, 0.000001, 0.000001, 1]"},
		{"[-0.2, 1.88, 0.67, 0.49, 1]", "[-0.2, 1.88, 2, 2, 1]"},
	} {
		if !strings.Contains(text, pair[0]) {
			t.Fatalf("missing station %q", pair[0])
		}
		text = strings.Replace(text, pair[0], pair[1], 1)
	}
	if _, err := ReadTheropodProfiles("allosaurs.yaml", []byte(text)); err == nil || !strings.Contains(err.Error(), "invalid interpolated station") {
		t.Fatalf("cubic inverted radii: %v", err)
	}
}

func TestAllosaurusProfilesRetainIndependentAnatomy(t *testing.T) {
	original := append([]byte(nil), allosaurusProfilesRaw...)
	first, err := ReadTheropodProfiles("first.yaml", original)
	if err != nil {
		t.Fatal(err)
	}
	from, to := "[-0.2, 1.88, 0.67, 0.49, 1]", "[-0.2, 1.88, 0.80, 0.49, 1]"
	if !strings.Contains(string(original), from) {
		t.Fatal("station fixture absent")
	}
	second, err := ReadTheropodProfiles("second.yaml", []byte(strings.Replace(string(original), from, to, 1)))
	if err != nil {
		t.Fatal(err)
	}
	a, _ := first.NewAllosaurus(.5)
	b, _ := second.NewAllosaurus(.5)
	for i := range original {
		original[i] = 'x'
	}
	one, err := a.how.made(sow(7))
	if err != nil {
		t.Fatal(err)
	}
	two, err := b.how.made(sow(7))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := birdAllosaurusWitness(one.native), allosaurusBaselineWitness[.5]; got != want {
		t.Fatalf("original book changed: %s != %s", got, want)
	}
	if birdAllosaurusWitness(one.native) == birdAllosaurusWitness(two.native) {
		t.Fatal("station edit changed no anatomy")
	}
}

func TestAllosaurusProfileScaleAndBlackAreApplied(t *testing.T) {
	base := string(allosaurusProfilesRaw)
	changed := strings.Replace(base, "scale: 1", "scale: 2", 1)
	changed = strings.Replace(changed, "hide: [0.245, 0.265, 0.185]", "hide: [0, 0, 0]", 1)
	if changed == base {
		t.Fatal("fixture not changed")
	}
	book, err := ReadTheropodProfiles("changed.yaml", []byte(changed))
	if err != nil {
		t.Fatal(err)
	}
	row, err := book.NewAllosaurus(0)
	if err != nil {
		t.Fatal(err)
	}
	made, err := row.how.made(sow(7))
	if err != nil {
		t.Fatal(err)
	}
	if made.native[0].Material.Colour.R != 0 || made.native[0].Material.Colour.G != 0 || made.native[0].Material.Colour.B != 0 {
		t.Fatal("black hide was not applied")
	}
	ordinary, err := defaultAllosaurusProfileKind(0).made(sow(7))
	if err != nil {
		t.Fatal(err)
	}
	a, b := made.Bounds(), ordinary.Bounds()
	if math.Abs(a.Min.X-2*b.Min.X) > 1e-9 || math.Abs(a.Max.Y-2*b.Max.Y) > 1e-9 {
		t.Fatalf("scale not reflected in bounds: %v vs %v", a, b)
	}
}

func defaultAllosaurusProfileKind(seconds float64) theropodKind {
	book, err := ReadTheropodProfiles("allosaurus-profiles.yaml", allosaurusProfilesRaw)
	if err != nil {
		panic(err)
	}
	row, err := book.NewAllosaurus(seconds)
	if err != nil {
		panic(err)
	}
	return row.how.(theropodKind)
}
