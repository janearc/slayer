package main

import (
	_ "embed"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

//go:embed testdata/sun-tint.yaml
var sunTintRaw []byte

func TestAuthoredSunTintReachesTheActualDirectLamp(test *testing.T) {
	var data struct {
		Written   string
		Expected  maths.Light
		Tolerance float64
		Invalid   []string
	}
	if err := yamlfile.Read("sun-tint.yaml", sunTintRaw, &data); err != nil {
		test.Fatal(err)
	}
	given := mustParse(test)
	given.set = onFixture(test, "light: moon", "light: sun\n"+
		"  sun: {azimuth: 90, altitude: 6, tint: "+data.Written+"}")
	raw, err := os.ReadFile(given.set)
	if err != nil {
		test.Fatal(err)
	}
	source := strings.ReplaceAll(string(raw), "alight: true", "alight: false")
	sequenceWrite(test, given.set, []byte(source))
	world, _, ready, err := staged(given)
	if err != nil {
		test.Fatal(err)
	}
	defer ready.live.closeBeds()
	_, _, light := world.Lamps[placementZero].Light(maths.V(placementZero, placementZero, placementZero), maths.YAxis)
	difference := math.Abs(light.R-data.Expected.R) +
		math.Abs(light.G-data.Expected.G) + math.Abs(light.B-data.Expected.B)
	if difference > data.Tolerance {
		test.Fatal("actual direct sun lost tint", light, data.Expected)
	}
	for _, invalid := range data.Invalid {
		bad := strings.Replace(source, data.Written, invalid, launcherUnit)
		if _, err := set.Read("bad-tint.yaml", []byte(bad)); err == nil {
			test.Fatal("accepted invalid tint", invalid)
		}
	}
}
