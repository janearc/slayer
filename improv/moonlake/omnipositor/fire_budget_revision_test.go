package omnipositor

import (
	"github.com/janearc/slayer/improv/moonlake/lights"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/settings"
	"testing"
)

func TestAssetFireDefaultBudgetUsesLargerConfiguredCache(t *testing.T) {
	for _, private := range []bool{false, true} {
		s := assetSettings(t)
		s.Output.MemoryCacheBytes = 2 << 30
		s.Fire.Frames = &settings.OptionalPath{Path: assetFireDir(t, 1)}
		loaded, err := loadAssets(s, private)
		if err != nil {
			t.Fatal(err)
		}
		config := lights.DefaultConfig()
		stage, err := loaded.NewLightStage(config)
		if err != nil {
			t.Fatal(err)
		}
		if stage.maxFireBytes != s.Output.MemoryCacheBytes {
			t.Fatalf("configured budget ignored: %d", stage.maxFireBytes)
		}
		config.MaxFireBytes = 96
		stage, err = loaded.NewLightStage(config)
		if err != nil {
			t.Fatal(err)
		}
		if stage.maxFireBytes != 96 {
			t.Fatal("explicit limit replaced")
		}
	}
}
