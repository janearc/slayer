package main

import (
	"image"
	"strings"
	"testing"

	"github.com/janearc/slayer/improv/moonlake/rig"
	"github.com/janearc/slayer/improv/walk"
	"github.com/janearc/slayer/maths"
)

// overhead is a pane's view straight down from 3 km over the show's eye,
// rows running east, the picture 200 across, through a 50 mm pinhole
// on a square frame, so every fixture lands on it.
func overhead(t *testing.T) view {
	t.Helper()
	eyeAt := maths.V(eye[0], eye[2]+3000, 0-eye[1])
	down := maths.V(0, -1, 0)
	seen, err := walk.NewEye(eyeAt, eyeAt.Add(down),
		maths.V(0, 0, -1))
	if err != nil {
		t.Fatal(err)
	}
	camera, err := walk.PinholeCamera(seen, 90, image.Pt(200, 200), "m")
	if err != nil {
		t.Fatal(err)
	}
	still := image.NewRGBA(image.Rect(0, 0, 200, 200))
	return view{camera: camera, still: still, part: still.Rect}
}

// lit counts the pixels of a picture brighter than black.
func lit(picture *image.RGBA) int {
	count := 0
	for i := 0; i < len(picture.Pix); i += 4 {
		if picture.Pix[i]+picture.Pix[i+1]+picture.Pix[i+2] > 0 {
			count++
		}
	}
	return count
}

// layout is the film's layout of the show.
var layout = rig.NewLayout(eyeOverLake, towerDeck)

// TestBeamFromItsHead: one beam drawn alone starts at its head's pixel
// and runs along its way: a point 500 m out along it is lit.
func TestBeamFromItsHead(t *testing.T) {
	pane := overhead(t)
	picture := image.NewRGBA(image.Rect(0, 0, 200, 200))
	head := layout.Heads()[0]
	way := rig.Toward(head.Aim, 0)
	far := way.Scale(1000)
	pane.ray(picture, head.Origin, head.Origin.Add(far),
		[3]float64{0, 1, 0}, 1)
	for _, along := range []float64{0, 500} {
		step := way.Scale(along)
		x, y, ahead := pane.onPane(head.Origin.Add(step),
			picture.Rect.Size())
		if !ahead || picture.RGBAAt(int(x), int(y)).G == 0 {
			t.Errorf("%v m along the beam, pixel %v %v is dark", along, x, y)
		}
	}
	// the near miss: a point 500 m out at right angles is dark
	aside := head.Origin.Add(rig.Toward(head.Aim+90, 0).Scale(500))
	x, y, _ := pane.onPane(aside, picture.Rect.Size())
	if picture.RGBAAt(int(x), int(y)).G != 0 {
		t.Errorf("a point off the beam, %v %v, is lit", x, y)
	}
}

// TestRigOnWhenCued: the lasers are drawn when the cues have them on,
// in their colour, and not at all when the rig is dark.
func TestRigOnWhenCued(t *testing.T) {
	pane := overhead(t)
	on, err := cuesOf("rig=1.000\nbeams=10\ncolour=1.000,0.000,0.000\n" +
		"sweep=0.2500\ngate=1.0000\nparity=-1\nfigure=fan\nwhite=0\n" +
		"magenta=0\ngate_left=1.0000\ngate_right=1.0000")
	if err != nil {
		t.Fatal(err)
	}
	picture := image.NewRGBA(image.Rect(0, 0, 200, 200))
	pane.lasers(picture, on, layout, 0)
	if lit(picture) == 0 {
		t.Fatal("lasers cued on drew nothing")
	}
	for i := 0; i < len(picture.Pix); i += 4 {
		if picture.Pix[i+1] != 0 || picture.Pix[i+2] != 0 {
			t.Fatalf("a red cue drew %v", picture.Pix[i:i+4])
		}
	}
	dark := on
	dark.rig = 0
	blank := image.NewRGBA(image.Rect(0, 0, 200, 200))
	drawRig(blank, pane, dark, layout, 0, lampTones{ship: [3]float64{1, 1, 1}})
	marked := lit(blank)
	pane.lasers(blank, dark, layout, 0)
	pane.searchlights(blank, dark, layout)
	pane.strobes(blank, dark, layout, 0)
	pane.floods(blank, dark, layout)
	pane.ship(blank, dark, layout)
	if lit(blank) != marked {
		t.Error("a dark rig drew light past its fixtures' marks")
	}
}

// TestOtherLights: a firing strobe, a lit flood, the ship's beam and the
// searchlights each draw; at nought each draws nothing.
func TestOtherLights(t *testing.T) {
	pane := overhead(t)
	state, _ := cuesOf("white=1\nflood_left=1\noverhead=1\nsearch=1\n" +
		"search_gate=1\nsearch_move=chase\nmagenta=0")
	for name, draw := range map[string]func(*image.RGBA, cues){
		"strobe": func(p *image.RGBA, c cues) { pane.strobes(p, c, layout, 0) },
		"flood":  func(p *image.RGBA, c cues) { pane.floods(p, c, layout) },
		"ship":   func(p *image.RGBA, c cues) { pane.ship(p, c, layout) },
		"search": func(p *image.RGBA, c cues) {
			pane.searchlights(p, c, layout)
		},
	} {
		picture := image.NewRGBA(image.Rect(0, 0, 200, 200))
		draw(picture, state)
		if lit(picture) == 0 {
			t.Errorf("the %s cued on drew nothing", name)
		}
		blank := image.NewRGBA(image.Rect(0, 0, 200, 200))
		draw(blank, cues{white: 0, magenta: 0})
		if lit(blank) != 0 {
			t.Errorf("the %s at nought drew light", name)
		}
	}
}

// TestCuesOf reads the lines and refuses a number that is not one.
func TestCuesOf(t *testing.T) {
	state, err := cuesOf("rig=0.500\ngate=0.2\ngate_left=0.7\n" +
		"figure=cone\nsearch_move=fan\ncolour=0.1,0.2,0.3\nbar=intro")
	if err != nil {
		t.Fatal(err)
	}
	if state.rig != 0.5 || state.figure != "cone" ||
		state.searchMove != "fan" || state.colour != [3]float64{0.1, 0.2,
		0.3} || state.gates() != [2]float64{0.7, 0.2} {
		t.Errorf("read %+v", state)
	}
	for _, wrong := range []string{"rig=lots", "beams=1.5", "colour=1,2",
		"colour=a,b,c"} {
		if _, err := cuesOf(wrong); err == nil ||
			!strings.Contains(err.Error(), strings.Split(wrong, "=")[0]) {
			t.Errorf("%q was read: %v", wrong, err)
		}
	}
}

// TestInStep: a pane at the frame's moment is in step; one a frame and
// more away is refused.
func TestInStep(t *testing.T) {
	if err := inStep(10.01, 10); err != nil {
		t.Error(err)
	}
	if err := inStep(10.1, 10); err == nil {
		t.Error("a pane a tenth of a second off was taken")
	}
}

// TestFilmSize keeps the film's shape and covers the pane.
func TestFilmSize(t *testing.T) {
	for _, pane := range []image.Point{{960, 540}, {1080, 792}} {
		got := filmSize(pane)
		if got.X < pane.X || got.Y < pane.Y ||
			abs(got.X*filmHeight-got.Y*filmWidth) > filmWidth {
			t.Errorf("for %v the film is read at %v", pane, got)
		}
	}
}

// abs is a whole number's size.
func abs(number int) int { return max(number, 0-number) }

// TestTracedSum: a traced pane is the plate at the master plus the
// passes at their cues' levels; brighter when a flood fires than not,
// and the plate alone, at nought master, black.
func TestTracedSum(t *testing.T) {
	pane := traced{size: image.Pt(1, 1), plate: []float64{1e-7, 1e-7, 1e-7},
		passes: map[string][]float64{"back-left": {1e-9, 1e-9, 1e-9},
			"back-right": {1e-9, 0, 0}, "overhead": {0, 0, 1e-6}}}
	whole, one := image.Rect(0, 0, 1, 1), image.Pt(1, 1)
	dark, err := pane.at(cues{master: 0}, whole, one)
	if err != nil {
		t.Fatal(err)
	}
	if dark.RGBAAt(0, 0).R != 0 {
		t.Errorf("no master and no light is %v", dark.RGBAAt(0, 0))
	}
	moonlit, _ := pane.at(cues{master: 1}, whole, one)
	flooded, _ := pane.at(cues{master: 1, floodLeft: 1}, whole, one)
	if flooded.RGBAAt(0, 0).G <= moonlit.RGBAAt(0, 0).G {
		t.Errorf("the flood adds nothing: %v against %v",
			flooded.RGBAAt(0, 0), moonlit.RGBAAt(0, 0))
	}
	broken := pane
	broken.passes = map[string][]float64{"back-left": {1}}
	if _, err := broken.at(cues{floodLeft: 1}, whole, one); err == nil {
		t.Error("a pass of the wrong size was summed")
	}
	if _, err := pane.at(cues{}, image.Rect(0, 0, 2, 2), one); err == nil {
		t.Error("a window off the passes was taken")
	}
}

// TestRendered: the traced panes show the rendered set when the show's
// light falls on it, the drawing otherwise.
func TestRendered(t *testing.T) {
	for state, want := range map[cues]bool{{rig: 1}: false,
		{rig: 0.15}: false, {overhead: 1}: true, {floodRight: 0.5}: true,
		{}: false} {
		if Rendered(state) != want {
			t.Errorf("%+v is rendered %v", state, !want)
		}
	}
	if _, err := tracedOf(t.TempDir(), "north"); err == nil {
		t.Error("passes that are not there were read")
	}
}

// TestMoves: a window holds before a move, is eased between its ends
// during it, and holds at its end after.
func TestMoves(t *testing.T) {
	home := image.Rect(0, 0, 100, 50)
	in := image.Rect(20, 10, 70, 35)
	moves := []move{{from: 10, until: 12, start: home, end: in}}
	if partAt(home, moves, 5) != home || partAt(home, moves, 13) != in {
		t.Error("the window does not hold before and after its move")
	}
	if middle := partAt(home, moves, 11); middle.Min.X != 10 {
		t.Errorf("half way, the window is %v", middle)
	}
}
