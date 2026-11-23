package lights

import (
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func rendererAssets() Assets {
	return Assets{
		Plate: Image{1, 3, []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9}},
		Mask:  Image{1, 3, []float64{1, 1, 1, 1, 1, 1, 0, 0, 0}},
	}
}

func TestRendererPrecomputesMasksAndOwnsAssets(t *testing.T) {
	assets := rendererAssets()
	trees := Image{1, 3, []float64{0, 0, 0, 1, 1, 1, 0, 0, 0}}
	assets.Trees = &trees
	assets.Passes = map[string]Image{"unused": assets.Plate}
	frames := [3]FireFrames{{Count: 2, Images: map[int]Image{1: assets.Plate}}}
	assets.Fire = &frames
	r, err := NewRenderer(DefaultConfig(), assets)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.open, []bool{true, true, false}) || !reflect.DeepEqual(r.rim, []int{2}) || !reflect.DeepEqual(r.near, []bool{false, false, true}) || !reflect.DeepEqual(r.closeTrees, []bool{false, true, false}) {
		t.Fatalf("open=%v rim=%v near=%v close=%v", r.open, r.rim, r.near, r.closeTrees)
	}
	want := float64(float32(0.1))
	assets.Plate.Light[0] = 99
	assets.Mask.Light[0] = 0
	trees.Light[3] = 0
	delete(assets.Passes, "unused")
	delete(frames[0].Images, 1)
	if r.assets.Plate.Light[0] != want || !r.open[0] || !r.closeTrees[1] || r.assets.Passes["unused"].Light[0] == 99 || r.assets.Fire[0].Images[1].Light[0] == 99 {
		t.Fatal("constructor retained caller-owned data")
	}
	if _, ok := r.assets.Fire[0].Images[1]; !ok {
		t.Fatal("constructor retained fire map")
	}
}

func TestRendererBaseRoundTripAndIndependentResults(t *testing.T) {
	config := DefaultConfig()
	config.SkyGlowGain = 0
	r, err := NewRenderer(config, rendererAssets())
	if err != nil {
		t.Fatal(err)
	}
	cue := CompleteCues{Master: 0.33335, Rig: 0, White: 0, Magenta: 0}
	request := Request{Seconds: 0, Cues: &cue}
	got, err := r.Render(request)
	if err != nil {
		t.Fatal(err)
	}
	for i, value := range rendererAssets().Plate.Light {
		narrow := float32(value)
		scaled := float64(narrow) * 0.3333
		want := float64(float32(scaled))
		if got.Sky.Light[i] != want || got.Front.Light[i] != 0 {
			t.Fatalf("channel %d: sky=%v want=%v front=%v", i, got.Sky.Light[i], want, got.Front.Light[i])
		}
	}
	if got.Strobe != 0 || cue.Master != 0.33335 {
		t.Fatal("strobe or input cues changed")
	}
	want := slices.Clone(got.Sky.Light)
	got.Sky.Light[0], got.Front.Light[0] = 999, 999
	again, err := r.Render(request)
	if err != nil || !slices.Equal(again.Sky.Light, want) || again.Front.Light[0] != 0 {
		t.Fatalf("result storage leaked: %v", err)
	}
}

func TestRendererDemandNeededPassesAndNamedErrors(t *testing.T) {
	assets := rendererAssets()
	assets.Passes = map[string]Image{"broken-unused": {}}
	r, err := NewRenderer(DefaultConfig(), assets)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.pass("flood-left"); err == nil || !strings.Contains(err.Error(), `pass "flood-left": missing`) {
		t.Fatalf("missing pass: %v", err)
	}
	if _, err := r.pass("broken-unused"); err == nil || !strings.Contains(err.Error(), `pass "broken-unused"`) {
		t.Fatalf("invalid pass: %v", err)
	}
	config := DefaultConfig()
	config.MaxFireFrames = 1
	frames := [3]FireFrames{{Count: 100, Images: map[int]Image{0: assets.Plate, 99: assets.Plate}}}
	assets.Fire = &frames
	if _, err := NewRenderer(config, assets); err == nil || !strings.Contains(err.Error(), "maximum 1") {
		t.Fatalf("unbounded fire: %v", err)
	}
}

func TestRendererFireNeedsOnlySelectedFramesAndUsesFloatBoundary(t *testing.T) {
	assets := rendererAssets()
	frames := [3]FireFrames{{Count: 100, Images: map[int]Image{1: assets.Plate, 99: {}}}}
	assets.Fire = &frames
	r, err := NewRenderer(DefaultConfig(), assets)
	if err != nil {
		t.Fatal(err)
	}
	config := DefaultFireConfig()
	config.Stops = 1
	cue := FireCues{Level: 0.3, Since: []float64{0.0209}}
	got, err := r.renderFire(config, cue)
	if err != nil {
		t.Fatal(err)
	}
	pass, err := rendererImage("expected", assets.Plate, 1, 3)
	if err != nil {
		t.Fatal(err)
	}
	want := cloneRendererImage(pass)
	amount := config.Gain * cue.Level * math.Exp(-cue.Since[0]/config.Decay)
	for i := range want.Light {
		want.Light[i] = amount * pass.Light[i]
	}
	fireShoulder(want, config.Top, math.Exp2(config.Stops))
	if !slices.Equal(got.Light, want.Light) {
		t.Fatalf("fire=%v want=%v", got.Light, want.Light)
	}
	cue.Since[0] = 0
	if _, err := r.renderFire(config, cue); err == nil || !strings.Contains(err.Error(), "fire steady frame 0: missing") {
		t.Fatalf("missing selected frame: %v", err)
	}
	cue.Since[0] = 99.0 / 24
	if _, err := r.renderFire(config, cue); err == nil || !strings.Contains(err.Error(), "fire steady frame 99") {
		t.Fatalf("invalid selected frame: %v", err)
	}
	cue.Since[0] = 100.0 / 24
	if _, err := r.renderFire(config, cue); err != nil {
		t.Fatalf("expired fire required a frame: %v", err)
	}
}

func TestRendererPercussiveFrameClampingAndUnusedSteady(t *testing.T) {
	config := DefaultConfig()
	config.Fire.Percussive = true
	assets := rendererAssets()
	frames := [3]FireFrames{{Count: 50, Images: map[int]Image{}}, {Count: 2, Images: map[int]Image{0: assets.Plate}}}
	assets.Fire = &frames
	r, err := NewRenderer(config, assets)
	if err != nil {
		t.Fatal(err)
	}
	cue := FireCues{Level: 1, Lit: 0, Since: []float64{0}}
	if _, err := r.renderFire(config.Fire, cue); err != nil {
		t.Fatalf("zero-amount steady required: %v", err)
	}
	cue.Lit, cue.Since[0] = 1, 1
	if _, err := r.renderFire(config.Fire, cue); err == nil || !strings.Contains(err.Error(), "fire steady frame 49: missing") {
		t.Fatalf("steady clamp: %v", err)
	}
	cue.Lit = -1
	if _, err := r.renderFire(config.Fire, cue); err != nil {
		t.Fatalf("unlit fire required frames: %v", err)
	}
}

func TestRendererStrobeReportsBeforeProjectionAndLightsClosedPixels(t *testing.T) {
	sky := Image{1, 2, make([]float64, 6)}
	camera := NewCamera(1, 2, CameraSettings{109.65, -0.5, 70.41, 150})
	input := StrobeInput{Open: []bool{false, false}, Near: []bool{true, true}, Rim: []int{1},
		Camera: camera, Moon: [3]float64{1, 1, 1}, FarTree: Toward(109.65, 0).Scale(1000),
		Azimuth: 109.65, Top: 7, Bottom: -7, Shore: -2, White: 1, Magenta: 0}
	if level := AddStrobes(sky, input); level != 0.5 || !slices.ContainsFunc(sky.Light, func(v float64) bool { return v > 0 }) {
		t.Fatalf("closed-pixel strobe: level=%v light=%v", level, sky.Light)
	}
	clear(sky.Light)
	input.Camera.Forward = input.Camera.Forward.Scale(-1)
	input.Magenta = 1
	if level := AddStrobes(sky, input); level != 1 || slices.ContainsFunc(sky.Light, func(v float64) bool { return v != 0 }) {
		t.Fatalf("behind-camera strobe: level=%v light=%v", level, sky.Light)
	}
	input.White, input.Magenta = -1, -1
	for _, seconds := range []float64{0, 0.04166, 0.04167, 1.25} {
		input.Seconds = seconds
		frame := int(math.Round(seconds * 12))
		want := 0.0
		for _, seed := range []uint64{1, 2} {
			if Lattice(int64(frame), 0, seed) < 0.28 {
				want += 0.5
			}
		}
		if got := AddStrobes(sky, input); got != want {
			t.Fatalf("seconds %v: strobe=%v want=%v", seconds, got, want)
		}
	}
}

func TestRendererLaserApertureSurvivesZeroBeamsAndBlackColour(t *testing.T) {
	sky := Image{8, 8, make([]float64, 8*8*3)}
	front := Image{8, 8, make([]float64, len(sky.Light))}
	camera := NewCamera(8, 8, CameraSettings{109.65, -0.5, 70.41, 150})
	origin := camera.Forward.Scale(1000)
	input := LaserInput{Camera: camera, Moon: [3]float64{1, 1, 1}, ShoreRow: 8}
	addLaserAperture(sky, front, input, origin, [3]float64{1, 1, 1})
	if !slices.ContainsFunc(sky.Light, func(v float64) bool { return v > 0 }) {
		t.Fatal("aperture did not light")
	}
	clear(sky.Light)
	addLaserAperture(sky, front, input, origin, [3]float64{})
	if slices.ContainsFunc(sky.Light, func(v float64) bool { return v != 0 }) {
		t.Fatal("black aperture emitted light")
	}
	ways, _, _ := figureWays("fan", 0, 0, 10, 0, 0)
	if len(ways) != 0 {
		t.Fatal("zero fan unexpectedly has beams")
	}
}

func TestRendererCachedMasksPreserveDistinctThresholds(t *testing.T) {
	assets := rendererAssets()
	mask := Image{1, 3, []float64{1, 0, 0, 0.2, 0.2, 0.2, 0, 0, 0}}
	assets.Trees, assets.Water = &mask, &mask
	r, err := NewRenderer(DefaultConfig(), assets)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.closeTrees, []bool{true, true, false}) || !reflect.DeepEqual(r.beamTrees, []bool{true, false, false}) || !reflect.DeepEqual(r.beamWater, r.beamTrees) || !r.hazeEnabled {
		t.Fatalf("close=%v UFO trees=%v water=%v haze=%v", r.closeTrees, r.beamTrees, r.beamWater, r.hazeEnabled)
	}
	front, trees, input := hazeFixture()
	want := cloneRendererImage(front)
	AddBackHaze(want, trees, input)
	classified, err := coveredBy(trees)
	if err != nil {
		t.Fatal(err)
	}
	addBackHaze(front, classified, input)
	if !slices.Equal(front.Light, want.Light) {
		t.Fatal("cached haze classification changed output")
	}
	sky, front, config, ufo := ufoFixture("golden")
	wantSky, wantFront := cloneRendererImage(sky), cloneRendererImage(front)
	AddUFOBeam(wantSky, wantFront, config, ufo)
	addUFOBeam(sky, front, config, ufo, classifyBeamMask(ufo.Trees), classifyBeamMask(ufo.Water))
	if !slices.Equal(sky.Light, wantSky.Light) || !slices.Equal(front.Light, wantFront.Light) {
		t.Fatal("cached UFO classification changed output")
	}
}

func TestRendererConstructorRejectsUnsafeGeometryAndDimensions(t *testing.T) {
	for _, mutate := range []func(*Config){
		func(c *Config) { c.Search.Count = 0 },
		func(c *Config) { c.Search.Count = -1 },
		func(c *Config) { c.Focal = 0 },
		func(c *Config) { c.Focal = math.MaxFloat64 },
		func(c *Config) { c.GaugeHeight = math.SmallestNonzeroFloat64 },
		func(c *Config) { c.LookDown = 90 },
	} {
		config := DefaultConfig()
		mutate(&config)
		if _, err := NewRenderer(config, rendererAssets()); err == nil {
			t.Fatalf("accepted unsafe config: %+v", config)
		}
	}
	for _, size := range [][2]int{{0, 3}, {-1, 3}, {1, 0}, {2, 3}, {int(^uint(0) >> 1), 3}} {
		assets := rendererAssets()
		assets.Mask.Width, assets.Mask.Height = size[0], size[1]
		if _, err := NewRenderer(DefaultConfig(), assets); err == nil {
			t.Fatalf("accepted mask dimensions %v", size)
		}
	}
	r, err := NewRenderer(DefaultConfig(), rendererAssets())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Render(Request{Cues: &CompleteCues{Beams: -1}}); err == nil {
		t.Fatal("negative beams accepted")
	}
	bad := Image{2, 1, make([]float64, 6)}
	if _, err := r.Render(Request{Water: &bad}); err == nil {
		t.Fatal("mismatched dynamic water accepted")
	}
}

func TestRendererCompleteCuesReplaceDefaults(t *testing.T) {
	r, err := NewRenderer(DefaultConfig(), rendererAssets())
	if err != nil {
		t.Fatal(err)
	}
	defaults := r.defaultCues()
	if defaults.Master != 1 || defaults.Rig != -1 || defaults.White != -1 {
		t.Fatal("nil-cue defaults changed")
	}
	got, err := r.Render(Request{Cues: &CompleteCues{}})
	if err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(got.Sky.Light, func(v float64) bool { return v != 0 }) || got.Strobe != 0 {
		t.Fatal("complete zero-valued cues were merged with defaults")
	}
}

func TestRendererRejectsNamedInvalidInputs(t *testing.T) {
	for _, name := range []string{"plate", "mask", "moon disc", "water mask", "tree mask"} {
		assets := rendererAssets()
		bad := Image{}
		switch name {
		case "plate":
			assets.Plate = bad
		case "mask":
			assets.Mask = bad
		case "moon disc":
			assets.Disc = &bad
		case "water mask":
			assets.Water = &bad
		case "tree mask":
			assets.Trees = &bad
		}
		if _, err := NewRenderer(DefaultConfig(), assets); err == nil || !strings.Contains(err.Error(), name) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	r, err := NewRenderer(DefaultConfig(), rendererAssets())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Render(Request{Seconds: math.NaN()}); err == nil || !strings.Contains(err.Error(), "seconds") {
		t.Fatalf("seconds: %v", err)
	}
	if _, err := r.Render(Request{Cues: &CompleteCues{Rig: math.Inf(1)}}); err == nil || !strings.Contains(err.Error(), "cues") {
		t.Fatalf("cues: %v", err)
	}
	bad := Image{}
	if _, err := r.Render(Request{Water: &bad}); err == nil || !strings.Contains(err.Error(), "water mask") {
		t.Fatalf("water: %v", err)
	}
}
