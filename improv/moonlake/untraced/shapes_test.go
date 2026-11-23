package main

import (
	"image"
	"image/color"
	"testing"

	"github.com/janearc/slayer/improv/moonlake/rig"
)

// blank is a transparent black picture of a size.
func blank(width, height int) *image.RGBA {
	return image.NewRGBA(image.Rect(0, 0, width, height))
}

// TestBlueprintGrades checks the grade at black, at white and at a mid
// grey, plain and peaked, against values worked by hand from the base
// and tint.
func TestBlueprintGrades(t *testing.T) {
	for _, c := range []struct {
		name   string
		peaked bool
		in     color.RGBA
		want   color.RGBA
	}{
		{"plain black", false, color.RGBA{0, 0, 0, 255}, color.RGBA{8, 12, 26, 255}},
		{"plain white", false, color.RGBA{255, 255, 255, 255},
			color.RGBA{70, 90, 130, 255}},
		{"plain grey", false, color.RGBA{128, 128, 128, 255},
			color.RGBA{48, 62, 93, 255}},
		{"peaked black", true, color.RGBA{0, 0, 0, 255}, color.RGBA{8, 12, 26, 255}},
		{"peaked white", true, color.RGBA{255, 255, 255, 255},
			color.RGBA{8, 12, 26, 255}},
		{"peaked quarter", true, color.RGBA{64, 64, 64, 255},
			color.RGBA{54, 70, 104, 255}},
	} {
		if got := blueprint(c.peaked)(c.in); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

// TestGlowAddsAndSaturates adds a share of a colour to a pixel, keeps the
// result within a byte, and ignores a pixel off the picture.
func TestGlowAddsAndSaturates(t *testing.T) {
	picture := blank(3, 3)
	glow(picture, 1, 1, [3]float64{1, 0.5, 0}, 0.5)
	if got, want := picture.RGBAAt(1, 1), (color.RGBA{127, 63, 0, 255}); got != want {
		t.Errorf("one glow made %v, want %v", got, want)
	}
	glow(picture, 1, 1, [3]float64{1, 0.5, 0}, 2)
	if got, want := picture.RGBAAt(1, 1), (color.RGBA{255, 255, 0, 255}); got != want {
		t.Errorf("a second glow made %v, want %v", got, want)
	}
	glow(picture, -1, 0, [3]float64{1, 1, 1}, 1)
	glow(picture, 3, 3, [3]float64{1, 1, 1}, 1)
	if lit(picture) != 1 {
		t.Errorf("a glow off the picture lit %d pixels, want only the one",
			lit(picture))
	}
}

// TestSpotFallsOffFromItsMiddle: a disc of radius two is full at its
// middle, half a unit out, and nothing at its radius.
func TestSpotFallsOffFromItsMiddle(t *testing.T) {
	picture := blank(9, 9)
	spot(picture, 4, 4, 2, [3]float64{1, 1, 1}, 1)
	for _, c := range []struct {
		x, y int
		want uint8
	}{{4, 4, 255}, {5, 4, 127}, {3, 4, 127}, {4, 5, 127}, {6, 4, 0}, {4, 2, 0}} {
		if got := picture.RGBAAt(c.x, c.y).R; got != c.want {
			t.Errorf("spot at %d %d is %d, want %d", c.x, c.y, got, c.want)
		}
	}
}

// TestStrokeIsALine: every pixel along a level line is lit and the rows
// beside it are not; a single point is one pixel.
func TestStrokeIsALine(t *testing.T) {
	picture := blank(8, 4)
	stroke(picture, 1, 1, 5, 1, [3]float64{1, 1, 1}, 1)
	for x := 0; x < 8; x++ {
		want := x >= 1 && x <= 5
		if got := picture.RGBAAt(x, 1).R > 0; got != want {
			t.Errorf("pixel %d of the line is lit %v, want %v", x, got, want)
		}
		if picture.RGBAAt(x, 0).R != 0 || picture.RGBAAt(x, 2).R != 0 {
			t.Errorf("the line spread to the rows beside it at %d", x)
		}
	}
	dot := blank(4, 4)
	stroke(dot, 2, 2, 2, 2, [3]float64{1, 1, 1}, 1)
	if lit(dot) != 1 {
		t.Errorf("a line of no length lit %d pixels", lit(dot))
	}
}

// TestLightenKeepsTheBrighter raises a dark pixel toward a colour and
// leaves a brighter channel as it was.
func TestLightenKeepsTheBrighter(t *testing.T) {
	picture := blank(1, 1)
	picture.SetRGBA(0, 0, color.RGBA{100, 100, 100, 255})
	lighten(picture, 0, 0, [3]float64{1, 0, 0}, 0.5)
	if got, want := picture.RGBAAt(0, 0), (color.RGBA{177, 100, 100, 255}); got != want {
		t.Errorf("lighten made %v, want %v", got, want)
	}
}

// TestInsideEitherWayRound checks a point in, a point out and a point on
// an edge for a triangle written both ways round.
func TestInsideEitherWayRound(t *testing.T) {
	a := [3][2]float64{{0, 0}, {10, 0}, {0, 10}}
	b := [3][2]float64{{0, 0}, {0, 10}, {10, 0}}
	for _, c := range []struct {
		x, y float64
		want bool
	}{{2, 2, true}, {9, 9, false}, {-1, 1, false}, {5, 5, true}, {5.1, 5.1, false}} {
		if inside(a, c.x, c.y) != c.want || inside(b, c.x, c.y) != c.want {
			t.Errorf("inside at %v %v is not %v for both windings", c.x, c.y,
				c.want)
		}
	}
}

// TestWedgeLightsItsTriangleOnly checks a pixel inside, a pixel outside
// and a triangle that runs off the picture.
func TestWedgeLightsItsTriangleOnly(t *testing.T) {
	picture := blank(10, 10)
	wedge(picture, [3][2]float64{{0, 0}, {8, 0}, {0, 8}}, [3]float64{1, 1, 1}, 1)
	if picture.RGBAAt(1, 1).R != 255 || picture.RGBAAt(7, 7).R != 0 ||
		picture.RGBAAt(9, 0).R != 0 {
		t.Error("the wedge did not fill its triangle and keep to it")
	}
	big := blank(4, 4)
	wedge(big, [3][2]float64{{-50, -50}, {60, -50}, {-50, 60}},
		[3]float64{1, 1, 1}, 1)
	if lit(big) != 16 {
		t.Errorf("a wedge over the picture lit %d of 16 pixels", lit(big))
	}
}

// TestOnPaneScalesAndShifts: a rig point's place on a pane doubles with
// the pane's size, and a window moved right moves it left.
func TestOnPaneScalesAndShifts(t *testing.T) {
	pane := overhead(t)
	at := rig.Point{200, -100, 0}
	x, y, ahead := pane.onPane(at, image.Pt(200, 200))
	if !ahead {
		t.Fatal("a point under the camera is not ahead")
	}
	x2, y2, _ := pane.onPane(at, image.Pt(400, 400))
	if !within(x2, 2*(x+0.5)-0.5) || !within(y2, 2*(y+0.5)-0.5) {
		t.Errorf("at twice the size the point is %v %v, from %v %v", x2, y2, x, y)
	}
	pane.part = image.Rect(10, 20, 210, 220)
	x3, y3, _ := pane.onPane(at, image.Pt(200, 200))
	if !within(x3, x-10) || !within(y3, y-20) {
		t.Errorf("with the window moved the point is %v %v, from %v %v", x3, y3, x, y)
	}
	behind := rig.Point{0, 0, 9000}
	if _, _, ahead := pane.onPane(behind, image.Pt(200, 200)); ahead {
		t.Error("a point over the camera was ahead of it")
	}
}

// within reports whether two numbers are the same to a small tolerance.
func within(got, want float64) bool {
	return got-want < 1e-6 && want-got < 1e-6
}

// TestPaneTakesItsPartAtASize: each pixel of the picture is the graded
// still at the middle of the pixel's share of the window.
func TestPaneTakesItsPartAtASize(t *testing.T) {
	still := blank(100, 100)
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			still.SetRGBA(x, y, color.RGBA{uint8(x), uint8(y), 7, 255})
		}
	}
	grade := func(was color.RGBA) color.RGBA {
		return color.RGBA{was.R, was.G, was.B + 1, 255}
	}
	pane := view{still: still, part: image.Rect(20, 40, 60, 80), grade: grade}
	got := pane.picture(image.Pt(20, 20))
	// a window 40 wide at 20 across is two still pixels to one
	for _, c := range []struct{ x, y int }{{0, 0}, {3, 5}, {19, 19}} {
		redOffset := int((float64(c.x) + 0.5) * 2)
		greenOffset := int((float64(c.y) + 0.5) * 2)
		want := color.RGBA{uint8(20 + redOffset),
			uint8(40 + greenOffset), 8, 255}
		if at := got.RGBAAt(c.x, c.y); at != want {
			t.Errorf("pixel %d %d is %v, want %v", c.x, c.y, at, want)
		}
	}
}

// TestMarkOutlinesAShape: a square mark has its corner a mark radius
// out from the fixture and its middle dark; one behind the camera draws
// nothing.
func TestMarkOutlinesAShape(t *testing.T) {
	pane := overhead(t)
	picture := blank(200, 200)
	at := rig.Point{}
	x, y, _ := pane.onPane(at, picture.Rect.Size())
	pane.mark(picture, at, shape{corners: 4, turn: 0}, [3]float64{1, 1, 1})
	if picture.RGBAAt(int(x+markRadius), int(y)).R == 0 {
		t.Error("the mark's corner is dark")
	}
	if picture.RGBAAt(int(x), int(y)).R != 0 {
		t.Error("the mark's middle is lit")
	}
	round := blank(200, 200)
	pane.mark(round, at, shape{}, [3]float64{1, 1, 1})
	if lit(round) < markSteps/2 {
		t.Errorf("a round mark lit only %d pixels", lit(round))
	}
	none := blank(200, 200)
	pane.mark(none, rig.Point{0, 0, 9000}, shape{corners: 4}, [3]float64{1, 1, 1})
	if lit(none) != 0 {
		t.Error("a mark behind the camera was drawn")
	}
}

// TestMiddleIsTheFirstPlaceSeen: places the pane cannot see are passed
// over, and a pane that sees none says so.
func TestMiddleIsTheFirstPlaceSeen(t *testing.T) {
	pane := overhead(t)
	size := image.Pt(200, 200)
	off := rig.Point{0, 0, 9000}
	out := rig.Point{900000, 0, 0}
	want, wantY, _ := pane.onPane(rig.Point{}, size)
	x, y, seen := pane.middle([]rig.Point{off, out, {}}, size)
	if !seen || !within(x, want) || !within(y, wantY) {
		t.Errorf("the middle is %v %v %v, want %v %v", x, y, seen, want, wantY)
	}
	if _, _, seen := pane.middle([]rig.Point{off, out}, size); seen {
		t.Error("places the pane cannot see were seen")
	}
}

// TestPulseLightsAMarkByItsLevel draws nothing under the least level and
// a glow at a lit one, brighter the higher the level.
func TestPulseLightsAMarkByItsLevel(t *testing.T) {
	pane := overhead(t)
	at := rig.Point{}
	x, y, _ := pane.onPane(at, image.Pt(200, 200))
	dim, bright, none := blank(200, 200), blank(200, 200), blank(200, 200)
	pane.pulse(none, at, [3]float64{1, 1, 1}, shown/2)
	pane.pulse(dim, at, [3]float64{1, 1, 1}, 0.25)
	pane.pulse(bright, at, [3]float64{1, 1, 1}, 1)
	if lit(none) != 0 {
		t.Error("a level under the least lit the pane")
	}
	d, b := dim.RGBAAt(int(x), int(y)).R, bright.RGBAAt(int(x), int(y)).R
	if d == 0 || b <= d {
		t.Errorf("the glow is %d at a quarter and %d at full", d, b)
	}
	behind := blank(200, 200)
	pane.pulse(behind, rig.Point{0, 0, 9000}, [3]float64{1, 1, 1}, 1)
	if lit(behind) != 0 {
		t.Error("a pulse behind the camera was drawn")
	}
}

// TestFanIsLinesThenASheet: a few beams are lines only; many are a sheet
// of light that lights more of the pane.
func TestFanIsLinesThenASheet(t *testing.T) {
	pane := overhead(t)
	origin := rig.Point{}
	var few, many []rig.Point
	for k := 0; k < 3; k++ {
		few = append(few, rig.Point{float64(100 * k), 300, 0})
	}
	for k := 0; k < 20; k++ {
		many = append(many, rig.Point{float64(10 * k), 300, 0})
	}
	lines, sheet, same := blank(200, 200), blank(200, 200), blank(200, 200)
	pane.fan(lines, origin, few, [3]float64{1, 1, 1}, 1)
	pane.fan(sheet, origin, many, [3]float64{1, 1, 1}, 1)
	for _, end := range few {
		pane.ray(same, origin, end, [3]float64{1, 1, 1}, beamShare)
	}
	if lit(lines) == 0 || lit(lines) != lit(same) {
		t.Errorf("a fan of three lit %d pixels, three rays %d", lit(lines),
			lit(same))
	}
	if lit(sheet) <= lit(lines) {
		t.Errorf("a sheet of twenty lit %d pixels, a fan of three %d",
			lit(sheet), lit(lines))
	}
}

// TestFiringLevelsByCue: each fixture's level follows its cue, as the
// panel's lamps do.
func TestFiringLevelsByCue(t *testing.T) {
	state := cues{rig: 1, gate: 0.5, gateLeft: 0.25, gateRight: -1, white: 1,
		magenta: 0, search: 0.8, searchGate: 0.5, searchMove: "chase",
		searchMovePhase: 0, floodLeft: 0.4, floodRight: 3, overhead: 2}
	now := firing(state, 1)
	// banks: the heads are left, right, right, left
	for k, want := range []float64{0.25, 0.5, 0.5, 0.25} {
		if got := now.head(k); got != want {
			t.Errorf("head %d is %v, want %v", k, got, want)
		}
	}
	if now.strobe(0) != 1 || now.strobe(1) != 0 {
		t.Errorf("strobes %v %v, want 1 0", now.strobe(0), now.strobe(1))
	}
	if got := now.search(0); !within(got, 0.4) {
		t.Errorf("the running searchlight is %v, want 0.4", got)
	}
	if got := now.search(3); !within(got, 0.4*chaseDim) {
		t.Errorf("a chase's other searchlight is %v, want %v", got, 0.4*chaseDim)
	}
	if now.flood(0) != 0.4 || now.flood(1) != 1 || now.ship(0) != 1 {
		t.Errorf("floods %v %v ship %v", now.flood(0), now.flood(1), now.ship(0))
	}
	state.rig = 0
	if got := firing(state, 1).head(0); got != 0 {
		t.Errorf("a head with the rig at nought is %v", got)
	}
}

// TestLasersByBankAndParity: a blackout draws nothing, a bank shut below
// the least level draws nothing, and a split pulse draws fewer beams.
func TestLasersByBankAndParity(t *testing.T) {
	pane := overhead(t)
	base := cues{rig: 1, gate: 1, gateLeft: -1, gateRight: -1, beams: 4,
		figure: "fan", parity: -1, sweep: -1, colour: [3]float64{1, 1, 1}}
	draw := func(state cues) int {
		picture := blank(200, 200)
		pane.lasers(picture, state, layout, 1)
		return lit(picture)
	}
	full := draw(base)
	if full == 0 {
		t.Fatal("a rig at one drew nothing")
	}
	dark := base
	dark.rig = 0
	if draw(dark) != 0 {
		t.Error("a rig of nought drew beams")
	}
	shut := base
	shut.gateLeft, shut.gateRight = 0.001, 0.001
	if draw(shut) != 0 {
		t.Error("both banks shut drew beams")
	}
	half := base
	half.parity = 0
	if got := draw(half); got == 0 || got >= full {
		t.Errorf("a split pulse drew %d pixels against the whole's %d", got, full)
	}
	unset := base
	unset.rig = -1
	if draw(unset) == 0 {
		t.Error("a rig left to the show drew nothing")
	}
}
