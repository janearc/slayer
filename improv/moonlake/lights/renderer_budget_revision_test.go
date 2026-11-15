package lights

import "testing"

func TestRendererFireRealisticDefaultBudget(t *testing.T) {
	layers := []FireLayer{{Index: 0}, {Index: 1}, {Index: 2}}
	working, err := MeasureFireWorkingSet(layers, 4096, 3002, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckFireBudget(working, DefaultMaxFireBytes); err != nil {
		t.Fatal(err)
	}
	if working.TransientBytes != working.SelectedBytes*2+int64(4096)*3002*24 {
		t.Fatalf("rounding copies missing: %+v", working)
	}
}

func TestRendererFireTransientRoundingBound(t *testing.T) {
	for _, value := range []float64{.1, .5} {
		image := rendererAssets().Plate
		for i := range image.Light {
			image.Light[i] = value
		}
		frames := [3]FireFrames{{Count: 1, Images: map[int]Image{0: image}}}
		renderer, err := NewRenderer(DefaultConfig(), rendererAssets())
		if err != nil {
			t.Fatal(err)
		}
		result, err := renderer.Render(Request{Cues: &CompleteCues{Fire: 1, FireSince: []float64{0}}, FireFrames: &frames})
		if err != nil {
			t.Fatal(err)
		}
		if result.FireWorkingSet.TransientBytes != 216 {
			t.Fatalf("rounding upper bound: %+v", result.FireWorkingSet)
		}
		if image.Light[0] != value {
			t.Fatal("input mutated")
		}
	}
}
