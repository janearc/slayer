package lights

import (
	"math"
	"strings"
	"testing"
)

// plannerFrame is a small frame, its camera and masks, shared by the planner tests.
type plannerFrame struct {
	width, height int
	open, close   []bool
	rim           []int
	camera        Camera
}

// newPlannerFrame builds a frame whose top rows are open sky, whose left
// quarter holds close trees, and whose camera looks along the show azimuth.
func newPlannerFrame(width, height int) plannerFrame {
	f := plannerFrame{width: width, height: height, open: make([]bool, width*height),
		close: make([]bool, width*height), rim: make([]int, width)}
	for y := range height {
		for x := range width {
			f.open[y*width+x] = y < height/2
			f.close[y*width+x] = x < width/4
		}
	}
	for x := range f.rim {
		f.rim[x] = height / 2
	}
	f.camera = NewCamera(width, height, CameraSettings{Azimuth: 109.65, LookDown: -0.5, GaugeWidth: 70.41, Focal: 150})
	return f
}

// blank returns an all-zero image the size of the frame.
func (f plannerFrame) blank() Image {
	return Image{Width: f.width, Height: f.height, Light: make([]float64, f.width*f.height*3)}
}

// searchInput returns a lit searchlight input for the frame.
func (f plannerFrame) searchInput(move string) SearchInput {
	return SearchInput{Open: f.open, CloseTrees: f.close, Camera: f.camera, Moon: [3]float64{1, 1, 1},
		FarTree: Toward(109.65, 0).Scale(1000), Azimuth: 109.65, ShoreRow: float64(f.height / 2),
		Level: 1, Gate: 1, Phase: 0.2, MovePhase: 0.4, Move: move}
}

// laserInput returns a lit laser input for the frame.
func (f plannerFrame) laserInput() (LaserConfig, LaserInput) {
	config := LaserConfig{Azimuth: 109.65, EyeOverLake: 53.6, WaterZ: -53.6, Reach: 4000,
		FarTree:    Vec{1727.5, -617.5, 1195.5 - (1076 + 53.6)},
		ShoreLeft:  Vec{547.9, -86.3, -45.6},
		ShoreRight: Vec{473.6, -294.5, -45.6}}
	return config, LaserInput{Open: f.open, CloseTrees: f.close, Rim: f.rim, Camera: f.camera,
		Moon: [3]float64{1, 1, 1}, Seconds: 0.7, ShoreRow: float64(f.height / 2),
		Cues: laserCues{Rig: 1, Gate: 1, GateLeft: -1, GateRight: -1, Sweep: -1, Beams: 4, Parity: -1, Figure: "fan"}}
}

// sameLight reports whether two images hold the same values to within 1e-9.
func sameLight(a, b Image) bool {
	if len(a.Light) != len(b.Light) {
		return false
	}
	for i := range a.Light {
		if math.Abs(a.Light[i]-b.Light[i]) > 1e-9 {
			return false
		}
	}
	return true
}

// TestPlanSearchlightsMatchesDrawingForEveryMove checks that splatting a plan gives the image addSearchlights draws.
func TestPlanSearchlightsMatchesDrawingForEveryMove(t *testing.T) {
	f := newPlannerFrame(32, 18)
	config := SearchConfig{Gain: 0.1, Count: 4}
	for _, move := range []string{"fan", "viewer", "chase", "sweep"} {
		input := f.searchInput(move)
		wantSky, wantFront := f.blank(), f.blank()
		addSearchlights(wantSky, wantFront, config, input)
		plan, err := PlanSearchlights(f.width, f.height, config, input)
		if err != nil {
			t.Fatalf("%s: %v", move, err)
		}
		if len(plan) == 0 {
			t.Fatalf("%s: empty plan", move)
		}
		gotSky, gotFront := f.blank(), f.blank()
		for _, s := range plan {
			if s.SkyOnly != (move != "viewer") {
				t.Fatalf("%s: SkyOnly=%v", move, s.SkyOnly)
			}
			if s.SkyOnly {
				splatSky(gotSky, f.open, s.X, s.Y, s.Sigma, s.Amount, s.Colour)
			} else {
				Splat(gotSky, gotFront, s.X, s.Y, s.Sigma, s.Amount, s.Colour, input.ShoreRow)
			}
		}
		if !sameLight(wantSky, gotSky) || !sameLight(wantFront, gotFront) {
			t.Fatalf("%s: plan and drawing differ", move)
		}
	}
}

// TestPlanSearchlightsQuietAndViewerGlare checks a dim cue plans nothing and a viewer beam without trees still plans.
func TestPlanSearchlightsQuietAndViewerGlare(t *testing.T) {
	f := newPlannerFrame(32, 18)
	config := SearchConfig{Gain: 0.1, Count: 2}
	input := f.searchInput("fan")
	input.Level = 0.005
	if plan, err := PlanSearchlights(f.width, f.height, config, input); err != nil || plan != nil {
		t.Fatalf("dim cue: plan=%v err=%v", plan, err)
	}
	input = f.searchInput("viewer")
	input.CloseTrees = nil
	plan, err := PlanSearchlights(f.width, f.height, config, input)
	if err != nil || len(plan) == 0 {
		t.Fatalf("viewer without trees: %d splats, err=%v", len(plan), err)
	}
}

// TestPlanSearchlightsRefusesBadInput feeds each rule of PlanSearchlights one value that breaks it.
func TestPlanSearchlightsRefusesBadInput(t *testing.T) {
	f := newPlannerFrame(32, 18)
	config := SearchConfig{Gain: 0.1, Count: 2}
	cases := []struct {
		name   string
		mutate func(*SearchConfig, *SearchInput)
		want   string
	}{
		{"short open mask", func(c *SearchConfig, i *SearchInput) { i.Open = i.Open[1:] }, "dimensions"},
		{"short close mask", func(c *SearchConfig, i *SearchInput) { i.CloseTrees = i.CloseTrees[1:] }, "dimensions"},
		{"count too big", func(c *SearchConfig, i *SearchInput) { c.Count = 257 }, "count"},
		{"nan gain", func(c *SearchConfig, i *SearchInput) { c.Gain = math.NaN() }, "nonfinite"},
		{"infinite phase", func(c *SearchConfig, i *SearchInput) { i.Phase = math.Inf(1) }, "nonfinite"},
		{"nan camera axis", func(c *SearchConfig, i *SearchInput) { i.Camera.Right.X = math.NaN() }, "camera vector"},
		{"huge camera axis", func(c *SearchConfig, i *SearchInput) { i.Camera.Up.Z = 1e8 }, "camera vector"},
		{"camera of another size", func(c *SearchConfig, i *SearchInput) { i.Camera.Width = 7 }, "camera or phase"},
		{"negative move phase", func(c *SearchConfig, i *SearchInput) { i.MovePhase = -0.1 }, "camera or phase"},
		{"move phase over a million", func(c *SearchConfig, i *SearchInput) { i.MovePhase = 2e6 }, "camera or phase"},
	}
	for _, tc := range cases {
		c, input := config, f.searchInput("fan")
		tc.mutate(&c, &input)
		_, err := PlanSearchlights(f.width, f.height, c, input)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err=%v, want it to mention %q", tc.name, err, tc.want)
		}
	}
}

// TestPlanLasersMatchesDrawing checks the plan and addLasers agree over several cue sets.
func TestPlanLasersMatchesDrawing(t *testing.T) {
	f := newPlannerFrame(32, 18)
	cases := map[string]func(*laserCues){
		"default":      func(c *laserCues) {},
		"sweep":        func(c *laserCues) { c.Sweep = 0.3 },
		"half rig":     func(c *laserCues) { c.Rig = 0.5 },
		"own colour":   func(c *laserCues) { c.HasColour, c.Colour = true, [3]float64{1, 0.2, 0.4} },
		"left only":    func(c *laserCues) { c.GateLeft, c.GateRight = 1, 0 },
		"right only":   func(c *laserCues) { c.GateLeft, c.GateRight = 0, 1 },
		"parity even":  func(c *laserCues) { c.Parity, c.Figure, c.Beams = 0, "fan64", 64 },
		"parity odd":   func(c *laserCues) { c.Parity, c.Figure, c.Beams = 1, "fan64", 64 },
		"wide fan":     func(c *laserCues) { c.Beams, c.Figure = 9, "fan" },
		"unset rig":    func(c *laserCues) { c.Rig = -1 },
		"phase figure": func(c *laserCues) { c.FigurePhase = 0.37 },
	}
	for name, mutate := range cases {
		config, input := f.laserInput()
		mutate(&input.Cues)
		wantSky, wantFront := f.blank(), f.blank()
		addLasers(wantSky, wantFront, config, input)
		plan, err := PlanLasers(f.width, f.height, config, input)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		gotSky, gotFront := f.blank(), f.blank()
		for _, s := range plan {
			Splat(gotSky, gotFront, s.X, s.Y, s.Sigma, s.Amount, s.Colour, input.ShoreRow)
		}
		if !sameLight(wantSky, gotSky) || !sameLight(wantFront, gotFront) {
			t.Errorf("%s: plan and drawing differ", name)
		}
		if len(plan) == 0 {
			t.Errorf("%s: empty plan", name)
		}
	}
}

// TestPlanLasersWithoutTreeMasks checks a plan with no close tree mask still draws beams.
func TestPlanLasersWithoutTreeMasks(t *testing.T) {
	f := newPlannerFrame(32, 18)
	config, input := f.laserInput()
	input.CloseTrees = nil
	plan, err := PlanLasers(f.width, f.height, config, input)
	if err != nil || len(plan) == 0 {
		t.Fatalf("%d splats, err=%v", len(plan), err)
	}
}

// TestPlanLasersPlansNothingWhenOffOrShut checks that no rig or two shut gates give an empty plan.
func TestPlanLasersPlansNothingWhenOffOrShut(t *testing.T) {
	f := newPlannerFrame(32, 18)
	config, input := f.laserInput()
	input.Cues.Rig = 0
	if plan, err := PlanLasers(f.width, f.height, config, input); plan != nil || err != nil {
		t.Fatalf("rig off: plan=%v err=%v", plan, err)
	}
	config, input = f.laserInput()
	input.Cues.Gate, input.Cues.GateLeft, input.Cues.GateRight = 0, 0.005, 0
	if plan, err := PlanLasers(f.width, f.height, config, input); plan != nil || err != nil {
		t.Fatalf("gates shut: plan=%v err=%v", plan, err)
	}
}

// TestPlanLasersScalesColourByRig checks that a beam splat is the default green times Rig, the gate and the beam weight 1.8.
func TestPlanLasersScalesColourByRig(t *testing.T) {
	f := newPlannerFrame(32, 18)
	config, input := f.laserInput()
	input.Cues.Rig = 0.5
	input.Cues.Gate = 0.5
	input.Cues.Beams = 1
	input.Cues.Figure = "beam"
	plan, err := PlanLasers(f.width, f.height, config, input)
	if err != nil || len(plan) == 0 {
		t.Fatalf("%d splats, err=%v", len(plan), err)
	}
	want := [3]float64{0.036, 0.45, 0.054}
	found := false
	for _, s := range plan {
		if s.Colour[1] > 0.45+1e-9 {
			t.Fatalf("green %v exceeds rig times gate", s.Colour[1])
		}
		if math.Abs(s.Colour[0]-want[0]) < 1e-9 && math.Abs(s.Colour[1]-want[1]) < 1e-9 && math.Abs(s.Colour[2]-want[2]) < 1e-9 {
			found = true
		}
	}
	if !found {
		t.Fatalf("no splat has colour %v", want)
	}
}

// TestPlanLasersRefusesBadInput feeds each rule of PlanLasers one value that breaks it.
func TestPlanLasersRefusesBadInput(t *testing.T) {
	f := newPlannerFrame(32, 18)
	cases := []struct {
		name   string
		mutate func(*LaserConfig, *LaserInput)
		want   string
	}{
		{"short rim", func(c *LaserConfig, i *LaserInput) { i.Rim = i.Rim[1:] }, "dimensions"},
		{"short close mask", func(c *LaserConfig, i *LaserInput) { i.CloseTrees = i.CloseTrees[1:] }, "dimensions"},
		{"too many beams", func(c *LaserConfig, i *LaserInput) { i.Cues.Beams = 257 }, "beam count"},
		{"reach too long", func(c *LaserConfig, i *LaserInput) { c.Reach = 2e7 }, "reach"},
		{"nan seconds", func(c *LaserConfig, i *LaserInput) { i.Seconds = math.NaN() }, "parameter"},
		{"huge rig deck", func(c *LaserConfig, i *LaserInput) { c.RigDeck = 1e8 }, "parameter"},
		{"nan far tree", func(c *LaserConfig, i *LaserInput) { c.FarTree.Y = math.NaN() }, "vector"},
		{"infinite camera axis", func(c *LaserConfig, i *LaserInput) { i.Camera.Forward.Z = math.Inf(-1) }, "vector"},
		{"nan colour", func(c *LaserConfig, i *LaserInput) { i.Cues.Colour[2] = math.NaN() }, "colour"},
	}
	for _, tc := range cases {
		c, input := f.laserInput()
		tc.mutate(&c, &input)
		_, err := PlanLasers(f.width, f.height, c, input)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err=%v, want it to mention %q", tc.name, err, tc.want)
		}
	}
}

// TestPlanSearchlightsBehindAndBesideTheCamera checks beams that never reach the frame plan nothing, as addSearchlights draws nothing.
func TestPlanSearchlightsBehindAndBesideTheCamera(t *testing.T) {
	f := newPlannerFrame(32, 18)
	config := SearchConfig{Gain: 0.1, Count: 4}
	behind := f.searchInput("sweep")
	behind.Camera.Forward = behind.Camera.Forward.Scale(-1)
	beside := f.searchInput("viewer")
	beside.Camera = NewCamera(f.width, f.height, CameraSettings{Azimuth: 109.65 + 40, LookDown: -0.5, GaugeWidth: 70.41, Focal: 150})
	for name, input := range map[string]SearchInput{"behind": behind, "beside": beside} {
		sky, front := f.blank(), f.blank()
		addSearchlights(sky, front, config, input)
		for i := range sky.Light {
			if sky.Light[i] != 0 || front.Light[i] != 0 {
				t.Fatalf("%s: drawing lit channel %d", name, i)
			}
		}
		plan, err := PlanSearchlights(f.width, f.height, config, input)
		if err != nil || len(plan) != 0 {
			t.Fatalf("%s: plan=%d splats err=%v", name, len(plan), err)
		}
	}
}

// TestPlanSearchlightsRefusesSplatThatIsNotFinite checks that a moon too bright to add up gives an error, not a splat.
func TestPlanSearchlightsRefusesSplatThatIsNotFinite(t *testing.T) {
	f := newPlannerFrame(32, 18)
	config := SearchConfig{Gain: 0.1, Count: 2}
	bright := [3]float64{1e308, 1e308, 1e308}
	closed := make([]bool, len(f.open))
	for name, mutate := range map[string]func(*SearchInput){
		"fan":                func(i *SearchInput) { i.Move = "fan"; i.Moon = bright },
		"viewer glare":       func(i *SearchInput) { i.Move = "viewer"; i.Moon = bright },
		"viewer with no sky": func(i *SearchInput) { i.Move = "viewer"; i.Moon = bright; i.Open = closed },
	} {
		input := f.searchInput("fan")
		mutate(&input)
		if _, err := PlanSearchlights(f.width, f.height, config, input); err == nil || !strings.Contains(err.Error(), "invalid splat") {
			t.Errorf("%s: err=%v", name, err)
		}
	}
}

// TestPlanLasersHidesBeamsBehindTheRim checks the plan still matches drawing when every row is under the rim.
func TestPlanLasersHidesBeamsBehindTheRim(t *testing.T) {
	f := newPlannerFrame(32, 18)
	config, input := f.laserInput()
	for x := range input.Rim {
		input.Rim[x] = f.height
	}
	wantSky, wantFront := f.blank(), f.blank()
	addLasers(wantSky, wantFront, config, input)
	plan, err := PlanLasers(f.width, f.height, config, input)
	if err != nil {
		t.Fatal(err)
	}
	gotSky, gotFront := f.blank(), f.blank()
	for _, s := range plan {
		Splat(gotSky, gotFront, s.X, s.Y, s.Sigma, s.Amount, s.Colour, input.ShoreRow)
	}
	if !sameLight(wantSky, gotSky) || !sameLight(wantFront, gotFront) {
		t.Fatal("plan and drawing differ")
	}
}
