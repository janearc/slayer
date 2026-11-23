package draft

import (
	"strings"
	"testing"
)

// TestZoomRefusesAFieldOfViewOutsideItsRange checks that an authored angle
// outside 5 to 120 degrees is refused in words and left exactly as written,
// for a catalogue and for a set.
func TestZoomRefusesAFieldOfViewOutsideItsRange(t *testing.T) {
	for name, fixture := range map[string]struct{ file, from, ending string }{
		"catalogue": {"subaru-settings.yaml", "wide: 50\n", "\n"},
		"set":       {"ferruccio-playground.yaml", "wide: 40", ""},
	} {
		for _, authored := range []string{"0.5", "150"} {
			t.Run(name+" "+authored, func(t *testing.T) {
				source := strings.Replace(string(exampleSource(t, fixture.file)), fixture.from, "wide: "+authored+fixture.ending, 1)
				document, err := Open(fixture.file, []byte(source))
				if err != nil {
					t.Fatal(err)
				}
				before := document.Text()
				err = document.Navigate(10, 5, 1.1)
				if err == nil || !strings.Contains(err.Error(), "left as written") {
					t.Fatalf("Navigate = %v; want a refusal that says it is left as written", err)
				}
				if document.Text() != before {
					t.Fatal("a refused zoom changed the source, including the orbit")
				}
			})
		}
	}
}

// TestZoomStillWorksInsideItsRange is the anti-test: an in-range angle zooms
// as before.
func TestZoomStillWorksInsideItsRange(t *testing.T) {
	document, err := Open("subaru-settings.yaml", exampleSource(t, "subaru-settings.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := document.Navigate(0, 0, 1.1); err != nil {
		t.Fatalf("in-range zoom refused: %v", err)
	}
	if got, err := document.Value("wide"); err != nil || got != "55" {
		t.Fatalf("wide = %q, %v; want 55", got, err)
	}
}

// TestACatalogueRefusedForAnotherReasonKeepsItsOwnWords: the zoom's words
// are for an angle out of range and nothing else. A catalogue whose angle is
// no number is refused as the schema refuses it, and is left as written.
func TestACatalogueRefusedForAnotherReasonKeepsItsOwnWords(t *testing.T) {
	source := strings.Replace(string(exampleSource(t, "subaru-settings.yaml")),
		"wide: 50\n", "wide: fifty\n", 1)
	document, err := Open("subaru-settings.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	before := document.Text()
	err = document.Navigate(0, 0, 1.1)
	if err == nil || strings.Contains(err.Error(), "left as written") {
		t.Fatalf("Navigate = %v; want the schema's own refusal", err)
	}
	if document.Text() != before {
		t.Fatal("a refused zoom changed the source")
	}
}

// TestAZoomedAngleIsWrittenAsMeant holds the angle a zoom writes: the
// product to a millionth of a degree, inside the range zoom works in. Each
// want is the product by hand: 50 by 1.1 is 55, 40 by 1.1 is 44.
func TestAZoomedAngleIsWrittenAsMeant(t *testing.T) {
	for _, each := range []struct{ wide, factor, want float64 }{
		{50, 1.1, 55},
		{40, 1.1, 44},
		{100, 2, maximumWide},
		{6, 0.5, minimumWide},
	} {
		if got := zoomed(each.wide, each.factor); got != each.want {
			t.Errorf("%g zoomed by %g is %v; want %v", each.wide, each.factor,
				got, each.want)
		}
	}
}

// TestASmallZoomIsNotSwallowed is its anti-test: a change of five millionths
// of a degree, five times the grain, is kept, so the rounding hides no zoom.
func TestASmallZoomIsNotSwallowed(t *testing.T) {
	const wide, factor, want = 50, 1.0000001, 50.000005
	if got := zoomed(wide, factor); got != want {
		t.Errorf("%v zoomed by %v is %v; want %v", float64(wide),
			float64(factor), got, float64(want))
	}
}
