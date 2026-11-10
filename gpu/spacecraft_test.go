//go:build darwin

package gpu_test

import (
	_ "embed"
	"errors"
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

//go:embed testdata/spacecraft.yaml
var spacecraftGPURaw []byte

type instrumentGPUView struct {
	Design      string
	Aperture    float64
	Eye, Target maths.Vec
	Source      emit.Lit
	Cutaway     bool
}
type instrumentGPUPlan struct {
	Width, Height, Samples   int
	Wide, Tolerance, Minimum float64
	Seed                     uint64
	Views                    []instrumentGPUView
}

type instrumentGPUProbe struct {
	plan  instrumentGPUPlan
	world *thing.World
	eye   *maths.Subspace
	lens  *film.Pinhole
	truth *frame.Frame
	name  string
}

func (view instrumentGPUView) world(test *testing.T,
	seed uint64,
) *thing.World {
	test.Helper()
	recipe, err := manufacture.Spacecraft(view.Design, measure.Metres(view.Aperture))
	if err != nil {
		test.Fatal(err)
	}
	object, err := manufacture.MakeWithComplexity(*recipe, manufacture.Sow(seed), "quick")
	if err != nil {
		test.Fatal(err)
	}
	parts, err := object.Things(maths.Local())
	if err != nil {
		test.Fatal(err)
	}
	world := thing.NewWorld()
	for _, part := range parts {
		world.Add(part)
	}
	world.Lamp(view.Source)
	return world
}

func TestSpacecraftMeshesAndMaterialsAgreeOnMetal(test *testing.T) {
	var plan instrumentGPUPlan
	if err := yamlfile.Read("testdata/spacecraft.yaml", spacecraftGPURaw, &plan); err != nil {
		test.Fatal(err)
	}
	lens, err := film.PinholeOf(plan.Wide * maths.Degree)
	if err != nil {
		test.Fatal(err)
	}
	for _, view := range plan.Views {
		eye, err := maths.View(view.Eye, view.Target, maths.YAxis)
		if err != nil {
			test.Fatal(err)
		}
		plan.compare(test, view.Design, view.world(test, plan.Seed), eye, lens)
	}
}

func (plan instrumentGPUPlan) compare(test *testing.T,
	name string, world *thing.World, eye *maths.Subspace, lens *film.Pinhole,
) {
	test.Helper()
	truth, err := trace.Sampled(world, *eye, lens, beamZero,
		plan.Width, plan.Height, plan.Samples)
	if err != nil {
		test.Fatal(err)
	}
	probe := instrumentGPUProbe{plan: plan, world: world, eye: eye,
		lens: lens, truth: truth, name: name}
	test.Run("software", probe.software)
	test.Run("native", probe.native)
}

func (probe instrumentGPUProbe) software(test *testing.T) {
	plan := probe.plan
	drawn, profile, err := gpu.SampledMeasured(probe.world, *probe.eye,
		probe.lens, beamZero, plan.Width, plan.Height, plan.Samples)
	probe.accept(test, probe.name+" software", probe.truth, drawn, profile, err)
}

func (probe instrumentGPUProbe) native(test *testing.T) {
	plan := probe.plan
	drawn, profile, err := gpu.SampledAccelerated(probe.world, *probe.eye,
		probe.lens, beamZero, plan.Width, plan.Height, plan.Samples)
	probe.accept(test, probe.name+" native", probe.truth, drawn, profile, err)
	if profile.Native.Queries == beamZero || profile.Native.Refined == beamZero {
		test.Fatal("instrument missed native closest-hit queries", profile.Native)
	}
	test.Logf("%s native outcomes: %+v", probe.name, profile.Native)
}

func (probe instrumentGPUProbe) accept(test *testing.T, name string,
	truth, drawn *frame.Frame, profile gpu.Profile, err error,
) {
	test.Helper()
	if errors.Is(err, metal.ErrNoDevice) {
		test.Skip("no Metal device")
	}
	if err != nil {
		test.Fatal(err)
	}
	if profile.Device == "" || profile.TraceGPU <= beamZero {
		test.Fatal("instrument did not reach hardware", profile)
	}
	energy, apart := beamDifference(truth, drawn)
	if !(energy >= probe.plan.Minimum) || !(apart/energy <= probe.plan.Tolerance) {
		worst := instrumentDisagreement(test, truth, drawn)
		probe.replay(test, worst)
		test.Fatal("missing instrument light or CPU/Metal disagreement",
			name, energy, apart)
	}
	test.Logf("%s: %s, relative RGB error %.8g",
		name, profile.Device, apart/energy)
}

// instrumentFirstPixel is the defined start of frame's row-major storage.
const instrumentFirstPixel = 0

// instrumentDisagreement preserves a concrete ray for a failed merge check.
// Its frames come from Sampled's positive-dimension rendering contract.
func instrumentDisagreement(test *testing.T, truth, drawn *frame.Frame) int {
	test.Helper()
	worst, largest := instrumentFirstPixel, float64(beamZero)
	for index, pixel := range truth.Pix {
		other := drawn.Pix[index]
		delta := math.Abs(float64(pixel.R-other.R)) +
			math.Abs(float64(pixel.G-other.G)) +
			math.Abs(float64(pixel.B-other.B))
		if delta > largest {
			worst, largest = index, delta
		}
	}
	test.Logf("largest RGB difference at (%d,%d): CPU %+v; GPU %+v",
		worst%truth.W, worst/truth.W, truth.Pix[worst], drawn.Pix[worst])
	return worst
}

// replay records only the failing pixel, rather than dumping a whole image.
func (probe instrumentGPUProbe) replay(test *testing.T, index int) {
	test.Helper()
	plan := probe.plan
	_, steps, err := trace.Replay(probe.world, *probe.eye, probe.lens,
		beamZero, plan.Width, plan.Height, plan.Samples,
		index%plan.Width, index/plan.Width)
	if err != nil {
		test.Log("could not replay the differing pixel", err)
		return
	}
	for _, step := range steps {
		test.Logf("CPU replay bounce %d: %s: %s",
			step.Bounce, step.What, step.Said)
	}
}
