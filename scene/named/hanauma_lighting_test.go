package named

import (
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

func TestHanaumaSolarLightingDayAndNight(t *testing.T) {
	values := values{"year": 2026, "month": 6, "day": 21, "hour": 12, "minute": 0}
	dayWorld := thing.NewWorld()
	night, err := hanaumaSolarLighting(dayWorld, values)
	if err != nil || night || len(dayWorld.Lamps) != 2 {
		t.Fatalf("day lighting: night=%v lamps=%d err=%v", night, len(dayWorld.Lamps), err)
	}
	sun, _, _ := dayWorld.Lamps[0].Light(maths.Vec{}, maths.Up)
	if sun.Y <= 0 {
		t.Fatalf("day sun is below horizon: %v", sun)
	}
	values["hour"] = 0
	nightWorld := thing.NewWorld()
	night, err = hanaumaSolarLighting(nightWorld, values)
	if err != nil || !night || len(nightWorld.Lamps) != 1 {
		t.Fatalf("night lighting: night=%v lamps=%d err=%v", night, len(nightWorld.Lamps), err)
	}
	if nightWorld.Sky.Light(maths.Up).R >= dayWorld.Sky.Light(maths.Up).R {
		t.Fatal("night sky is no darker than day")
	}
}

func TestHanaumaSolarLightingRejectsInvalidDate(t *testing.T) {
	values := values{"year": 2026, "month": 2, "day": 30, "hour": 12, "minute": 0}
	if _, err := hanaumaSolarLighting(thing.NewWorld(), values); err == nil {
		t.Fatal("accepted February 30")
	}
}

type hanaumaTestGlow struct{}

func (hanaumaTestGlow) At(maths.Vec) maths.Light { return maths.RGB(.1, .2, .3) }

func TestHanaumaNightDisablesOnlySeabedEmission(t *testing.T) {
	world := thing.NewWorld()
	shine := &thing.Material{Name: "Hanauma sand, reef pavement and tuff", Shine: hanaumaTestGlow{}}
	world.Add(thing.Thing{Material: shine})
	darkenHanaumaSeabed(world)
	if world.Things[0].Material.Shine != nil || shine.Shine == nil {
		t.Fatal("night should remove seabed glow without mutating the original material")
	}
}

func TestHanaumaNightSeabedRemainsDarkAcrossAdvance(t *testing.T) {
	built, err := hanaumaScene().Build(map[string]any{"lighting": "solar", "hour": 0}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, at := range []time.Duration{0, time.Second} {
		if _, err := built.advance(at); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, object := range built.world.Things {
			if object.Material != nil && object.Material.Name == "Hanauma sand, reef pavement and tuff" {
				found = true
				if object.Material.Shine != nil {
					t.Fatalf("seabed glows at %v", at)
				}
			}
		}
		if !found {
			t.Fatal("Hānauma seabed missing")
		}
	}
}
