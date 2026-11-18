package named

import (
	"io"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

func TestOrbitalUrbanDensityRequiresCity(t *testing.T) {
	if _, err := orbitalScene().Build(map[string]any{"surface-urban-density": true}, io.Discard); err == nil {
		t.Fatal("urban density without a city was accepted")
	}
}

// Explicit opt-in: production scene construction is intentionally expensive.
func TestOrbitalScenicProductionBuild(t *testing.T) {
	if os.Getenv("ORBITAL_SCENIC_FULL") != "1" {
		t.Skip("set ORBITAL_SCENIC_FULL=1")
	}
	cut := 10909090909 * time.Nanosecond
	denseOnly := os.Getenv("ORBITAL_SCENIC_DENSE_ONLY") == "1"
	var daylight maths.Subspace
	for _, night := range []bool{false, true} {
		params := map[string]any{
			"far": false, "floor-hex-metres": 10, "distance": 4400,
			"gas": .06, "glow": .04, "neon": .15, "argon": .35,
			"still": false, "day": "0s", "surface-structures": 24,
			"surface-landforms": 24, "surface-style": "mixed", "surface-tour": true,
			"surface-district": 24, "surface-tour-duration": "15s",
			"zappy-flares": true, "surface-night": night, "surface-tour-offset": "0s",
			"zappy-flare-offset": 0, "surface-city": true, "surface-relief": true,
			"surface-park-flythrough": !denseOnly, "surface-urban-density": denseOnly,
		}
		if night {
			params["surface-tour-offset"] = "10.909090909s"
			params["zappy-flare-offset"] = 6.8
		}
		started := time.Now()
		view, err := orbitalScene().Build(params, os.Stdout)
		if err != nil {
			t.Fatalf("night=%v build: %v", night, err)
		}
		world := view.World()
		if len(world.Lamps) != 9 {
			t.Fatalf("night=%v lights=%d, want nine", night, len(world.Lamps))
		}
		var members, stored, faces, lit int
		for _, obj := range world.Things {
			if obj.Material.Glow.Luminance() > 0 {
				lit++
			}
			if g, ok := obj.Shape.(*shape.MeshGroup); ok {
				members += g.MemberCount()
				stored += g.StoredFaces()
				faces += g.FaceCount()
			}
		}
		for _, at := range []time.Duration{0, 2500 * time.Millisecond, 5 * time.Second} {
			if err := view.Advance(at); err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 6; i++ {
				lamp, ok := world.Lamps[i].(emit.Lit)
				if !ok || lamp.Strength.Luminance() <= 0 {
					t.Fatalf("night=%v time=%s missing city lamp %d", night, at, i)
				}
			}
			flare, ok := world.Lamps[8].(emit.Lit)
			if !ok {
				t.Fatal("missing flare")
			}
			offset := time.Duration(0)
			if night {
				offset = 6800 * time.Millisecond
			}
			want := orbitalFlare(at + offset)
			if want > 1e-6 && flare.Strength.Luminance() <= 0 {
				t.Fatalf("night=%v time=%s missing flare output", night, at)
			}
		}
		at := cut
		if night {
			at = 0
		}
		if err := view.Advance(at); err != nil {
			t.Fatal(err)
		}
		if !night {
			daylight = view.Eye()
		} else if !view.Eye().Origin.Near(daylight.Origin, 1e-9) || !view.Eye().Orientation.X.Near(daylight.Orientation.X, 1e-9) || !view.Eye().Orientation.Y.Near(daylight.Orientation.Y, 1e-9) || !view.Eye().Orientation.Z.Near(daylight.Orientation.Z, 1e-9) {
			t.Fatal("day/night camera discontinuity")
		}
		t.Logf("night=%v build and callbacks=%s things=%d lamps=%d mesh-members=%d stored-faces=%d logical-faces=%d glowing-materials=%d", night, time.Since(started), len(world.Things), len(world.Lamps), members, stored, faces, lit)
		view = nil
		world = nil
		runtime.GC()
	}
}
