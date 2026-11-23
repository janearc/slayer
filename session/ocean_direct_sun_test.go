package session

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

func directSunSet(t *testing.T, extraScene string) set.Set {
	t.Helper()
	raw, err := os.ReadFile("../set/testdata/kilauea-iki.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), "  light: moon\n",
		"  light: sun\n  sun: {azimuth: 180, altitude: 35}\n"+extraScene, 1)
	world := `  world:
    ocean:
      material: water
      direct-sun: mean-plane
      middle: [0, 0, 0]
      extent: {value: 100, unit: m}
      samples: 9
      sea:
        wind: {value: 0, unit: m/s}
        fetch: {value: 0, unit: m}
        depth: {value: 0, unit: m}
        toward: {value: 0, unit: rad}
      seed: 1
      time: {value: 0, unit: s}
      source: "synthetic validation inputs"
    materials:
      water:
        clear: {through: 1, index: 1.33, gloss: 1, rank: 0, extinction-per-metre: [0.1, 0.1, 0.1]}
        source: "synthetic validation coefficients"
`
	text = strings.Replace(text, "  set:\n", world+"  set:\n", 1)
	read, err := set.Read("direct-sun-ground.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return read
}

func TestGroundDirectSunTagUsesCapturedDistantLampAndSeparateFill(t *testing.T) {
	read := directSunSet(t, "")
	sun := skyward(180, 35)
	direct, err := emit.NewDistant(sun, maths.Grey(day))
	if err != nil {
		t.Fatal(err)
	}
	fill, err := emit.NewDistant(fillWay, maths.Grey(day*fill))
	if err != nil {
		t.Fatal(err)
	}
	stage := staging{given: asked{set: "direct-sun-ground.yaml"},
		world: thing.NewWorld()}
	stage.world.Lamps = []thing.Lamp{direct, fill}
	if err := stage.tagOceanDirectSun(read, 0); err != nil {
		t.Fatal(err)
	}
	if stage.given.oceanSunTag == nil ||
		stage.given.oceanSunTag.sun != 0 || stage.given.oceanSunTag.fill != 1 {
		t.Fatalf("tag = %+v, want captured sun 0 and separate fill 1",
			stage.given.oceanSunTag)
	}
}

func TestGroundDirectSunTagRefusesWrongSourceOrUnsupportedRoute(t *testing.T) {
	read := directSunSet(t, "")
	fillLamp, err := emit.NewDistant(fillWay, maths.Grey(1))
	if err != nil {
		t.Fatal(err)
	}
	other, err := emit.NewDistant(maths.V(1, 1, 0), maths.Grey(1))
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		name  string
		given asked
		lamps []thing.Lamp
		read  set.Set
	}{
		{name: "index does not hold the expected sun", given: asked{set: "bad"},
			lamps: []thing.Lamp{fillLamp, other}, read: read},
		{name: "GPU", given: asked{set: "gpu", onGPU: true},
			lamps: []thing.Lamp{other, fillLamp}, read: read},
		{name: "air", given: asked{set: "air"},
			lamps: []thing.Lamp{other, fillLamp},
			read:  directSunSet(t, "  air: earth\n")},
	} {
		stage := staging{given: row.given, world: thing.NewWorld()}
		stage.world.Lamps = row.lamps
		if err := stage.tagOceanDirectSun(row.read, 0); err == nil {
			t.Errorf("%s route was silently accepted", row.name)
		}
	}
}
