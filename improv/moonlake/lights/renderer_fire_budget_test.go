package lights

import (
	"errors"
	"strings"
	"testing"
)

func TestRendererFireByteBudget(t *testing.T) {
	config := DefaultConfig()
	config.MaxFireBytes = 72
	r, err := NewRenderer(config, rendererAssets())
	if err != nil {
		t.Fatal(err)
	}
	frames := [3]FireFrames{{Count: 3, Images: map[int]Image{}}}
	cue := CompleteCues{Fire: 1, FireSince: []float64{0, 1.0 / 24}}
	result, err := r.Render(Request{Cues: &cue, FireFrames: &frames})
	if !errors.Is(err, ErrFireResources) || !strings.Contains(err.Error(), "144") || len(result.Sky.Light) != 0 {
		t.Fatalf("budget must precede allocations and frame access: %+v, %v", result, err)
	}
	frames[0].Images[0] = rendererAssets().Plate
	cue.FireSince = []float64{0, 0, 0}
	result, err = r.Render(Request{Cues: &cue, FireFrames: &frames})
	if err != nil || result.FireWorkingSet.SelectedBytes != 72 {
		t.Fatalf("distinct-frame reuse: %+v, %v", result.FireWorkingSet, err)
	}
	config.MaxFireBytes = 71
	r, err = NewRenderer(config, rendererAssets())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Render(Request{Cues: &cue, FireFrames: &frames}); !errors.Is(err, ErrFireResources) {
		t.Fatalf("single oversized frame: %v", err)
	}
	config.MaxFireBytes = 0
	if _, err := NewRenderer(config, rendererAssets()); err == nil {
		t.Fatal("unbounded byte budget accepted")
	}
}

func TestRendererFireBudgetAt4K(t *testing.T) {
	layers := make([]FireLayer, 4096)
	for index := range layers {
		layers[index] = FireLayer{Index: index}
	}
	workingSet, err := MeasureFireWorkingSet(layers, 3840, 2160, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckFireBudget(workingSet, DefaultMaxFireBytes); !errors.Is(err, ErrFireResources) {
		t.Fatalf("4K working set accepted: %+v, %v", workingSet, err)
	}
	config := DefaultConfig()
	config.MaxFireBytes = 71
	assets := rendererAssets()
	frames := [3]FireFrames{{Count: 1, Images: map[int]Image{0: assets.Plate}}}
	assets.Fire = &frames
	if r, err := NewRenderer(config, assets); r != nil || !errors.Is(err, ErrFireResources) {
		t.Fatalf("persistent frame budget: %v, %v", r, err)
	}
}

func TestRendererFireConcurrentRefusal(t *testing.T) {
	r, err := NewRenderer(DefaultConfig(), rendererAssets())
	if err != nil {
		t.Fatal(err)
	}
	r.requestMu.Lock()
	defer r.requestMu.Unlock()
	result, err := r.Render(Request{})
	if !errors.Is(err, ErrFireResources) || len(result.Sky.Light) != 0 {
		t.Fatalf("concurrent request: %+v, %v", result, err)
	}
}
