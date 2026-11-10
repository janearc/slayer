//go:build darwin

package metal

import (
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestNativeBoundaryFailuresRemainErrors injects platform replies in an
// isolated subprocess. It proves cgo error propagation, not physical GPU
// exhaustion or a driver crash. Healthy-device controls run in the main suite.
func TestNativeBoundaryFailuresRemainErrors(t *testing.T) {
	if mode := os.Getenv("SLAYER_METAL_BOUNDARY_FAULT"); mode != "" {
		checkNativeBoundaryFault(t, mode)
		return
	}
	library := filepath.Join(t.TempDir(), "boundary.dylib")
	compile := exec.Command("cc", "-dynamiclib", "-fobjc-arc",
		"-framework", "Foundation", "-framework", "Metal",
		"testdata/boundary_faults.m", "-o", library)
	if out, err := compile.CombinedOutput(); err != nil {
		t.Fatalf("compile platform fault fixture: %v\n%s", err, out)
	}
	for _, mode := range []string{"no-device", "no-raytracing", "no-buffers", "command-error", "small-buffer-limit"} {
		t.Run(mode, func(t *testing.T) {
			args := []string{"-test.run=^TestNativeBoundaryFailuresRemainErrors$", "-test.v"}
			if dir := flag.Lookup("test.gocoverdir"); testing.CoverMode() != "" && dir != nil && dir.Value.String() != "" {
				args = append(args, "-test.gocoverdir="+dir.Value.String())
			}
			child := exec.Command(os.Args[0], args...)
			child.Env = append(os.Environ(), "DYLD_INSERT_LIBRARIES="+library,
				"SLAYER_METAL_BOUNDARY_FAULT="+mode)
			out, err := child.CombinedOutput()
			if err != nil {
				t.Fatalf("platform fault %s: %v\n%s", mode, err, out)
			}
			t.Logf("%s: platform refusal and cause verified", mode)
		})
	}
}

func checkNativeBoundaryFault(t *testing.T, mode string) {
	t.Helper()
	if mode == "no-device" {
		_, one := Open(first, "first")
		_, two := OpenSession(twoSteps, "twice")
		_, three := OpenNoise()
		for _, err := range []error{one, two, three} {
			if !errors.Is(err, ErrNoDevice) {
				t.Fatalf("device absence misclassified: %v", err)
			}
		}
		return
	}
	if mode == "no-raytracing" {
		kernel, err := Open(acceleratedProbe, "probe")
		requireDevice(t, err)
		defer kernel.Close()
		plan, err := NewAcceleration(nil, 16)
		if err != nil {
			t.Fatal(err)
		}
		root, err := plan.Instances()
		if err != nil {
			t.Fatal(err)
		}
		if err := kernel.Accelerate(plan, root); !errors.Is(err, ErrNoRayTracing) {
			t.Fatalf("native capability refusal: %v", err)
		}
		return
	}
	if mode == "small-buffer-limit" {
		session, err := OpenSession(twoSteps, "twice")
		requireDevice(t, err)
		defer session.Close()
		empty, err := session.Buffer(0)
		if err != nil || len(empty.Numbers()) != 0 {
			t.Fatalf("zero-length buffer: %v, %v", empty.Numbers(), err)
		}
		last, err := session.Buffer(4)
		if err != nil || len(last.Numbers()) != 4 {
			t.Fatalf("four floats at the 16-byte device limit: %v, %v", last.Numbers(), err)
		}
		if _, err := session.Buffer(5); err == nil ||
			!strings.Contains(err.Error(), "buffer exceeds device limit") {
			t.Fatalf("five floats exceed the 16-byte device limit: %v", err)
		}
		return
	}
	want := "could not make"
	if mode == "command-error" {
		want = "injected command failure"
	}
	points, err := Open(first, "first")
	requireDevice(t, err)
	defer points.Close()
	if _, err := points.Points([][3]float32{{1, 0, 0}}, nil); err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("native points cause: %v", err)
	}
	grid, err := Open(fillGrid, "fill_grid")
	requireDevice(t, err)
	defer grid.Close()
	if _, err := grid.Grid(2, 2, 1, 1, nil); err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("native grid cause: %v", err)
	}
	session, err := OpenSession(twoSteps, "twice")
	requireDevice(t, err)
	defer session.Close()
	native, err := Open(acceleratedProbe, "probe")
	requireDevice(t, err)
	defer native.Close()
	plan, err := NewAcceleration(triangleData(), 16)
	if err != nil {
		t.Fatal(err)
	}
	triangle, err := plan.Triangles(0, 1)
	if err != nil {
		t.Fatal(err)
	}
	place, err := Placed(triangle, translated(0, 0, 0), 3)
	if err != nil {
		t.Fatal(err)
	}
	root, err := plan.Instances(place)
	if err != nil {
		t.Fatal(err)
	}
	nativeError := native.Accelerate(plan, root)
	nativePrefix := "metal: could not allocate acceleration vertices"
	if mode == "command-error" {
		nativePrefix = "metal: acceleration build"
	}
	if nativeError == nil || !strings.HasPrefix(nativeError.Error(), nativePrefix) {
		t.Fatalf("native acceleration cause: %v", nativeError)
	}
	buffer, err := session.Buffer(2)
	if mode == "no-buffers" {
		if err == nil || !strings.Contains(err.Error(), "could not make a buffer") {
			t.Fatalf("native buffer cause: %v", err)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Dispatch("twice", 2, 1, nil, buffer); err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(); err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("native commit cause: %v", err)
	}
}
