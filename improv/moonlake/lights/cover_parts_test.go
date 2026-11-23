package lights

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"strings"
	"testing"
	"testing/iotest"
)

// TestVecSubLeadsFromSecondToFirst checks Sub on a hand-worked pair.
func TestVecSubLeadsFromSecondToFirst(t *testing.T) {
	got := Vec{5, 7, 9}.Sub(Vec{1, 2, 3})
	if got != (Vec{4, 5, 6}) {
		t.Fatalf("Sub=%v", got)
	}
}

// TestConeChordCases checks coneChord on rays whose entry and exit were worked out by hand.
func TestConeChordCases(t *testing.T) {
	cases := []struct {
		name       string
		d, apex    Vec
		slope, low float64
		in, out    float64
		ok         bool
	}{
		{"level ray through the cone", Vec{1, 0, 0}, Vec{0, 0, 10}, 1, 0, 0, 10, true},
		{"level ray under the cut off", Vec{1, 0, 0}, Vec{0, 0, 10}, 1, 1, 0, 0, false},
		{"level ray above a tip below it", Vec{1, 0, 0}, Vec{0, 0, -5}, 1, -10, 0, 0, false},
		{"falling ray with two stretches", Vec{0.6, 0, -0.8}, Vec{0, 0, 10}, 1, -10, 0, 12.5, true},
		{"rising ray skims a cone it is steeper than", Vec{0, 1, 1}, Vec{1, 0, 10}, 1, 0, 0, 4.95, true},
		{"falling ray on the flat side", Vec{0, 1, -1}, Vec{1, 0, 10}, 1, -20, 0, 20, true},
		{"zero direction", Vec{0, 0, 0}, Vec{0, 0, 10}, 1, 0, 0, 0, false},
		{"ray that misses", Vec{1, 0, 0}, Vec{0, 100, 10}, 0.1, 0, 0, 0, false},
	}
	for _, tc := range cases {
		in, out, ok := coneChord(tc.d, tc.apex, tc.slope, tc.low)
		if ok != tc.ok || math.Abs(in-tc.in) > 1e-9 || math.Abs(out-tc.out) > 1e-9 {
			t.Errorf("%s: got (%v, %v, %v), want (%v, %v, %v)", tc.name, in, out, ok, tc.in, tc.out, tc.ok)
		}
	}
}

// TestPlanFixturesRefusesBadGain checks a gain that is not a number is named in the error.
func TestPlanFixturesRefusesBadGain(t *testing.T) {
	config := FixtureConfig{FloodPasses: [2]string{"flood-left", "flood-right"}, FloodGain: math.NaN(), SpotGain: 1}
	_, err := planFixtures(config, fixtureCues{FloodLeft: 1})
	if err == nil || !strings.Contains(err.Error(), "flood-left") {
		t.Fatalf("err=%v", err)
	}
}

// TestPFMDecodeRefusals feeds the decoders headers and bodies that each break one rule.
func TestPFMDecodeRefusals(t *testing.T) {
	if _, _, err := DecodePFMExpected(strings.NewReader("PF\n1 1\n-1\n"), 0, 0); err == nil || !strings.Contains(err.Error(), "expected PFM dimensions") {
		t.Fatalf("zero expected size: %v", err)
	}
	long := "PF\n" + strings.Repeat(" ", 4090) + "\n1 1 -1\n"
	if _, _, err := DecodePFMExpected(strings.NewReader(long), 1, 1); err == nil || !strings.Contains(err.Error(), "4096") {
		t.Fatalf("long header: %v", err)
	}
	if _, _, err := DecodePFM(strings.NewReader("PF\n1 1\n-1")); !errors.Is(err, io.EOF) {
		t.Fatalf("scale with no end: %v", err)
	}
	if _, _, err := DecodePFM(strings.NewReader("PF\n1 1\n-1\r")); !errors.Is(err, io.EOF) {
		t.Fatalf("carriage return with no newline: %v", err)
	}
	broken := errors.New("synthetic read failure")
	body := io.MultiReader(strings.NewReader("PF\n1 1\n-1\n"), iotest.ErrReader(broken))
	if _, _, err := DecodePFM(body); !errors.Is(err, broken) {
		t.Fatalf("pixel read failure: %v", err)
	}
}

// TestPFMEncodeRefusals checks encodePFM refuses a scale that is not a number and reports a writer that fails part way.
func TestPFMEncodeRefusals(t *testing.T) {
	image := Image{Width: 1, Height: 1, Light: []float64{1, 2, 3}}
	if err := encodePFM(&bytes.Buffer{}, image, math.NaN()); err == nil {
		t.Fatal("scale NaN accepted")
	}
	big := Image{Width: 40, Height: 40, Light: make([]float64, 40*40*3)}
	want := errors.New("synthetic write failure")
	if err := encodePFM(failingWriter(t, want), big, -1); !errors.Is(err, want) {
		t.Fatalf("large image write error=%v", err)
	}
}

// TestAddMoonDiscRefusesBadSky checks an invalid sky image is refused before the disc is read.
func TestAddMoonDiscRefusesBadSky(t *testing.T) {
	disc := Image{Width: 1, Height: 1, Light: []float64{1, 1, 1}}
	if err := AddMoonDisc(Image{}, disc, []bool{true}, MoonDiscConfig{}); err == nil {
		t.Fatal("empty sky accepted")
	}
}

// TestHazeOverTheLake checks the lake term: nothing above the lake band, and between 0.35 and 1 times the lake level inside it.
func TestHazeOverTheLake(t *testing.T) {
	if got := Haze(10, 100, -1000, 400, 720, 0); got != 0 {
		t.Fatalf("above the lake: %v", got)
	}
	// At the bottom row the lake term is 0.5 + 0.5*exp(-1.5); the wisps
	// factor lies between 0.35 and 1.
	lake := 0.5 + 0.5*math.Exp(-1.5)
	got := Haze(10, 720, -1000, 400, 720, 0)
	if got < 0.35*lake-1e-9 || got > lake+1e-9 {
		t.Fatalf("bottom row haze %v outside [%v, %v]", got, 0.35*lake, lake)
	}
	// On the shore row itself the lake term is exactly 1.
	if got := Haze(10, 400, -1000, 400, 720, 0); got < 0.35-1e-9 || got > 1+1e-9 {
		t.Fatalf("shore row haze %v outside [0.35, 1]", got)
	}
}

// TestAddStrobesOffFrameAndBlocked checks a strobe that projects off the frame still lights open sky, and that fully blocked pixels get no light.
func TestAddStrobesOffFrameAndBlocked(t *testing.T) {
	f := newPlannerFrame(16, 9)
	far := Toward(109.65, 0).Scale(1000).Add(f.camera.Right.Scale(3000))
	input := StrobeInput{Open: f.open, Near: make([]bool, 16*9), Rim: f.rim, Camera: f.camera,
		Moon: [3]float64{1, 1, 1}, FarTree: far, Azimuth: 109.65, Top: 7, Bottom: -7, Shore: -2,
		White: 1, Magenta: 0}
	sky := f.blank()
	if level := AddStrobes(sky, input); level != 0.5 {
		t.Fatalf("level=%v", level)
	}
	lit := 0.0
	for _, v := range sky.Light {
		lit += v
	}
	if lit <= 0 {
		t.Fatal("open sky stayed dark")
	}
	input.Open = make([]bool, 16*9)
	sky = f.blank()
	AddStrobes(sky, input)
	for i, v := range sky.Light {
		if v != 0 {
			t.Fatalf("blocked pixel channel %d got %v", i, v)
		}
	}
}

// TestFireContributionsGuards checks the early exits and the bad steady timing of FireContributions.
func TestFireContributionsGuards(t *testing.T) {
	config := DefaultFireConfig()
	config.Percussive = true
	for name, cue := range map[string]FireCues{
		"no lit beat":  {Level: 1, Lit: -1, Since: []float64{0.1}},
		"no beat ages": {Level: 1, Lit: 0.1},
	} {
		if got, err := fireContributions(config, cue); got != nil || err != nil {
			t.Errorf("%s: layers=%v err=%v", name, got, err)
		}
	}
	config.SteadyFrom = -1
	if _, err := fireContributions(config, FireCues{Level: 1, Lit: 0.1, Since: []float64{5}}); err == nil || !strings.Contains(err.Error(), "frame timing") {
		t.Fatalf("negative steady start: %v", err)
	}
}

// TestPlanFireDropsAndRefuses checks PlanFire drops a steady layer past the frames, and refuses a percussive layer whose sequence is empty.
func TestPlanFireDropsAndRefuses(t *testing.T) {
	config := DefaultFireConfig()
	got, err := PlanFire(config, FireCues{Level: 1, Since: []float64{1}}, [3]int{5})
	if err != nil || len(got) != 0 {
		t.Fatalf("past the end: layers=%v err=%v", got, err)
	}
	config.Percussive = true
	if _, err := PlanFire(config, FireCues{Level: 1, Lit: 0.1, Since: []float64{0.1}}, [3]int{}); err == nil || !strings.Contains(err.Error(), "holds no frames") {
		t.Fatalf("empty sequence: %v", err)
	}
}

// TestRenderFireRefusals feeds RenderFire a bad size, a bad frame and a frame of the wrong size.
func TestRenderFireRefusals(t *testing.T) {
	if _, err := RenderFire(DefaultFireConfig(), FireInput{}); err == nil || !strings.Contains(err.Error(), "dimensions") {
		t.Fatalf("zero size: %v", err)
	}
	config := DefaultFireConfig()
	cue := FireCues{Level: 1, Since: []float64{0}}
	input := FireInput{Width: 1, Height: 1, Cues: cue}
	input.Frames[FireSteady] = FireFrames{Count: 1, Images: map[int]Image{0: {Width: 1, Height: 1}}}
	if _, err := RenderFire(config, input); err == nil || !strings.Contains(err.Error(), "channel count") {
		t.Fatalf("invalid frame: %v", err)
	}
	input.Frames[FireSteady].Images[0] = Image{Width: 2, Height: 1, Light: make([]float64, 6)}
	if _, err := RenderFire(config, input); err == nil || !strings.Contains(err.Error(), "is 2x1, the frame 1x1") {
		t.Fatalf("wrong size frame: %v", err)
	}
}

// TestFireShoulderEdges checks an infinite pixel is set to top over gain, and dark or not-a-number pixels are left alone.
func TestFireShoulderEdges(t *testing.T) {
	fire := Image{Width: 3, Height: 1, Light: []float64{
		math.Inf(1), 0, 0,
		-1, -1, -1,
		math.NaN(), 0, 0,
	}}
	top, gain := 0.6, 3.0
	fireShoulder(fire, top, gain)
	for c := range 3 {
		if fire.Light[c] != top/gain {
			t.Fatalf("infinite pixel channel %d = %v, want %v", c, fire.Light[c], top/gain)
		}
	}
	for c := 3; c < 6; c++ {
		if fire.Light[c] != -1 {
			t.Fatalf("dark pixel changed: %v", fire.Light[3:6])
		}
	}
	if !math.IsNaN(fire.Light[6]) || fire.Light[7] != 0 {
		t.Fatalf("not-a-number pixel changed: %v", fire.Light[6:9])
	}
}

// TestClassifyBeamMaskOfBlackImage checks an all-zero image gives a mask of the right size with nothing set.
func TestClassifyBeamMaskOfBlackImage(t *testing.T) {
	mask := classifyBeamMask(Image{Width: 2, Height: 2, Light: make([]float64, 12)})
	if len(mask) != 4 {
		t.Fatalf("mask length %d", len(mask))
	}
	for i, v := range mask {
		if v {
			t.Fatalf("pixel %d set", i)
		}
	}
}

// TestPlanFireDropsSpentBurstLayers checks a percussive plan with no gain keeps no layers.
func TestPlanFireDropsSpentBurstLayers(t *testing.T) {
	config := DefaultFireConfig()
	config.Percussive, config.Burst, config.Gain = true, true, 0
	got, err := PlanFire(config, FireCues{Level: 1, Lit: 0.1, Since: []float64{0.1}}, [3]int{50, 5, 5})
	if err != nil || len(got) != 0 {
		t.Fatalf("layers=%v err=%v", got, err)
	}
}

// TestFireContributionsSteadyGuards checks a steady config with no ages gives nothing and one with no frame rate is refused.
func TestFireContributionsSteadyGuards(t *testing.T) {
	config := DefaultFireConfig()
	if got, err := fireContributions(config, FireCues{Level: 1}); got != nil || err != nil {
		t.Fatalf("no ages: layers=%v err=%v", got, err)
	}
	config.FPS = 0
	if _, err := fireContributions(config, FireCues{Level: 1, Since: []float64{1}}); err == nil || !strings.Contains(err.Error(), "frame timing") {
		t.Fatalf("zero frame rate: %v", err)
	}
}

// TestPFMExpectedRefusesReadFailureAfterPixels checks an error from the reader after the pixels is reported, not taken for the end.
func TestPFMExpectedRefusesReadFailureAfterPixels(t *testing.T) {
	broken := errors.New("synthetic read failure")
	data := pfmBytes(t, "PF\n1 1\n-1\n", binary.LittleEndian, []float32{1, 2, 3})
	failing := io.MultiReader(bytes.NewReader(data), iotest.ErrReader(broken))
	_, _, err := DecodePFMExpected(failing, 1, 1)
	if !errors.Is(err, broken) {
		t.Fatalf("err=%v", err)
	}
}

// TestBeamInsideAWideConeKeepsFirstShellColour checks that where the camera sits inside every shell of a wide cone, the light is the white-blue of the first shell alone.
func TestBeamInsideAWideConeKeepsFirstShellColour(t *testing.T) {
	const w, h = 64, 48
	cam := NewCamera(w, h, CameraSettings{Azimuth: 109.65, LookDown: -0.5, GaugeWidth: 70.41, Focal: 150})
	config := UFOConfig{Out: 100, Radius: 900, Apex: 450, Gain: 40, Azimuth: 109.65, WaterZ: -53.6}
	input := UFOInput{Camera: cam, Moon: [3]float64{1, 1, 1}, Cues: UFOCues{Overhead: 1}}
	sky := Image{Width: w, Height: h, Light: make([]float64, w*h*3)}
	front := Image{Width: w, Height: h, Light: make([]float64, w*h*3)}
	addUFOBeam(sky, front, config, input, nil, nil)
	i := sky.At(0, 30)
	r, g, b := sky.Light[i], sky.Light[i+1], sky.Light[i+2]
	if b <= 0 {
		t.Fatalf("pixel got no light: %v %v %v", r, g, b)
	}
	if math.Abs(r/b-0.82) > 1e-9 || math.Abs(g/b-0.9) > 1e-9 {
		t.Fatalf("colour ratio %v : %v : 1, want 0.82 : 0.9 : 1", r/b, g/b)
	}
	for _, v := range front.Light {
		if v != 0 {
			t.Fatal("front got light with no water mask")
		}
	}
}
