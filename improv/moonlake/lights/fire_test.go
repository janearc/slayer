package lights

import (
	"math"
	"reflect"
	"slices"
	"testing"
)

func TestContextFirePlanIgnoresUnusedSteadyTiming(t *testing.T) {
	config := DefaultFireConfig()
	cue := FireCues{Level: 1, Since: []float64{0.1}}
	want, err := PlanFire(config, cue, [3]int{8})
	if err != nil {
		t.Fatal(err)
	}
	for _, setting := range []float64{-1, math.NaN(), math.Inf(1), 1e30} {
		config.SteadyFrom = setting
		got, err := PlanFire(config, cue, [3]int{8})
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("unused steady timing %v: layers=%v err=%v", setting, got, err)
		}
	}
}

func TestContextFirePlanPreservesLayerOrderAndFrameSelection(t *testing.T) {
	config := DefaultFireConfig()
	config.Percussive, config.Burst = true, true
	cue := FireCues{Level: 0.75, Lit: 0.2, Since: []float64{0.1, 0.2, 2}}
	layers, err := PlanFire(config, cue, [3]int{43, 2, 10})
	if err != nil {
		t.Fatal(err)
	}
	level := config.Gain * cue.Level * (1 + config.Pulse*math.Exp(-0.1/config.Snap))
	swell := math.Exp(-0.1 / config.SwellFall)
	want := []FireLayer{
		{FireSteady, 42, level * (1 - swell)},
		{fireSwell, 1, level * swell},
		{FireBurst, 2, config.Gain * cue.Level * config.BurstGain},
		{FireBurst, 5, config.Gain * cue.Level * config.BurstGain * 2},
	}
	if !reflect.DeepEqual(layers, want) {
		t.Fatalf("layers=%v want=%v", layers, want)
	}
	config.Percussive = false
	layers, err = PlanFire(config, cue, [3]int{6})
	want = []FireLayer{
		{FireSteady, 2, config.Gain * cue.Level * math.Exp(-0.1/config.Decay)},
		{FireSteady, 5, config.Gain * cue.Level * math.Exp(-0.2/config.Decay)},
	}
	if err != nil || !reflect.DeepEqual(layers, want) {
		t.Fatalf("decaying layers=%v want=%v err=%v", layers, want, err)
	}
}

func TestContextFireRenderUsesOnlyExplicitInputs(t *testing.T) {
	config := DefaultFireConfig()
	config.Percussive, config.Burst = true, true
	cue := FireCues{Level: 0.75, Lit: 0.2, Since: []float64{0.1, 0.2}}
	input := FireInput{Width: 2, Height: 1, Cues: cue, CloseTrees: []bool{false, true}}
	pass := Image{Width: 2, Height: 1, Light: []float64{1e-7, 2e-8, 3e-9, 1, 1, 1}}
	before := slices.Clone(pass.Light)
	for i, count := range [3]int{43, 2, 10} {
		input.Frames[i] = FireFrames{Count: count, Images: map[int]Image{}}
	}
	layers, err := PlanFire(config, cue, [3]int{43, 2, 10})
	if err != nil {
		t.Fatal(err)
	}
	want := Image{Width: 2, Height: 1, Light: make([]float64, 6)}
	for _, layer := range layers {
		input.Frames[layer.Sequence].Images[layer.Index] = pass
		for i := range 3 {
			want.Light[i] += layer.Amount * pass.Light[i]
		}
	}
	fireShoulder(want, config.Top*FireStrike(config, cue), math.Exp2(config.Stops))
	t.Setenv("FIRE_GAIN", "invalid")
	t.Setenv("FIRE_FRAMES", "/nonexistent")
	t.Setenv("FIRE_SWELL", "invalid")
	for range 3 {
		got, err := RenderFire(config, input)
		if err != nil || !slices.Equal(got.Light, want.Light) {
			t.Fatalf("fire=%v want=%v err=%v", got.Light, want.Light, err)
		}
		got.Light[0] = 999
	}
	if !slices.Equal(pass.Light, before) || !reflect.DeepEqual(cue.Since, []float64{0.1, 0.2}) || !reflect.DeepEqual(input.CloseTrees, []bool{false, true}) {
		t.Fatal("render mutated inputs")
	}
}

func TestContextFireRejectsMissingFramesAndInvalidMasks(t *testing.T) {
	config := DefaultFireConfig()
	input := FireInput{Width: 1, Height: 1, Cues: FireCues{Level: 1, Since: []float64{0}}}
	input.Frames[0].Count = 1
	if _, err := RenderFire(config, input); err == nil {
		t.Fatal("missing selected frame accepted")
	}
	input.Frames[0].Images = map[int]Image{0: {Width: 1, Height: 1, Light: []float64{1, 1, 1}}}
	input.CloseTrees = []bool{}
	if _, err := RenderFire(config, input); err == nil {
		t.Fatal("invalid mask accepted")
	}
	input.CloseTrees = nil
	input.Cues.Since = []float64{-0.1}
	if _, err := RenderFire(config, input); err == nil {
		t.Fatal("negative frame age accepted")
	}
	input.Cues.Since = []float64{math.Inf(1)}
	if _, err := RenderFire(config, input); err == nil {
		t.Fatal("infinite frame age accepted")
	}
}

func TestContextFireConcurrentRendersAreIndependent(t *testing.T) {
	input := FireInput{Width: 1, Height: 1, Cues: FireCues{Level: 1, Since: []float64{0}}}
	input.Frames[0] = FireFrames{Count: 1, Images: map[int]Image{0: {Width: 1, Height: 1, Light: []float64{1e-7, 2e-8, 3e-9}}}}
	for _, gain := range []float64{0.1, 0.8} {
		config := DefaultFireConfig()
		config.Gain = gain
		want, err := RenderFire(config, input)
		if err != nil {
			t.Fatal(err)
		}
		t.Run("independent", func(t *testing.T) {
			t.Parallel()
			for range 8 {
				got, err := RenderFire(config, input)
				if err != nil || !slices.Equal(got.Light, want.Light) {
					t.Fatalf("fire changed: %v", err)
				}
			}
		})
	}
}
