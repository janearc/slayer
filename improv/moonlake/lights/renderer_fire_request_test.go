package lights

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestRendererRequestFireFrameLimit(t *testing.T) {
	config := DefaultConfig()
	config.MaxFireFrames = 1
	r, err := NewRenderer(config, rendererAssets())
	if err != nil {
		t.Fatal(err)
	}
	frames := [3]FireFrames{{Count: 3, Images: map[int]Image{}}}
	cue := CompleteCues{Fire: 1, FireSince: []float64{0, 1.0 / 24}}
	if _, err := r.Render(Request{Cues: &cue, FireFrames: &frames}); err == nil || !strings.Contains(err.Error(), "maximum 1") {
		t.Fatalf("limit must precede frame access: %v", err)
	}
	frames[0].Images[0] = rendererAssets().Plate
	cue.FireSince = []float64{0, 0, 0}
	result, err := r.Render(Request{Cues: &cue, FireFrames: &frames})
	if err != nil {
		t.Fatal(err)
	}
	if result.FireWorkingSet.Frames != 1 || result.FireWorkingSet.SelectedBytes != 72 || result.FireWorkingSet.TransientBytes != 216 {
		t.Fatalf("working set: %+v", result.FireWorkingSet)
	}
	config.MaxFireFrames = 3
	r, err = NewRenderer(config, rendererAssets())
	if err != nil {
		t.Fatal(err)
	}
	frames[0].Images[1], frames[0].Images[2] = frames[0].Images[0], frames[0].Images[0]
	cue.FireSince = []float64{0, 1.0 / 24, 2.0 / 24}
	if _, err := r.Render(Request{Cues: &cue, FireFrames: &frames}); err != nil {
		t.Fatalf("configured stretch: %v", err)
	}
}

func TestRendererRequestFireFrames(t *testing.T) {
	assets := rendererAssets()
	persistent := [3]FireFrames{{Count: 2, Images: map[int]Image{0: assets.Plate}}}
	assets.Fire = &persistent
	r, err := NewRenderer(DefaultConfig(), assets)
	if err != nil {
		t.Fatal(err)
	}
	cue := CompleteCues{Fire: 1, FireSince: []float64{0}}
	base, err := r.Render(Request{Cues: &cue})
	if err != nil {
		t.Fatal(err)
	}
	dynamic := [3]FireFrames{{Count: 2, Images: map[int]Image{0: {Width: 1, Height: 3, Light: make([]float64, 9)}}}}
	got, err := r.Render(Request{Cues: &cue, FireFrames: &dynamic})
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(base.Front, got.Front) {
		t.Fatal("request frames ignored")
	}
	delete(dynamic[0].Images, 0)
	if _, err := r.Render(Request{Cues: &cue, FireFrames: &dynamic}); err == nil || !strings.Contains(err.Error(), "fire steady frame 0: missing") {
		t.Fatalf("selected missing frame fell back to persistent assets: %v", err)
	}
	again, err := r.Render(Request{Cues: &cue})
	if err != nil || !reflect.DeepEqual(base, again) {
		t.Fatalf("request overwrote persistent frames: %v", err)
	}
	r, err = NewRenderer(DefaultConfig(), rendererAssets())
	if err != nil {
		t.Fatal(err)
	}
	dynamic[0].Images[0] = assets.Plate
	if _, err := r.Render(Request{Cues: &cue, FireFrames: &dynamic}); err != nil {
		t.Fatalf("request-only fire: %v", err)
	}
	if r.assets.Fire != nil {
		t.Fatal("request fire retained")
	}
}

func TestRendererRequestFireValidation(t *testing.T) {
	r, err := NewRenderer(DefaultConfig(), rendererAssets())
	if err != nil {
		t.Fatal(err)
	}
	cue := CompleteCues{Fire: 1, FireSince: []float64{0}}
	for _, image := range []Image{
		{Width: 3, Height: 1, Light: make([]float64, 9)},
		{Width: 1, Height: 3, Light: make([]float64, 8)},
		{Width: 1, Height: 3, Light: []float64{math.NaN(), 0, 0, 0, 0, 0, 0, 0, 0}},
		{Width: 1, Height: 3, Light: []float64{math.MaxFloat64, 0, 0, 0, 0, 0, 0, 0, 0}},
	} {
		frames := [3]FireFrames{{Count: 1, Images: map[int]Image{0: image}}}
		if _, err := r.Render(Request{Cues: &cue, FireFrames: &frames}); err == nil || !strings.Contains(err.Error(), "fire steady frame 0") {
			t.Fatalf("invalid selected frame: %v", err)
		}
	}
	frames := [3]FireFrames{{Count: -1}}
	if _, err := r.Render(Request{Cues: &cue, FireFrames: &frames}); err == nil {
		t.Fatal("negative count accepted")
	}
}

func TestRendererFireBorrowedFloatBoundary(t *testing.T) {
	image := Image{Width: 1, Height: 1, Light: []float64{1, 2, 3}}
	got, err := rendererFireImage("fire", image, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if &got.Light[0] != &image.Light[0] {
		t.Fatal("already quantized frame copied")
	}
	image.Light[0] = 0.1
	got, err = rendererFireImage("fire", image, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Light[0] != float64(float32(0.1)) || image.Light[0] != 0.1 {
		t.Fatal("float boundary or input ownership changed")
	}
}
