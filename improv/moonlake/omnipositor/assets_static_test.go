package omnipositor

import (
	"context"
	"github.com/janearc/slayer/improv/moonlake/lights"
	"testing"
)

func TestAssetPrivateStaticStageReuseAndIsolation(t *testing.T) {
	settings := assetSettings(t)
	loaded, err := LoadStaticAssets(settings)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Lighting.Plate.Light != nil || loaded.Lighting.Mask.Light != nil || len(loaded.Lighting.Passes) != 0 || loaded.Lighting.Disc != nil {
		t.Fatal("loader retained renderer asset copies")
	}
	first, err := loaded.NewLightStage(lights.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	second, err := loaded.NewLightStage(lights.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	request := LightRequest{Width: 2, Height: 2, WaterMask: *loaded.Lighting.Water}
	before, err := first.Light(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	loaded.MoonReference.Light[0] = 999
	loaded.Lighting.Water.Light[0] = 0
	loaded.Lighting.Trees.Light[0] = 0
	after, err := second.Light(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if before.Sky.Light[0] != after.Sky.Light[0] {
		t.Fatal("public snapshot mutated renderer")
	}
	if &before.Sky.Light[0] == &after.Sky.Light[0] || &before.Sky.Light[0] == &before.Front.Light[0] {
		t.Fatal("frame buffers alias")
	}
	first.fireLoader = nil
	loaded = nil
	if _, err := first.Light(context.Background(), request); err != nil {
		t.Fatal(err)
	}
}
