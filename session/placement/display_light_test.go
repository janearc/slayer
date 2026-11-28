package placement

import (
	"math"
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

func TestSeededSpaceDisplayKeepsAuthoredSunOnLocalRebuild(t *testing.T) {
	const path = "../../sets/voyager-1-hero.yaml"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	read, err := set.Read(path, source)
	if err != nil {
		t.Fatal(err)
	}
	cache := NewDisplayCache()
	for pass := range 2 {
		scene, err := cache.Scene(read)
		if err != nil {
			t.Fatal(err)
		}
		want := DisplayLight{Kind: "Sun", Ordinal: 3,
			Author: thing.AuthorRef{Collection: "scene.world.suns", Name: "cool-rim"}}
		if len(scene.Lights) != 2 {
			t.Fatalf("build %d: expected scene sun and world rim light, got %+v", pass+1, scene.Lights)
		}
		sceneSun := scene.Lights[0]
		if sceneSun.Author != (thing.AuthorRef{Collection: "scene", Name: "sun"}) ||
			sceneSun.Kind != "Sun" || sceneSun.Ordinal != 1 || sceneSun.At != (maths.Vec{}) {
			t.Fatalf("build %d: wrong scene sun %+v", pass+1, sceneSun)
		}
		// 225° from north and 28° high points southwest and up.
		line := math.Cos(28*maths.Degree) / math.Sqrt2
		directionOfSun := maths.V(-line, math.Sin(28*maths.Degree), line)
		if sceneSun.Way.Sub(directionOfSun).Len() > 1e-12 {
			t.Fatalf("build %d: wrong scene sun direction %v", pass+1, sceneSun.Way)
		}
		got := scene.Lights[1]
		direction := got.Way
		got.Way = maths.Vec{}
		if got != want || direction.Sub(maths.V(8, 4, -4).Unit()).Len() > 1e-12 {
			t.Fatalf("build %d: wrong authored rim light %+v", pass+1, scene.Lights[1])
		}
	}
}

func TestWorldOnlySpaceDisplayOmitsSceneSun(t *testing.T) {
	const path = "../../sets/voyager-1-hero.yaml"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	before, craft, ok := strings.Cut(string(source), "  set:\n")
	if !ok {
		t.Fatal("fixture has no set block")
	}
	_, after, ok := strings.Cut(craft, "  film:\n")
	if !ok {
		t.Fatal("fixture has no film block after the craft")
	}
	read, err := set.Read(path, []byte(before+"  film:\n"+after))
	if err != nil {
		t.Fatal(err)
	}
	scene, err := NewDisplayCache().Scene(read)
	if err != nil {
		t.Fatal(err)
	}
	if len(scene.Lights) != 1 || scene.Lights[0].Author.Name != "cool-rim" {
		t.Fatalf("world-only route should carry its own authored rim light: %+v", scene.Lights)
	}
}

func TestSpaceMoonPreflightKeepsOnlySupportedDirectSource(t *testing.T) {
	const path = "../../sets/voyager-1-hero.yaml"
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const naturalSun = "light: sun\n  sun: {azimuth: 225, altitude: 28}"
	for _, row := range []struct {
		name, sources string
		want          bool
	}{
		{"direct", "moon: {azimuth: 180, altitude: 45, disc: false}", true},
		{"horizon", "moon: {azimuth: 180, altitude: 0, disc: false}", false},
		{"disc", "moon: {azimuth: 180, altitude: 45, disc: true}", false},
		{"twilight", "moon: {azimuth: 180, altitude: 45, disc: false}\n  sun: {azimuth: 80, altitude: -5}", false},
	} {
		t.Run(row.name, func(t *testing.T) {
			source := strings.Replace(string(raw), naturalSun, "light: moon\n  "+row.sources, 1)
			read, err := set.Read(path, []byte(source))
			if err != nil {
				t.Fatal(err)
			}
			scene, err := NewDisplayCache().Scene(read)
			if err != nil {
				t.Fatal(err)
			}
			if scene.SkyGradient != nil || !scene.WorkingSkyOmitted {
				t.Fatal("fast space Moon must disclose the omitted lunar sky")
			}
			marker := scene.Lights[0]
			if (marker.Kind == "Moon") != row.want {
				t.Fatalf("Moon marker for %s: %+v", row.name, scene.Lights)
			}
			if row.want && (marker.Way.Sub(maths.V(0, 1, 1).Unit()).Len() > 1e-12 ||
				marker.At != (maths.Vec{}) || marker.Ordinal != 1 || marker.Author.Name != "moon") {
				t.Fatalf("wrong directional Moon: %+v", marker)
			}
		})
	}
}

func TestWorldOnlySpaceDisplayMoonKeepsWorldLampOrderAndSkyRefusal(t *testing.T) {
	const source = `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: world-only moon
  setting: space
  seed: 7
  light: moon
  moon: {azimuth: 180, altitude: 45, disc: false}
  film: {width: 32, height: 24, wide: 45, rays: 1, exposure: 1}
  cameras:
    eye:
      location: {at: [0, -2, 0]}
      orientation: {look_at: [0, 0, 0]}
  world:
    unit: 1
    lamps:
      porch: {at: [0, 0, 1], temperature: 2700, strength: 0.35, radius: 0.0625, source: eye}
`
	const sky = `    sky:
      horizon: {srgb_linear: [0.018, 0.025, 0.045]}
      zenith: {srgb_linear: [0.004, 0.008, 0.018]}
      ground: {srgb_linear: [0.008, 0.010, 0.016]}
      source: eye
`
	for _, withSky := range []bool{false, true} {
		text := source
		if withSky {
			text += sky
		}
		read, err := set.Read("moon.yaml", []byte(text))
		if err != nil {
			t.Fatal(err)
		}
		scene, err := NewDisplayCache().Scene(read)
		if err != nil {
			t.Fatal(err)
		}
		if withSky {
			if len(scene.Lights) != 1 || scene.Lights[0].Author.Name != "porch" {
				t.Fatalf("world-only authored sky allows only its world light: %+v", scene.Lights)
			}
		} else if len(scene.Lights) != 2 || scene.Lights[1].Kind != "Moon" ||
			scene.Lights[1].Ordinal != 2 || scene.Lights[0].Ordinal != 1 {
			t.Fatalf("world-only producer stages the moon after its world light: %+v", scene.Lights)
		}
		if !withSky {
			checkWorldOnlyMoonLightLimit(t, read)
		}
	}
}

func checkWorldOnlyMoonLightLimit(t *testing.T, read set.Set) {
	t.Helper()
	world := thing.NewWorld()
	for range LightLimit - 1 {
		world.AuthoredLamp(emit.Lit{}, thing.AuthorRef{Collection: "scene.world.lamps", Name: "world lamp"})
	}
	world.Lamp(emit.Lit{})
	scene := &DisplayScene{Lights: WorkingLights(world)}
	scene.addSpaceLight(read, len(world.Lamps))
	if len(scene.Lights) != LightLimit || scene.Lights[LightLimit-1].Kind != "Moon" {
		t.Fatalf("Moon should replace the anonymous lamp, got %+v", scene.Lights[LightLimit-1])
	}
	world.AuthoredLamp(emit.Lit{}, thing.AuthorRef{Collection: "scene.world.lamps", Name: "last world lamp"})
	scene.Lights = WorkingLights(world)
	scene.addSpaceLight(read, len(world.Lamps))
	if len(scene.Lights) != LightLimit || scene.Lights[LightLimit-1].Author.Name != "last world lamp" {
		t.Fatalf("earlier authored lamp should keep the last marker, got %+v", scene.Lights[LightLimit-1])
	}
}

func TestWorkingLightsKeepStagedIdentityAndPreferAuthoredSources(t *testing.T) {
	world := thing.NewWorld()
	world.Lamp(film.Flood{})
	for range LightLimit {
		world.Lamp(emit.Lit{At: maths.V(0, 0, -5)})
	}
	world.Lamp(emit.Lit{At: maths.V(math.Inf(1), 0, 0)})
	source := thing.AuthorRef{Collection: "scene.world.spots", Name: "desk"}
	world.AuthoredLamp(film.Spot{At: maths.V(1, 2, -3), Aim: maths.XAxis}, source)
	world.Lamp(film.Sun{Toward: maths.V(0, 1, -2)})

	kept := *world
	kept.KeepLamps(func(i int) bool { return i != 0 })
	lights := WorkingLights(&kept)
	if len(lights) != LightLimit {
		t.Fatalf("got %d light markers", len(lights))
	}
	want := DisplayLight{Kind: "Spot", At: maths.V(1, 2, -3), Way: maths.XAxis, Author: source, Ordinal: LightLimit + 2}
	if lights[0] != want || lights[1].Kind != "Sun" || lights[2].Ordinal != 1 {
		t.Fatalf("wrong staged lights or order: %+v", lights[:3])
	}
	if world.LampAuthor(LightLimit+2) != source || world.LampAuthor(LightLimit+1).Name != "" {
		t.Fatal("filtering a retained world changed its source identities")
	}
}

func TestProjectLightUsesPositionAimAndDistantDirection(t *testing.T) {
	scene := &DisplayScene{Eye: maths.Local(), Width: 100, Height: 100,
		Lens: film.Pinhole{Wide: math.Pi / 2, Pixel: 1}}
	scene.Eye.Origin = maths.V(10, 0, 0)
	spot := DisplayLight{Kind: "Spot", At: maths.V(10, 0, -4), Way: maths.V(1, 1, 0)}
	got, ok := scene.ProjectLight(spot)
	if !ok || got != (LightProjection{X: .5, Y: .5, AimX: 1, AimY: -1}) {
		t.Fatalf("projected spot = %+v, %t", got, ok)
	}
	sun := DisplayLight{Kind: "Sun", Way: maths.V(0, 0, -DisplayNear/2)}
	got, ok = scene.ProjectLight(sun)
	if !ok || got.X != .5 || got.Y != .5 {
		t.Fatalf("camera translation shifted a distant light: %+v, %t", got, ok)
	}
	for _, invisible := range []DisplayLight{
		{Kind: "Sun", Way: maths.ZAxis},
		{Kind: "Point", At: maths.V(99, 0, -1)},
		{Kind: "Point", At: maths.V(10, 0, 1)},
		{Kind: "Point", At: maths.V(10, 0, -DisplayNear/2)},
		{Kind: "Point", At: maths.V(10, 0, -DisplayFar*2)},
	} {
		if _, ok := scene.ProjectLight(invisible); ok {
			t.Fatalf("showed offscreen light %+v", invisible)
		}
	}
}
