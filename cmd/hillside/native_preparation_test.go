//go:build darwin

package main

import (
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/metal"
)

// TestFuelPreparationPropagatesNativeRefusals uses isolated platform replies.
// It holds real cgo propagation, not physical GPU exhaustion or driver crashes.
func TestFuelPreparationPropagatesNativeRefusals(t *testing.T) {
	if mode := os.Getenv("SLAYER_METAL_BOUNDARY_FAULT"); mode != "" {
		given := fuelOnFixture(t)
		_, _, _, err := staged(given)
		if mode != "no-device" && errors.Is(err, metal.ErrNoDevice) {
			t.Skip(err)
		}
		if mode == "no-device" {
			if !errors.Is(err, metal.ErrNoDevice) {
				t.Fatalf("staging lost device cause: %v", err)
			}
		} else if err == nil || !strings.Contains(err.Error(), "could not make a buffer") {
			t.Fatalf("staging lost allocation cause: %v", err)
		}
		return
	}
	library := filepath.Join(t.TempDir(), "boundary.dylib")
	compile := exec.Command("cc", "-dynamiclib", "-fobjc-arc", "-framework", "Foundation", "-framework", "Metal", "../../metal/testdata/boundary_faults.m", "-o", library)
	if out, err := compile.CombinedOutput(); err != nil {
		t.Fatalf("compile platform fixture: %v\n%s", err, out)
	}
	for _, mode := range []string{"no-device", "no-buffers"} {
		t.Run(mode, func(t *testing.T) {
			args := []string{"-test.run=^TestFuelPreparationPropagatesNativeRefusals$"}
			if dir := flag.Lookup("test.gocoverdir"); testing.CoverMode() != "" && dir != nil && dir.Value.String() != "" {
				args = append(args, "-test.gocoverdir="+dir.Value.String())
			}
			child := exec.Command(os.Args[0], args...)
			child.Env = append(os.Environ(), "DYLD_INSERT_LIBRARIES="+library, "SLAYER_METAL_BOUNDARY_FAULT="+mode)
			if out, err := child.CombinedOutput(); err != nil {
				t.Fatalf("%s: %v\n%s", mode, err, out)
			}
		})
	}
}
