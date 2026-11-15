package diagram

import (
	"image/color"
	"math"
	"testing"

	"github.com/janearc/slayer/improv/quasar/rings"
)

// The numbers the tests choose: the quasar slice, 2026-11-12.
const (
	testSide  = 300
	testReach = 15
	// testEye and testLean are smoke 36's eye: fifty masses out, ten
	// degrees above the disc.
	testEye   = 50
	testLean  = 10 * math.Pi / 180
	testInner = 6
	testOuter = 60
	// offEdge is an angle past the shadow's edge, in radians, whose ray
	// lands behind the hole on the disc's first crossing.
	offEdge = 0.05
	// channelSlack is how far a smoothed pixel may be from the colour
	// painted, on each channel.
	channelSlack = 40
)

var testRay = color.RGBA{R: 80, G: 220, B: 255, A: opaque}

func testView(t *testing.T) View {
	t.Helper()
	view, err := NewView(testSide, testReach)
	if err != nil {
		t.Fatal(err)
	}
	return view
}

func near(got color.RGBA, want color.RGBA) bool {
	far := func(a, b uint8) bool {
		return math.Abs(float64(a)-float64(b)) > channelSlack
	}
	return !far(got.R, want.R) && !far(got.G, want.G) && !far(got.B, want.B)
}

func TestPlacePutsTheCentreInTheMiddleAndUpUp(t *testing.T) {
	view := testView(t)
	x, y := view.Place(rings.Point{})
	if x != testSide/two || y != testSide/two {
		t.Fatalf("the centre is at %v, %v", x, y)
	}
	_, above := view.Place(rings.Point{Up: one})
	if !(above < y) {
		t.Fatal("up is drawn down")
	}
	edge, _ := view.Place(rings.Point{Along: testReach})
	if edge != testSide {
		t.Fatalf("the reach is at %v, not the edge", edge)
	}
}

func TestNewViewRefusesNoPictureAndNoHorizon(t *testing.T) {
	if _, err := NewView(none, testReach); err == nil {
		t.Error("a side of no pixels was taken")
	}
	for _, reach := range []float64{horizon, math.NaN(), math.Inf(1)} {
		if _, err := NewView(testSide, reach); err == nil {
			t.Errorf("a reach of %v was taken", reach)
		}
	}
}

func TestDrawPaintsTheHoleTheDiscAndWhereTheRayLands(t *testing.T) {
	view := testView(t)
	disc, err := rings.NewDisc(testInner, testOuter)
	if err != nil {
		t.Fatal(err)
	}
	lens, err := rings.NewLens(testEye, testLean)
	if err != nil {
		t.Fatal(err)
	}
	path := lens.Follow(lens.Edge() + offEdge)
	if !path.Lands(none, disc) {
		t.Fatal("the test's ray does not land on its first crossing")
	}
	picture := view.Draw(disc, []Ray{{Path: path, Colour: testRay}})
	x, y := view.Place(rings.Point{})
	if got := picture.RGBAAt(int(x), int(y)); !near(got, black) {
		t.Errorf("the hole's middle is %v", got)
	}
	x, y = view.Place(rings.Point{Along: (testInner + testReach) / two})
	if got := picture.RGBAAt(int(x), int(y)); !near(got, discColour) {
		t.Errorf("the disc is %v", got)
	}
	landed := -path.Crossings()[none].Radius
	x, y = view.Place(rings.Point{Along: landed})
	if got := picture.RGBAAt(int(x), int(y)); !near(got, testRay) {
		t.Errorf("where the ray lands, %v masses behind, is %v",
			-landed, got)
	}
}

func TestARayThatDoesNotLandHasNoDot(t *testing.T) {
	view := testView(t)
	disc, err := rings.NewDisc(testInner, testOuter)
	if err != nil {
		t.Fatal(err)
	}
	lens, err := rings.NewLens(testEye, testLean)
	if err != nil {
		t.Fatal(err)
	}
	path := lens.Follow(lens.Edge() + offEdge)
	picture := view.Draw(disc, []Ray{{Path: path, Order: two,
		Colour: testRay}})
	landed := -path.Crossings()[none].Radius
	x, y := view.Place(rings.Point{Along: landed, Up: -one})
	if got := picture.RGBAAt(int(x), int(y)); near(got, testRay) {
		t.Errorf("a ray asked to land at an order it never made has "+
			"a dot below its first crossing: %v", got)
	}
}
