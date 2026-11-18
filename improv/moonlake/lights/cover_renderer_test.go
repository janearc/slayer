package lights

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"strings"
	"testing"
)

// fakeAccelerator is a BaseAccelerator that returns copies of its inputs, so a
// test can see which calls Render makes, in what order, and what it does with
// the answers. A call is named by its stage and its count, such as "fixtures#2".
type fakeAccelerator struct {
	calls   []string
	counts  map[string]int
	failAt  string
	badAt   string
	level   float64
	stops   float64
	layers  [][]FixtureLayer
	shift   float64
	master  float64
	fireOut float64
}

// key records a call of stage and returns its name with the count.
func (f *fakeAccelerator) key(stage string) string {
	if f.counts == nil {
		f.counts = map[string]int{}
	}
	f.counts[stage]++
	name := stage + "#" + string(rune('0'+f.counts[stage]))
	f.calls = append(f.calls, name)
	return name
}

// step returns a copy of img, an error or an invalid image, as the test asked.
func (f *fakeAccelerator) step(name string, img Image) (Image, error) {
	if f.failAt == name {
		return Image{}, errors.New("synthetic " + name)
	}
	if f.badAt == name {
		return Image{}, nil
	}
	return cloneRendererImage(img), nil
}

// pair is step for the two image stages: ":front" in badAt spoils the second.
func (f *fakeAccelerator) pair(name string, sky, front Image) (Image, Image, error) {
	a, err := f.step(name, sky)
	if err != nil {
		return Image{}, Image{}, err
	}
	b := cloneRendererImage(front)
	if f.badAt == name+":front" {
		b = Image{}
	}
	return a, b, nil
}

// Base returns a copy of the plate and records the master, shift and moon it was given.
func (f *fakeAccelerator) Base(plate Image, disc *Image, open []bool, master, shift, moon float64) (Image, error) {
	f.shift, f.master = shift, master
	return f.step(f.key("base"), plate)
}

// Fixtures returns a copy of sky and records the layers it was asked to add.
func (f *fakeAccelerator) Fixtures(sky Image, layers []FixtureLayer, passes map[string]Image) (Image, error) {
	f.layers = append(f.layers, layers)
	return f.step(f.key("fixtures"), sky)
}

// Fire returns an image of the input size filled with the level the test chose, and records the stops.
func (f *fakeAccelerator) Fire(config FireConfig, input FireInput) (Image, error) {
	name := f.key("fire")
	f.stops = config.Stops
	if f.failAt == name {
		return Image{}, errors.New("synthetic " + name)
	}
	if f.badAt == name {
		return Image{}, nil
	}
	out := Image{Width: input.Width, Height: input.Height, Light: make([]float64, input.Width*input.Height*3)}
	for i := range out.Light {
		out.Light[i] = f.fireOut
	}
	return out, nil
}

// SkyGlow returns a copy of sky.
func (f *fakeAccelerator) SkyGlow(sky Image, input SkyGlowInput) (Image, error) {
	return f.step(f.key("skyglow"), sky)
}

// Strobes returns a copy of sky and the strobe level the test chose.
func (f *fakeAccelerator) Strobes(sky Image, input StrobeInput) (Image, float64, error) {
	name := f.key("strobes")
	out, err := f.step(name, sky)
	level := f.level
	if f.badAt == name+":level" {
		level = math.NaN()
	}
	return out, level, err
}

// Searchlights returns copies of sky and front.
func (f *fakeAccelerator) Searchlights(sky, front Image, config SearchConfig, input SearchInput) (Image, Image, error) {
	return f.pair(f.key("searchlights"), sky, front)
}

// Lasers returns copies of sky and front.
func (f *fakeAccelerator) Lasers(sky, front Image, config LaserConfig, input LaserInput) (Image, Image, error) {
	return f.pair(f.key("lasers"), sky, front)
}

// BackHaze returns a copy of front.
func (f *fakeAccelerator) BackHaze(front Image, close []bool, input BackHazeInput) (Image, error) {
	return f.step(f.key("backhaze"), front)
}

// Beam returns copies of sky and front.
func (f *fakeAccelerator) Beam(sky, front Image, config UFOConfig, input UFOInput, trees, water []bool) (Image, Image, error) {
	return f.pair(f.key("beam"), sky, front)
}

// rigAssets builds a 32 by 18 set of assets with a sky above row 9, a moon
// disc, a water mask, a tree mask, every pass a full show asks for, and one
// steady fire frame.
func rigAssets() Assets {
	const w, h = 32, 18
	fill := func(value func(x, y int) float64) Image {
		img := Image{Width: w, Height: h, Light: make([]float64, w*h*3)}
		for y := range h {
			for x := range w {
				for c := range 3 {
					img.Light[img.At(x, y)+c] = value(x, y)
				}
			}
		}
		return img
	}
	plate := fill(func(x, y int) float64 { return 0.1 + 0.01*float64(y%5) })
	mask := fill(func(x, y int) float64 {
		if y < 9 {
			return 1
		}
		return 0
	})
	disc := fill(func(x, y int) float64 {
		if x == 16 && y == 3 {
			return 1
		}
		return 0
	})
	water := fill(func(x, y int) float64 {
		if y >= 13 {
			return 1
		}
		return 0
	})
	trees := fill(func(x, y int) float64 {
		if x < 8 && y >= 6 {
			return 1
		}
		return 0
	})
	passes := map[string]Image{}
	for _, name := range []string{"overhead", "flood-left", "flood-right", "spot-left-0", "spot-left-1", "spot-left-2", "spot-right-0", "spot-right-1", "spot-right-2"} {
		passes[name] = fill(func(x, y int) float64 { return 0.05 })
	}
	frames := [3]FireFrames{{Count: 4, Images: map[int]Image{0: fill(func(x, y int) float64 { return 0.02 })}}}
	return Assets{Plate: plate, Mask: mask, Disc: &disc, Water: &water, Trees: &trees, Passes: passes, Fire: &frames}
}

// rigCues returns cues that light every stage of the renderer.
func rigCues() *CompleteCues {
	return &CompleteCues{Rig: 1, Beams: 3, Gate: 1, GateLeft: -1, GateRight: -1, Sweep: -1, Parity: -1,
		Figure: "fan", FloodLeft: 1, FloodRight: 1, Spots: 1, Search: 1, SearchGate: 1, SearchMove: "fan",
		Master: 1, Overhead: 0.5, Fire: 1, FireSince: []float64{0}, FireLit: -1, Moon: 1, MoonAlt: 3}
}

// TestRenderLightsEveryStageOnTheCPU checks a full show renders with the sizes, working set and light the cues call for.
func TestRenderLightsEveryStageOnTheCPU(t *testing.T) {
	assets := rigAssets()
	r, err := NewRenderer(DefaultConfig(), assets)
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.Render(Request{Seconds: 0.5, Cues: rigCues()})
	if err != nil {
		t.Fatal(err)
	}
	if got.Sky.Width != 32 || got.Sky.Height != 18 || got.Front.Width != 32 || got.Front.Height != 18 {
		t.Fatalf("sizes %dx%d and %dx%d", got.Sky.Width, got.Sky.Height, got.Front.Width, got.Front.Height)
	}
	// One 32x18 frame is 32*18*24 = 13824 bytes; transient is twice that plus one frame.
	if want := (FireWorkingSet{Frames: 1, SelectedBytes: 13824, TransientBytes: 41472}); got.FireWorkingSet != want {
		t.Fatalf("working set %+v, want %+v", got.FireWorkingSet, want)
	}
	brighter, frontLit := 0, 0
	for i, plate := range assets.Plate.Light {
		if got.Sky.Light[i] < float64(float32(plate))-1e-9 {
			t.Fatalf("channel %d: sky %v is under the plate %v", i, got.Sky.Light[i], plate)
		}
		if got.Sky.Light[i] > float64(float32(plate))+1e-6 {
			brighter++
		}
		if got.Front.Light[i] < 0 {
			t.Fatalf("channel %d: front %v is negative", i, got.Front.Light[i])
		}
		if got.Front.Light[i] > 0 {
			frontLit++
		}
	}
	if brighter == 0 || frontLit == 0 {
		t.Fatalf("lights added nothing: %d sky channels brighter, %d front channels lit", brighter, frontLit)
	}
}

// TestRenderMoonDiscFollowsTheMoonCue checks the disc adds its light to open sky at the cue's level and not to closed rows.
func TestRenderMoonDiscFollowsTheMoonCue(t *testing.T) {
	const w, h = 4, 4
	flat := func(value float64) Image {
		img := Image{Width: w, Height: h, Light: make([]float64, w*h*3)}
		for i := range img.Light {
			img.Light[i] = value
		}
		return img
	}
	plate, mask, disc := flat(0.25), flat(0), flat(0)
	for i := 0; i < 2*w*3; i++ {
		mask.Light[i] = 1
	}
	for c := range 3 {
		disc.Light[disc.At(1, 0)+c] = 1
		disc.Light[disc.At(1, 3)+c] = 1
	}
	config := DefaultConfig()
	config.SkyGlowGain = 0
	r, err := NewRenderer(config, Assets{Plate: plate, Mask: mask, Disc: &disc})
	if err != nil {
		t.Fatal(err)
	}
	cue := &CompleteCues{Rig: 0, Beams: 1, Gate: 1, GateLeft: -1, GateRight: -1, Sweep: -1, Parity: -1,
		Figure: "fan", Master: 1, Moon: 0.5, MoonAlt: config.MoonDiscAltitude, SearchMove: "fan"}
	got, err := r.Render(Request{Cues: cue})
	if err != nil {
		t.Fatal(err)
	}
	for c := range 3 {
		if v := got.Sky.Light[got.Sky.At(1, 0)+c]; v != 0.75 {
			t.Errorf("moon pixel channel %d = %v, want 0.75", c, v)
		}
		if v := got.Sky.Light[got.Sky.At(0, 0)+c]; v != 0.25 {
			t.Errorf("sky pixel channel %d = %v, want 0.25", c, v)
		}
		if v := got.Sky.Light[got.Sky.At(1, 3)+c]; v != 0.25 {
			t.Errorf("closed pixel channel %d = %v, want 0.25", c, v)
		}
	}
}

// TestRenderWithAcceleratorCallsEachStageInOrder checks the order of the accelerator calls and that its answers, not the CPU's, reach the result.
func TestRenderWithAcceleratorCallsEachStageInOrder(t *testing.T) {
	r, err := NewRenderer(DefaultConfig(), rigAssets())
	if err != nil {
		t.Fatal(err)
	}
	fake := &fakeAccelerator{level: 0.75, fireOut: 0.01}
	stops := 1.0
	got, err := r.Render(Request{Seconds: 0.5, Cues: rigCues(), Stops: &stops, Accelerator: fake})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"base#1", "fixtures#1", "fixtures#2", "fire#1", "skyglow#1", "strobes#1", "searchlights#1", "lasers#1", "backhaze#1", "beam#1"}
	if strings.Join(fake.calls, " ") != strings.Join(want, " ") {
		t.Fatalf("calls %v, want %v", fake.calls, want)
	}
	if got.Strobe != 0.75 {
		t.Errorf("strobe %v, want the accelerator's 0.75", got.Strobe)
	}
	if fake.stops != 1 {
		t.Errorf("fire stops %v, want the request's 1", fake.stops)
	}
	if fake.master != 1 {
		t.Errorf("base master %v, want 1", fake.master)
	}
	if len(fake.layers) != 2 || len(fake.layers[0]) != 1 || fake.layers[0][0].Name != "overhead" || fake.layers[0][0].Amount != 0.5 {
		t.Errorf("overhead layer %v, want one layer named overhead at 0.5", fake.layers)
	}
	wantFront := float64(float32(0.01))
	for i, v := range got.Front.Light {
		if v != wantFront {
			t.Fatalf("front channel %d = %v, want the accelerator's fire %v", i, v, wantFront)
		}
	}
}

// TestRenderWithAcceleratorRefusesEachFailure checks every stage's error and invalid output is named in the error.
func TestRenderWithAcceleratorRefusesEachFailure(t *testing.T) {
	cases := []struct{ name, failAt, badAt, want string }{
		{"base error", "base#1", "", "lighting base: synthetic base#1"},
		{"base output", "", "base#1", "lighting base: invalid output"},
		{"overhead error", "fixtures#1", "", "overhead: synthetic fixtures#1"},
		{"overhead output", "", "fixtures#1", "overhead: invalid output"},
		{"fixtures error", "fixtures#2", "", "fixtures: synthetic fixtures#2"},
		{"fixtures output", "", "fixtures#2", "fixtures: invalid output"},
		{"fire error", "fire#1", "", "fire: synthetic fire#1"},
		{"fire output", "", "fire#1", "fire: invalid output"},
		{"sky glow error", "skyglow#1", "", "sky glow: synthetic skyglow#1"},
		{"sky glow output", "", "skyglow#1", "sky glow: invalid output"},
		{"strobes error", "strobes#1", "", "strobes: synthetic strobes#1"},
		{"strobes output", "", "strobes#1", "strobes: invalid output"},
		{"strobes level", "", "strobes#1:level", "strobes: invalid output"},
		{"searchlights error", "searchlights#1", "", "searchlights: synthetic searchlights#1"},
		{"searchlights sky", "", "searchlights#1", "searchlights: invalid output"},
		{"searchlights front", "", "searchlights#1:front", "searchlights: invalid output"},
		{"lasers error", "lasers#1", "", "lasers: synthetic lasers#1"},
		{"lasers sky", "", "lasers#1", "lasers: invalid output"},
		{"lasers front", "", "lasers#1:front", "lasers: invalid output"},
		{"back haze error", "backhaze#1", "", "back haze: synthetic backhaze#1"},
		{"back haze output", "", "backhaze#1", "back haze: invalid output"},
		{"beam error", "beam#1", "", "beam: synthetic beam#1"},
		{"beam sky", "", "beam#1", "beam: invalid output"},
		{"beam front", "", "beam#1:front", "beam: invalid output"},
	}
	r, err := NewRenderer(DefaultConfig(), rigAssets())
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		fake := &fakeAccelerator{failAt: tc.failAt, badAt: tc.badAt, fireOut: 0.01}
		_, err := r.Render(Request{Seconds: 0.5, Cues: rigCues(), Accelerator: fake})
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err=%v, want it to contain %q", tc.name, err, tc.want)
		}
	}
}

// TestRenderRefusals feeds Render a nil renderer, stops, passes, gains and cues that each break one rule.
func TestRenderRefusals(t *testing.T) {
	var none *Renderer
	if _, err := none.Render(Request{}); err == nil || !strings.Contains(err.Error(), "nil renderer") {
		t.Fatalf("nil renderer: %v", err)
	}
	cases := []struct {
		name   string
		assets func(*Assets)
		config func(*Config)
		cue    func(*CompleteCues)
		stops  *float64
		want   string
	}{
		{name: "stops not a number", stops: ptr(math.NaN()), want: "fire stops"},
		{name: "stops too large", stops: ptr(2000), want: "fire stops"},
		{name: "stops too small", stops: ptr(-2000), want: "fire stops"},
		{name: "no overhead pass", assets: func(a *Assets) { delete(a.Passes, "overhead") }, want: `pass "overhead": missing`},
		{name: "no flood pass", assets: func(a *Assets) { delete(a.Passes, "flood-left") }, want: `pass "flood-left": missing`},
		{name: "flood too bright to add up", config: func(c *Config) { c.Fixtures.FloodGain = 1e300 }, cue: func(c *CompleteCues) { c.FloodLeft = 1e300 }, want: "fixtures: invalid fixture flood-left intensity"},
		{name: "sky too bright for float32", cue: func(c *CompleteCues) { c.Master = 1e300 }, want: "outside finite float32 range"},
		{name: "search phase out of range", cue: func(c *CompleteCues) { c.SearchMovePhase = 1e10 }, want: "search move phase"},
	}
	for _, tc := range cases {
		assets := rigAssets()
		if tc.assets != nil {
			tc.assets(&assets)
		}
		config := DefaultConfig()
		if tc.config != nil {
			tc.config(&config)
		}
		cue := rigCues()
		if tc.cue != nil {
			tc.cue(cue)
		}
		r, err := NewRenderer(config, assets)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if _, err := r.Render(Request{Cues: cue, Stops: tc.stops}); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err=%v, want it to contain %q", tc.name, err, tc.want)
		}
	}
}

// ptr returns a pointer to a copy of v.
func ptr(v float64) *float64 { return &v }

// TestRendererRefusesBadSettingsAndImages feeds the constructor one bad config value or asset at a time.
func TestRendererRefusesBadSettingsAndImages(t *testing.T) {
	cases := []struct {
		name   string
		assets func(*Assets)
		config func(*Config)
		want   string
	}{
		{name: "nan setting", config: func(c *Config) { c.HazeGain = math.NaN() }, want: "nonfinite setting"},
		{name: "no fire frame bound", config: func(c *Config) { c.MaxFireFrames = 0 }, want: "fire frame maximum"},
		{name: "no frame rate", config: func(c *Config) { c.Fire.FPS = 0 }, want: "fire timing"},
		{name: "no beam length", config: func(c *Config) { c.Beam.Out = 0 }, want: "beam or moon geometry"},
		{name: "unnamed flood pass", config: func(c *Config) { c.Fixtures.FloodPasses[1] = "" }, want: "flood passes"},
		{name: "plate too bright for float32", assets: func(a *Assets) { a.Plate.Light[0] = 1e300 }, want: "plate channel 0"},
		{name: "mask not a number", assets: func(a *Assets) { a.Mask.Light[4] = math.NaN() }, want: "mask channel 4"},
		{name: "negative fire count", assets: func(a *Assets) { a.Fire[0].Count = -1 }, want: "fire steady: negative frame count"},
		{name: "fire frame outside count", assets: func(a *Assets) { a.Fire[1] = FireFrames{Count: 1, Images: map[int]Image{3: a.Plate}} }, want: "fire swell frame 3: outside count 1"},
	}
	for _, tc := range cases {
		assets := rigAssets()
		if tc.assets != nil {
			tc.assets(&assets)
		}
		config := DefaultConfig()
		if tc.config != nil {
			tc.config(&config)
		}
		if _, err := NewRenderer(config, assets); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err=%v, want it to contain %q", tc.name, err, tc.want)
		}
	}
}

// pfmOf encodes image as a PFM and returns the bytes.
func pfmOf(t *testing.T, image Image) *bytes.Reader {
	t.Helper()
	var buf bytes.Buffer
	if err := EncodePFM(&buf, image, -1); err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(buf.Bytes())
}

// TestStaticAssetsDecodeSnapshotAndRenderer checks stored images are decoded, copied out, shared into a renderer and refused when bad.
func TestStaticAssetsDecodeSnapshotAndRenderer(t *testing.T) {
	var stored StaticAssets
	assets := rendererAssets()
	for name, image := range map[string]Image{"plate": assets.Plate, "mask": assets.Mask, "disc": assets.Plate, "water": assets.Mask, "trees": assets.Mask} {
		if err := stored.Decode(name, pfmOf(t, image), 1, 3); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if err := stored.Decode("wrong", pfmOf(t, assets.Plate), 2, 3); err == nil {
		t.Fatal("size mismatch accepted")
	}
	infinity := math.Float32frombits(0x7f800000)
	bad := pfmBytes(t, "PF\n1 1\n-1\n", binary.LittleEndian, []float32{infinity, 0, 0})
	if err := stored.Decode("inf", bytes.NewReader(bad), 1, 1); err == nil || !strings.Contains(err.Error(), "inf channel 0") {
		t.Fatalf("infinite value: %v", err)
	}
	var nobody *StaticAssets
	if got := nobody.Snapshot("plate"); got.Light != nil || got.Width != 0 {
		t.Fatalf("nil snapshot %+v", got)
	}
	if got := stored.Snapshot("missing"); got.Light != nil {
		t.Fatalf("missing snapshot %+v", got)
	}
	snap := stored.Snapshot("plate")
	snap.Light[0] = 99
	if again := stored.Snapshot("plate"); again.Light[0] != float64(float32(0.1)) {
		t.Fatalf("snapshot shares pixels: %v", again.Light[0])
	}
	r, err := NewStaticRenderer(DefaultConfig(), &stored, StaticAssetNames{Plate: "plate", Mask: "mask", Disc: "disc", Water: "water", Trees: "trees"})
	if err != nil {
		t.Fatal(err)
	}
	if r.assets.Disc == nil || r.assets.Water == nil || r.assets.Trees == nil {
		t.Fatal("named optional images were not taken")
	}
	stored.images["disc"].Light[0] = 7
	if r.assets.Disc.Light[0] != 7 {
		t.Fatal("renderer copied the disc instead of sharing it")
	}
}

// TestNewStaticRendererRefusals checks a nil store, a missing mask and a mask of the wrong size are each refused.
func TestNewStaticRendererRefusals(t *testing.T) {
	if _, err := NewStaticRenderer(DefaultConfig(), nil, StaticAssetNames{}); err == nil || !strings.Contains(err.Error(), "uninitialized") {
		t.Fatalf("nil store: %v", err)
	}
	var stored StaticAssets
	assets := rendererAssets()
	if err := stored.Decode("plate", pfmOf(t, assets.Plate), 1, 3); err != nil {
		t.Fatal(err)
	}
	if _, err := NewStaticRenderer(DefaultConfig(), &stored, StaticAssetNames{Plate: "plate", Mask: "none"}); err == nil || !strings.Contains(err.Error(), "mask:") {
		t.Fatalf("missing mask: %v", err)
	}
	if err := stored.Decode("tall", pfmOf(t, Image{Width: 1, Height: 2, Light: make([]float64, 6)}), 1, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := NewStaticRenderer(DefaultConfig(), &stored, StaticAssetNames{Plate: "plate", Mask: "tall"}); err == nil || !strings.Contains(err.Error(), "mask: is 1x2, the frame 1x3") {
		t.Fatalf("mask of another size: %v", err)
	}
}

// TestFireByteBudgetRaisesOnlyTheDefault checks a bigger cache lifts the default limit and nothing else.
func TestFireByteBudgetRaisesOnlyTheDefault(t *testing.T) {
	cases := []struct{ configured, cache, want int64 }{
		{DefaultMaxFireBytes, DefaultMaxFireBytes * 2, DefaultMaxFireBytes * 2},
		{DefaultMaxFireBytes, DefaultMaxFireBytes / 2, DefaultMaxFireBytes},
		{5, 1000, 5},
		{DefaultMaxFireBytes + 1, DefaultMaxFireBytes * 2, DefaultMaxFireBytes + 1},
	}
	for _, tc := range cases {
		if got := FireByteBudget(tc.configured, tc.cache); got != tc.want {
			t.Errorf("FireByteBudget(%d, %d) = %d, want %d", tc.configured, tc.cache, got, tc.want)
		}
	}
}

// TestMeasureFireWorkingSetRefusals checks bad sizes, a zero maximum and a byte count too large for 64 bits are refused.
func TestMeasureFireWorkingSetRefusals(t *testing.T) {
	layers := []FireLayer{{Index: 0}, {Index: 1}, {Index: 2}}
	if _, err := MeasureFireWorkingSet(layers, 4, 4, 0); err == nil {
		t.Error("zero maximum accepted")
	}
	if _, err := MeasureFireWorkingSet(layers, 0, 4, 5); err == nil {
		t.Error("zero width accepted")
	}
	if _, err := MeasureFireWorkingSet(layers, 1<<30, 1<<26, 10); err == nil || !strings.Contains(err.Error(), "overflow") {
		t.Errorf("overflow: %v", err)
	}
}

// TestRenderFireWithoutFramesAndWithBadLimits checks the fire layer with no frames is empty, and a bad rate, frame limit or byte limit is refused.
func TestRenderFireWithoutFramesAndWithBadLimits(t *testing.T) {
	r, err := NewRenderer(DefaultConfig(), rendererAssets())
	if err != nil {
		t.Fatal(err)
	}
	cue := FireCues{Level: 1, Since: []float64{0}}
	got, err := r.renderFire(DefaultFireConfig(), cue)
	if err != nil {
		t.Fatal(err)
	}
	if got.Width != 1 || got.Height != 3 || len(got.Light) != 9 {
		t.Fatalf("fire image %dx%d with %d values", got.Width, got.Height, len(got.Light))
	}
	for i, v := range got.Light {
		if v != 0 {
			t.Fatalf("channel %d = %v with no frames", i, v)
		}
	}
	slow := DefaultFireConfig()
	slow.FPS = 0
	if _, err := r.renderFire(slow, cue); err == nil || !strings.Contains(err.Error(), "fire plan") {
		t.Errorf("zero frame rate: %v", err)
	}
	frames := [3]FireFrames{{Count: 4, Images: map[int]Image{0: rendererAssets().Plate, 1: rendererAssets().Plate}}}
	two := FireCues{Level: 1, Since: []float64{0, 1.0 / 24}}
	r.config.MaxFireFrames = 1
	if _, err := r.renderFireAccelerated(DefaultFireConfig(), two, nil, &frames); !errors.Is(err, ErrFireResources) {
		t.Errorf("frame limit: %v", err)
	}
	r.config.MaxFireFrames = 10
	r.config.MaxFireBytes = 1
	if _, err := r.renderFireAccelerated(DefaultFireConfig(), two, nil, &frames); !errors.Is(err, ErrFireResources) {
		t.Errorf("byte limit: %v", err)
	}
}

// TestRenderFireRefusesBadAcceleratorOutput checks the fire stage names an error or a wrong-sized image from the accelerator.
func TestRenderFireRefusesBadAcceleratorOutput(t *testing.T) {
	r, err := NewRenderer(DefaultConfig(), rigAssets())
	if err != nil {
		t.Fatal(err)
	}
	cue := FireCues{Level: 1, Since: []float64{0}}
	for _, tc := range []struct{ name, failAt, badAt, want string }{
		{"error", "fire#1", "", "fire: synthetic fire#1"},
		{"invalid image", "", "fire#1", "fire: invalid output"},
	} {
		fake := &fakeAccelerator{failAt: tc.failAt, badAt: tc.badAt}
		if _, err := r.renderFireAccelerated(DefaultFireConfig(), cue, fake); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v", tc.name, err)
		}
	}
}
