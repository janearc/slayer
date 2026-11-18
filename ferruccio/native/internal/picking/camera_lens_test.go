package picking

import (
	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/hillside/placement"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
	"math"
	"os"
	"strings"
	"testing"
)

// A camera-specific lens must drive display and picking, as it does the final draw.
func TestCameraLensOverridesFilmInWorkingGeometry(t *testing.T) {
	raw, err := os.ReadFile("../../../../sets/ferruccio-playground.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, wide := range []float64{24, 70} {
		lens := "24"
		if wide == 70 {
			lens = "70"
		}
		source := strings.Replace(string(raw), "    arrival:\n", "    arrival:\n      wide: "+lens+"\n", 1)
		selected, err := New(source, 1.6)
		if err != nil {
			t.Fatal(err)
		}
		read, err := set.Read("camera-lens.yaml", []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		displayed, err := placement.NewDisplayCache().Scene(read)
		if err != nil {
			t.Fatal(err)
		}
		want := wide * maths.Degree
		if math.Abs(selected.lens.Wide-want) > 1e-12 || math.Abs(displayed.Lens.Wide-want) > 1e-12 {
			t.Fatalf("camera %g: picking=%g display=%g", wide, selected.lens.Wide, displayed.Lens.Wide)
		}
	}
}

// Zoom must edit the lens that actually wins in the producer.
func TestZoomEditsTheAuthoredCameraLens(t *testing.T) {
	raw, err := os.ReadFile("../../../../sets/ferruccio-playground.yaml")
	if err != nil {
		t.Fatal(err)
	}
	source := strings.Replace(string(raw), "    arrival:\n", "    arrival:\n      wide: 70\n", 1)
	document, err := draft.Open("camera-lens.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if err = document.Navigate(0, 0, .5); err != nil {
		t.Fatal(err)
	}
	read, err := set.Read("camera-lens.yaml", []byte(document.Text()))
	if err != nil {
		t.Fatal(err)
	}
	shots, err := read.Shots(func(string) (maths.Vec, bool) { return maths.Vec{}, false })
	if err != nil {
		t.Fatal(err)
	}
	if len(shots) != 1 || math.Abs(shots[0].Lens().Wide-35*maths.Degree) > 1e-12 {
		t.Fatal("zoom did not change the effective camera lens")
	}
	settings, _ := read.Film()
	if settings.Wide() != 40 {
		t.Fatal("zoom modified the film lens instead of the camera lens")
	}
}
