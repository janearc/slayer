//go:build darwin

package gpu

import (
	"errors"
	stdflag "flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

func TestGPUPropagatesPlatformFailuresThroughBakesAndDrawing(t *testing.T) {
	if mode := os.Getenv("SLAYER_METAL_BOUNDARY_FAULT"); mode != "" {
		checkGPUPlatformFailure(t, mode)
		return
	}
	library := filepath.Join(t.TempDir(), "boundary.dylib")
	compile := exec.Command("cc", "-dynamiclib", "-fobjc-arc",
		"-framework", "Foundation", "-framework", "Metal",
		"../metal/testdata/boundary_faults.m", "-o", library)
	if out, err := compile.CombinedOutput(); err != nil {
		t.Fatalf("compile platform fixture: %v\n%s", err, out)
	}
	for _, mode := range []string{"no-device", "no-raytracing", "no-buffers", "command-error", "no-smoke-kernel", "lighting-error-after-work"} {
		t.Run(mode, func(t *testing.T) {
			args := []string{"-test.run=^TestGPUPropagatesPlatformFailuresThroughBakesAndDrawing$"}
			if dir := stdflag.Lookup("test.gocoverdir"); testing.CoverMode() != "" && dir != nil && dir.Value.String() != "" {
				args = append(args, "-test.gocoverdir="+dir.Value.String())
			}
			child := exec.Command(os.Args[0], args...)
			child.Env = append(os.Environ(), "DYLD_INSERT_LIBRARIES="+library,
				"SLAYER_METAL_BOUNDARY_FAULT="+mode)
			if out, err := child.CombinedOutput(); err != nil {
				t.Fatalf("platform fault %s: %v\n%s", mode, err, out)
			}
		})
	}
}

func checkGPUPlatformFailure(t *testing.T, mode string) {
	t.Helper()
	if mode == "lighting-error-after-work" {
		checkLightingCostAfterEarlierWork(t)
		return
	}
	want := "could not make"
	if mode == "command-error" {
		want = "injected command failure"
	}
	check := func(name string, err error) {
		t.Helper()
		if mode == "no-device" {
			if !errors.Is(err, metal.ErrNoDevice) {
				t.Fatalf("%s lost device-absence cause: %v", name, err)
			}
		} else if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("%s lost native cause %q: %v", name, want, err)
		}
	}
	if mode == "no-device" {
		// Empty preparation is a host-side no-op, even without Metal.
		empty := newScene()
		elapsed, err := empty.bakeAll()
		if err != nil || elapsed != 0 {
			t.Fatalf("empty bake used an unavailable device: %v %v", elapsed, err)
		}
	}
	world := thing.NewWorld()
	eye, lens := maths.Subspace{}, film.Pinhole{}
	if mode == "no-raytracing" {
		_, _, err := SampledNative(world, eye, lens, 0, 4, 3, 1)
		if !errors.Is(err, metal.ErrNoRayTracing) {
			t.Fatalf("native mode changed after capability refusal: %v", err)
		}
		return
	}
	if mode == "no-smoke-kernel" {
		// An optional cache has no job without smoke. Its missing
		// kernel must not affect a render that needs only trace.
		idleWorld, idleEye := fire(t)
		idleWorld.Volumes = nil
		picture, idle, idleErr := SampledCached(idleWorld,
			idleEye, lens, 0, 4, 3, 1)
		if idleErr != nil || picture == nil {
			t.Fatalf("no-smoke render requested an unused lighting kernel: %v", idleErr)
		}
		if idle.LightingWall != 0 || idle.LightingGPU != 0 {
			t.Fatalf("no-smoke render reported lighting preparation: %+v", idle)
		}
		world, eye := fire(t)
		world.Volumes = world.Volumes[1:]
		_, profile, err := SampledCached(world, eye, lens, 0, 4, 3, 1)
		if err == nil || !strings.Contains(err.Error(), "no kernel named smoke_light") {
			t.Fatalf("cached draw ignored its missing lighting kernel: %v", err)
		}
		if profile.LightingGPU != 0 || profile.TraceGPU != 0 {
			t.Fatalf("missing lighting kernel reported unsubmitted GPU work: %+v", profile)
		}
		return
	}
	_, _, err := SampledMeasured(world, eye, lens, 0, 4, 3, 1)
	check("draw", err)
	world.AddVolume(thing.Volume{Medium: medium.Wispy{
		Of: medium.Puff{Radius: 1, Thick: .3}, Scale: 1, Amount: .5}})
	encoded, err := encode(world, eye, lens, 0, 4, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded.bakes) != 1 {
		t.Fatal("fixture lost its wispy bake")
	}
	_, err = encoded.bakeWisps()
	check("wispy bake", err)
	_, err = encoded.bakeAll()
	check("combined bake", err)
	drawn, profile, drawErr := SampledMeasured(world, eye, lens, 0, 4, 3, 1)
	check("draw preparation", drawErr)
	if drawn != nil || profile.Compile != 0 || profile.AccelerationWall != 0 ||
		profile.DispatchWall != 0 || profile.Finish != 0 || profile.InputBytes != 0 ||
		profile.SampleBytes != 0 || profile.Device != "" {
		t.Fatalf("failed preparation entered later render stages: %+v", profile)
	}
	if mode == "no-device" || mode == "no-buffers" {
		// Neither refusal can submit work. Failed renders still expose
		// their measured profile, so a made-up duration is observable.
		if profile.BakeGPU != 0 || profile.TraceGPU != 0 {
			t.Fatalf("failed preparation reported GPU work: %+v", profile)
		}
	}
	grid := fireGridFixture(t, 5, 0)
	density, shine, elapsed, bakeErr := bakeFlame(grid)
	check("flame grid", bakeErr)
	if mode == "no-device" || mode == "no-buffers" {
		if density != nil || shine != nil || elapsed != 0 {
			t.Fatalf("unsubmitted flame bake reported output or GPU work: %d %d %v",
				len(density), len(shine), elapsed)
		}
	}
	encoded = newScene()
	encoded.deferFlame(grid)
	elapsed, err = encoded.bakeFlames()
	check("deferred flame", err)
	flameWorld := thing.NewWorld()
	flameWorld.AddVolume(thing.Volume{Medium: grid.med,
		Material: &thing.Material{Name: "flame", Shine: grid.shine}})
	flamePicture, flameProfile, flameErr := SampledMeasured(flameWorld, eye, lens, 0, 4, 3, 1)
	check("flame draw preparation", flameErr)
	if flamePicture != nil || flameProfile.Compile != 0 || flameProfile.AccelerationWall != 0 ||
		flameProfile.DispatchWall != 0 || flameProfile.Finish != 0 || flameProfile.InputBytes != 0 ||
		flameProfile.SampleBytes != 0 || flameProfile.Device != "" {
		t.Fatalf("failed flame preparation entered later render stages: %+v", flameProfile)
	}
	if mode == "no-device" || mode == "no-buffers" {
		if elapsed != 0 || flameProfile.BakeGPU != 0 || flameProfile.TraceGPU != 0 {
			t.Fatalf("failed flame preparation reported GPU work: %v %+v", elapsed, flameProfile)
		}
	}
	if mode == "no-device" {
		return
	}
	world, eye = fire(t)
	world.Volumes = world.Volumes[1:]
	encoded, err = encode(world, eye, lens, 0, 4, 3)
	if err != nil {
		t.Fatal(err)
	}
	encoded.reserveLightingAt(4)
	if len(encoded.smokes) != 1 {
		t.Fatal("fixture lost its lighting grid")
	}
	kernel, err := metal.Open(source(), "trace")
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	_, err = encoded.bakeLighting(kernel, encoded.inputs())
	check("smoke lighting dispatch", err)
}

// checkLightingCostAfterEarlierWork uses a real successful dispatch, then
// activates the existing native command-error fixture before the next one.
func checkLightingCostAfterEarlierWork(t *testing.T) {
	t.Helper()
	world, eye := fire(t)
	world.Volumes = world.Volumes[1:]
	encoded, err := encode(world, eye, film.Pinhole{}, 0, 4, 3)
	if err != nil {
		t.Fatal(err)
	}
	encoded.reserveLightingAt(4)
	if len(encoded.smokes) != 1 {
		t.Fatal("fixture lost its lighting grid")
	}
	kernel, err := metal.Open(source(), "trace")
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	if _, err := kernel.Grid(4, 3, rows, encoded.perSample(), encoded.inputs()); err != nil {
		t.Fatalf("prior successful trace: %v", err)
	}
	prior := kernel.GPUTime()
	if prior <= 0 {
		t.Fatal("prior dispatch reported no measured work")
	}
	lighting, err := encoded.bakeLighting(kernel, encoded.inputs())
	if err != nil || lighting <= 0 || prior+lighting != kernel.GPUTime() {
		t.Fatalf("successful lighting lost its own incremental cost: prior=%v lighting=%v after=%v err=%v",
			prior, lighting, kernel.GPUTime(), err)
	}
	before := kernel.GPUTime()

	// The fixture installs its command-buffer fault when a device is opened.
	// Activating it now preserves the real cost already on the first kernel.
	t.Setenv("SLAYER_METAL_BOUNDARY_FAULT", "command-error")
	activation, err := metal.Open(source(), "trace")
	if err != nil {
		t.Fatal(err)
	}
	activation.Close()
	spent, err := encoded.bakeLighting(kernel, encoded.inputs())
	if err == nil || !strings.Contains(err.Error(), "injected command failure") {
		t.Fatalf("lighting dispatch lost the original native failure: %v", err)
	}
	if spent < 0 || before+spent != kernel.GPUTime() {
		t.Fatalf("lighting cost counted earlier work: before=%v increment=%v after=%v",
			before, spent, kernel.GPUTime())
	}
}
