//go:build draw && darwin

package gpu_test

import (
	_ "embed"
	"errors"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

//go:embed testdata/beam-lobe.yaml
var beamLobeRaw []byte

// beamLobePlan is the lobed deck: beam-deck's plan and how far the lobed
// picture must stand from the even one.
type beamLobePlan struct {
	Weather                                   medium.GroveWeather `json:"weather"`
	Bounds                                    maths.Box           `json:"bounds"`
	Base, Ground, Minimum, Tolerance, Differs float64
}

// ground is the plan's level ground.
func (plan beamLobePlan) ground(x, z float64) float64 { return plan.Ground }

// deckDrawn is the guide scene through a level deck of a weather, by the
// CPU and on Metal; it skips without a device.
func deckDrawn(test *testing.T, plan beamLobePlan,
	weather medium.GroveWeather) (*frame.Frame, *frame.Frame) {
	test.Helper()
	scene := guideScene(test)
	field, err := medium.NewDeck(weather, plan.Bounds, plan.ground, plan.Base)
	if err != nil {
		test.Fatal(err)
	}
	material := thing.Diffuse("lobed deck", scene.plan.Volume.Material.Colour)
	scene.world.Volumes = nil
	scene.world.AddVolume(thing.NewVolume(field, material))
	size := scene.plan
	truth, err := trace.Sampled(scene.world, *scene.eye, scene.lens, beamZero,
		size.Width, size.Height, size.Samples)
	if err != nil {
		test.Fatal(err)
	}
	drawn, _, err := gpu.SampledMeasured(scene.world, *scene.eye, scene.lens,
		beamZero, size.Width, size.Height, size.Samples)
	if errors.Is(err, metal.ErrNoDevice) {
		test.Skip("no Metal device")
	}
	if err != nil {
		test.Fatal(err)
	}
	return truth, drawn
}

// TestLopsidedDeckAgreesOnMetal: a deck that scatters forward by a lobe
// draws on Metal as the CPU draws it, within beam-deck's tolerance; and,
// its near miss, the lobed picture stands apart from the even one, so
// the lobe reached both tracers and is not the even deck again.
func TestLopsidedDeckAgreesOnMetal(test *testing.T) {
	var plan beamLobePlan
	if err := yamlfile.Read("testdata/beam-lobe.yaml", beamLobeRaw,
		&plan); err != nil {
		test.Fatal(err)
	}
	if plan.Weather.Asymmetry == 0 {
		test.Fatal("the lobed deck's weather says no asymmetry")
	}
	truth, drawn := deckDrawn(test, plan, plan.Weather)
	energy, apart := beamDifference(truth, drawn)
	if !(energy >= plan.Minimum) || !(apart/energy <= plan.Tolerance) {
		test.Fatal("lost lobed scattering or CPU agreement", energy, apart)
	}
	even := plan.Weather
	even.Asymmetry = 0
	evenTruth, evenDrawn := deckDrawn(test, plan, even)
	if _, off := beamDifference(evenTruth, truth); !(off/energy >= plan.Differs) {
		test.Fatal("the CPU's lobed deck is the even one", off/energy)
	}
	if _, off := beamDifference(evenDrawn, drawn); !(off/energy >= plan.Differs) {
		test.Fatal("Metal's lobed deck is the even one", off/energy)
	}
	test.Logf("lobed deck: relative RGB error %.8g", apart/energy)
}
