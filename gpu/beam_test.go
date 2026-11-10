//go:build darwin

package gpu_test

import (
	_ "embed"
	"errors"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
	"math"
	"strings"
	"testing"
)

// beamZero is defined as no light or elapsed time.
const beamZero = 0

//go:embed testdata/beam.yaml
var guideRaw []byte

// beamPlan holds authored fixture inputs, rather than hardcoded scenery.
type beamPlan struct {
	Width      int          `json:"width"`
	Height     int          `json:"height"`
	Samples    int          `json:"samples"`
	Wide       float64      `json:"wide"`
	Eye        maths.Vec    `json:"eye"`
	Target     maths.Vec    `json:"target"`
	Source     emit.Lit     `json:"source"`
	Radius     float64      `json:"radius"`
	Reach      float64      `json:"reach"`
	Divergence float64      `json:"divergence"`
	Mist       medium.Puff  `json:"mist"`
	Volume     thing.Volume `json:"volume"`
	Minimum    float64      `json:"minimum"`
	Tolerance  float64      `json:"tolerance"`
}

// beamScene owns a small CPU/Metal comparison, including its checked camera.
type beamScene struct {
	plan  beamPlan
	world *thing.World
	eye   *maths.Subspace
	lens  *film.Pinhole
}

func guideScene(test *testing.T) *beamScene {
	test.Helper()
	var plan beamPlan
	if err := yamlfile.Read("testdata/beam.yaml", guideRaw, &plan); err != nil {
		test.Fatal(err)
	}
	lens, err := film.PinholeOf(plan.Wide * maths.Degree)
	if err != nil {
		test.Fatal(err)
	}
	eye, err := maths.View(plan.Eye, plan.Target, maths.YAxis)
	if err != nil {
		test.Fatal(err)
	}
	beam, err := emit.NewBeam(plan.Source, measure.Metres(plan.Radius), measure.Metres(plan.Reach), measure.Degrees(plan.Divergence))
	if err != nil {
		test.Fatal(err)
	}
	world := thing.NewWorld()
	world.Lamp(beam)
	plan.Volume.Medium = plan.Mist
	world.AddVolume(plan.Volume)
	return &beamScene{plan: plan, world: world, eye: eye, lens: lens}
}

func TestGuideBeamLightsMistOnTheGPU(test *testing.T) {
	scene := guideScene(test)
	plan := scene.plan
	drawn, profile, err := gpu.SampledMeasured(scene.world, *scene.eye, scene.lens, beamZero, plan.Width, plan.Height, plan.Samples)
	if errors.Is(err, metal.ErrNoDevice) {
		test.Skip("no Metal device")
	}
	if err != nil {
		test.Fatal(err)
	}
	if profile.Device == "" || profile.TraceGPU <= beamZero {
		test.Fatal("beam was not traced on hardware", profile)
	}
	truth, err := trace.Sampled(scene.world, *scene.eye, scene.lens, beamZero, plan.Width, plan.Height, plan.Samples)
	if err != nil {
		test.Fatal(err)
	}
	energy, apart := beamDifference(truth, drawn)
	if !(energy >= plan.Minimum) || !(apart/energy <= plan.Tolerance) {
		test.Fatal("mist beam is absent or differs from CPU", energy, apart)
	}
	scene.assertVacuum(test)
	test.Logf("%s: beam through mist, GPU %s; RGB relative error %.8g", profile.Device, profile.TraceGPU, apart/energy)
}

// beamDifference integrates absolute channel energy and absolute error.
func beamDifference(truth, drawn *frame.Frame) (float64, float64) {
	energy, apart := float64(beamZero), float64(beamZero)
	for index, pixel := range truth.Pix {
		other := drawn.Pix[index]
		energy += math.Abs(float64(pixel.R)) + math.Abs(float64(pixel.G)) + math.Abs(float64(pixel.B))
		apart += math.Abs(float64(pixel.R-other.R)) + math.Abs(float64(pixel.G-other.G)) + math.Abs(float64(pixel.B-other.B))
	}
	return energy, apart
}

// assertVacuum holds the lack of an emissive beam skin independently of CPU.
func (scene *beamScene) assertVacuum(test *testing.T) {
	test.Helper()
	scene.world.Volumes = nil
	plan := scene.plan
	vacuum, err := gpu.Sampled(scene.world, *scene.eye, scene.lens, beamZero, plan.Width, plan.Height, plan.Samples)
	if err != nil {
		test.Fatal(err)
	}
	for _, pixel := range vacuum.Pix {
		if pixel.R != beamZero || pixel.G != beamZero || pixel.B != beamZero {
			test.Fatal("beam acquired a visible skin in vacuum", pixel)
		}
	}
}

func TestGuideBeamRefusesTypedNil(test *testing.T) {
	scene := guideScene(test)
	scene.world.Lamps = nil
	var missing *emit.Beam
	scene.world.Lamp(missing)
	plan := scene.plan
	_, err := gpu.Sampled(scene.world, *scene.eye, scene.lens, beamZero,
		plan.Width, plan.Height, plan.Samples)
	if err == nil || !strings.Contains(err.Error(), "a nil beam") {
		test.Fatal("nil beam was not refused", err)
	}
}
