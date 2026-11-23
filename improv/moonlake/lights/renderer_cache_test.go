package lights

import (
	"bytes"
	"testing"
)

func TestRendererStaticStorageAndPassReuse(t *testing.T) {
	source := rendererAssets()
	var stored StaticAssets
	for name, image := range map[string]Image{"plate": source.Plate, "mask": source.Mask, "overhead": source.Plate} {
		var data bytes.Buffer
		if err := encodePFM(&data, image, -1); err != nil {
			t.Fatal(err)
		}
		if err := stored.Decode(name, &data, 1, 3); err != nil {
			t.Fatal(err)
		}
	}
	snapshot := stored.Snapshot("plate")
	snapshot.Light[0] = 99
	r, err := NewStaticRenderer(DefaultConfig(), &stored, StaticAssetNames{Plate: "plate", Mask: "mask", Passes: map[string]string{"overhead": "overhead"}})
	if err != nil {
		t.Fatal(err)
	}
	if &r.assets.Plate.Light[0] != &stored.images["plate"].Light[0] {
		t.Fatal("static plate copied")
	}
	second, err := NewStaticRenderer(DefaultConfig(), &stored, StaticAssetNames{Plate: "plate", Mask: "mask", Passes: map[string]string{"overhead": "overhead"}})
	if err != nil {
		t.Fatal(err)
	}
	if &second.assets.Plate.Light[0] != &r.assets.Plate.Light[0] || &second.assets.Mask.Light[0] != &r.assets.Mask.Light[0] || &second.assets.Passes["overhead"].Light[0] != &r.assets.Passes["overhead"].Light[0] {
		t.Fatal("second renderer copied static images")
	}
	done := make(chan error, 2)
	for _, renderer := range []*Renderer{r, second} {
		go func(renderer *Renderer) {
			_, err := renderer.Render(Request{Cues: &CompleteCues{Master: 1}})
			done <- err
		}(renderer)
	}
	for i := 0; i < 2; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	stored = StaticAssets{}
	if r.assets.Plate.Light[0] != float64(float32(.1)) {
		t.Fatal("renderer lost owned storage")
	}
	for i := 0; i < 3; i++ {
		pass, err := r.pass("overhead")
		if err != nil {
			t.Fatal(err)
		}
		if &pass.Light[0] != &r.assets.Passes["overhead"].Light[0] {
			t.Fatal("pass copied")
		}
	}
	if r.assets.Plate.Light[0] == 99 {
		t.Fatal("snapshot aliases immutable storage")
	}
}

func TestRendererPassRoundedOnceAndIsolated(t *testing.T) {
	assets := rendererAssets()
	assets.Passes = map[string]Image{"overhead": cloneRendererImage(assets.Plate)}
	r, err := NewRenderer(DefaultConfig(), assets)
	if err != nil {
		t.Fatal(err)
	}
	pass, err := r.pass("overhead")
	if err != nil {
		t.Fatal(err)
	}
	if pass.Light[0] != float64(float32(.1)) || &pass.Light[0] == &assets.Passes["overhead"].Light[0] {
		t.Fatal("pass ownership or rounding")
	}
	assets.Passes["overhead"].Light[0] = 999
	if allocations := testing.AllocsPerRun(10, func() {
		again, err := r.pass("overhead")
		if err != nil || &again.Light[0] != &pass.Light[0] || again.Light[0] == 999 {
			t.Fatal("pass cache changed")
		}
	}); allocations != 0 {
		t.Fatalf("cached pass allocated %v times", allocations)
	}
}

func TestRendererWaterCacheIsolationAndReuse(t *testing.T) {
	assets := rendererAssets()
	water := cloneRendererImage(assets.Mask)
	assets.Water = &water
	r, err := NewRenderer(DefaultConfig(), assets)
	if err != nil {
		t.Fatal(err)
	}
	cue := CompleteCues{Master: 1}
	request := Request{Cues: &cue, Water: &water}
	if _, err := r.Render(request); err != nil {
		t.Fatal(err)
	}
	if r.overrideWater.Light != nil {
		t.Fatal("matching static water copied")
	}
	water.Light[0] = .25
	if _, err := r.Render(request); err != nil {
		t.Fatal(err)
	}
	cached, classified := &r.overrideWater.Light[0], &r.overrideBeamWater[0]
	if cached == &water.Light[0] {
		t.Fatal("override aliases input")
	}
	if _, err := r.Render(request); err != nil {
		t.Fatal(err)
	}
	if cached != &r.overrideWater.Light[0] || classified != &r.overrideBeamWater[0] {
		t.Fatal("unchanged override rebuilt")
	}
	water.Light[0] = .75
	if r.overrideWater.Light[0] != .25 {
		t.Fatal("caller mutation reached cache")
	}
	if _, err := r.Render(request); err != nil {
		t.Fatal(err)
	}
	if r.overrideWater.Light[0] != .75 {
		t.Fatal("changed override ignored")
	}
	if _, err := r.Render(Request{Cues: &cue}); err != nil {
		t.Fatal(err)
	}
	if r.assets.Water.Light[0] != 1 {
		t.Fatal("static water mutated")
	}
}
