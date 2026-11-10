//go:build darwin

package gpu_test

import (
	_ "embed"
	"errors"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

//go:embed testdata/beam-deck.yaml
var beamDeckRaw []byte

type beamDeckPlan struct {
	Weather                          medium.GroveWeather `json:"weather"`
	Bounds                           maths.Box           `json:"bounds"`
	Base, Ground, Minimum, Tolerance float64
}

func (plan beamDeckPlan) ground(x, z float64) float64 { return plan.Ground }

func TestGuideBeamScattersThroughALevelDeckOnMetal(test *testing.T) {
	var deck beamDeckPlan
	if err := yamlfile.Read("testdata/beam-deck.yaml", beamDeckRaw, &deck); err != nil {
		test.Fatal(err)
	}
	scene := guideScene(test)
	field, err := medium.NewDeck(deck.Weather, deck.Bounds, deck.ground, deck.Base)
	if err != nil {
		test.Fatal(err)
	}
	material := thing.Diffuse("level deck", scene.plan.Volume.Material.Colour)
	scene.world.Volumes = nil
	scene.world.AddVolume(thing.NewVolume(field, material))
	plan := scene.plan
	truth, err := trace.Sampled(scene.world, *scene.eye, scene.lens, beamZero, plan.Width, plan.Height, plan.Samples)
	if err != nil {
		test.Fatal(err)
	}
	drawn, profile, err := gpu.SampledMeasured(scene.world, *scene.eye, scene.lens, beamZero, plan.Width, plan.Height, plan.Samples)
	if errors.Is(err, metal.ErrNoDevice) {
		test.Skip("no Metal device")
	}
	if err != nil {
		test.Fatal(err)
	}
	if profile.Device == "" || profile.TraceGPU <= beamZero {
		test.Fatal("deck was not traced on hardware", profile)
	}
	energy, apart := beamDifference(truth, drawn)
	if !(energy >= deck.Minimum) || !(apart/energy <= deck.Tolerance) {
		test.Fatal("lost deck scattering or CPU agreement", energy, apart)
	}
	test.Logf("%s: level deck, GPU %s; relative RGB error %.8g", profile.Device, profile.TraceGPU, apart/energy)
}
