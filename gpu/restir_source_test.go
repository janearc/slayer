//go:build darwin

package gpu

import (
	"strings"
	"testing"
)

// each place restir changes is in the kernel's text once, and the
// edited text has the four changes.
func TestRestirEditsAreFoundOnceEach(t *testing.T) {
	text, err := restirKernel(sourceWith(0, probeVolumes))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"int restir_on;", "falling_restir(world",
		"world.restir_on = 0;", "world.restir_on = 1;",
		"kernel void restir_gbuffer", "kernel void restir_resample",
		"kernel void restir_spatial"} {
		if strings.Count(text, want) < 1 {
			t.Errorf("the restir kernel lacks %q", want)
		}
	}
}

// anti: a kernel text that has changed under restir is refused by name
// and not drawn with a place missing.
func TestRestirRefusesAKernelThatMovedUnderIt(t *testing.T) {
	text := strings.Replace(sourceWith(0, probeVolumes),
		"\tworld.shared = sky_share(header, pixel);\n", "", 1)
	if _, err := restirKernel(text); err == nil ||
		!strings.Contains(err.Error(), "trace kernel names its reservoir") {
		t.Errorf("a kernel with a place gone was taken: %v", err)
	}
	twice := sourceWith(0, probeVolumes)
	twice += "\tworld.shared = sky_share(header, pixel);\n"
	if _, err := restirKernel(twice); err == nil {
		t.Error("a kernel with a place twice was taken")
	}
}

// every mode but restir's compiles a kernel with no word of restir in it.
func TestOtherModesCompileNoRestir(t *testing.T) {
	encoded := &scene{}
	for _, mode := range []traversal{softwareTraversal, nativeShadowTraversal,
		nativeTraversal, cachedTraversal, fastTraversal, fasterTraversal} {
		lower := strings.ToLower(encoded.source(mode))
		if strings.Contains(lower, "restir") {
			t.Errorf("mode %d's kernel text names restir", mode)
		}
	}
	if !strings.Contains(sourceWith(0, probeVolumes), "falling(") {
		t.Fatal("the legacy text has no falling")
	}
}
