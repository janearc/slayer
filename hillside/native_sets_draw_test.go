//go:build draw && darwin

package hillside

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// nativeSmallWidth is the width, in pixels, the native sets are drawn at
// here: small, so the sets together stay under three minutes. It is picked;
// the time goes in building each scene, which no width shortens.
const nativeSmallWidth = "64"

// nativeSmallRays is the rays a side the native sets are drawn at here,
// the fewest.
const nativeSmallRays = "1"

// TestNativeRefusedSaysWhatHillsideSays requires that the reason each
// refused set is listed with is what hillside says when it stages the set.
func TestNativeRefusedSaysWhatHillsideSays(t *testing.T) {
	for name, reason := range nativeRefused {
		given := mustParse(t, "-set", filepath.Join("..", name))
		_, _, _, err := staged(given)
		if err == nil || err.Error() != reason {
			t.Errorf("%s: hillside says %v, listed as %q", name, err, reason)
		}
	}
}

// TestNativeSetsDrawnFinish draws each set that says renderer: gpu-native,
// small, and requires it to finish and write a picture. A hardware ray that
// never came back used to stall the draw for good; every Metal wait now
// gives up after a minute with an error naming the kernel and rows, so a
// stall shows here as a failure. It needs the shelf, so it skips with the
// other heavy set tests, without SLAYER_SHELF or in short mode.
func TestNativeSetsDrawnFinish(t *testing.T) {
	if os.Getenv("SLAYER_SHELF") == "" || testing.Short() {
		t.Skip("no shelf named by SLAYER_SHELF, or short")
	}
	requireCommandDevice(t)
	for _, name := range nativeSets {
		t.Run(name, func(t *testing.T) {
			drawNativeSet(t, name)
		})
	}
}

// drawNativeSet requires the set called name to say renderer: gpu-native,
// then draws it small through run and requires a picture that is not empty.
func drawNativeSet(t *testing.T, name string) {
	t.Helper()
	path := filepath.Join("..", name)
	authored, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(authored), nativeLine) {
		t.Fatalf("%s no longer says renderer: gpu-native, which is the "+
			"path this test draws", name)
	}
	var said, wrong bytes.Buffer
	output := filepath.Join(t.TempDir(), "native.png")
	args := []string{"-set", path, "-gpu", "-w", nativeSmallWidth,
		"-n", nativeSmallRays, "-out", output}
	if code := run(args, &said, &wrong); code != 0 {
		t.Fatalf("%s drawn native: exit %d: %s", name, code, wrong.String())
	}
	if info, err := os.Stat(output); err != nil || info.Size() == 0 {
		t.Fatalf("%s drawn native left no picture: %v", name, err)
	}
}
