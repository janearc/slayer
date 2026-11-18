//go:build darwin

package hillside

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// nativeLine is the line a set carries under scene: when the gpu's own
// ray queries find its meshes and shadows.
const nativeLine = "\n  renderer: gpu-native\n"

// nativeSets are the sets whose files say renderer: gpu-native and that
// hillside draws, with the folder each lies in.
var nativeSets = []string{
	"doc/city-clustered-10000.yaml",
	"doc/city-clustered-green-10000.yaml",
	"doc/city-clustered-night-10000.yaml",
	"doc/city-grid-10000.yaml",
	"doc/fire-cluster.yaml",
	"doc/fire-fuel.yaml",
	"doc/fire-night.yaml",
	"doc/fire-study.yaml",
	"doc/fire-woodland.yaml",
	"doc/golden-hour-site.yaml",
	"doc/guide-star-deck.yaml",
	"doc/guide-star-mist.yaml",
	"doc/macrostructures.yaml",
	"hillside/testdata/quality-summit-golden.yaml",
	"hillside/testdata/quality-summit-moon.yaml",
	"hillside/testdata/quality-summit-yard.yaml",
	"doc/space-hubble.yaml",
	"doc/space-spitzer.yaml",
	"doc/space-webb.yaml",
	"sets/kilauea-iki-forest.yaml",
	"sets/melancholia.yaml",
	"sets/moonlake.yaml",
}

// nativeTwins are sets that say renderer: gpu-native and are the same
// file as a set in nativeSets but for the film's width, height and rays,
// so they are not drawn again; the coverage test reads both files and
// requires that they differ in no other line.
var nativeTwins = map[string]string{
	"hillside/testdata/quality-summit-golden-4k.yaml":      "hillside/testdata/quality-summit-golden.yaml",
	"hillside/testdata/quality-summit-golden-preview.yaml": "hillside/testdata/quality-summit-golden.yaml",
	"hillside/testdata/quality-summit-moon-4k.yaml":        "hillside/testdata/quality-summit-moon.yaml",
	"hillside/testdata/quality-summit-moon-preview.yaml":   "hillside/testdata/quality-summit-moon.yaml",
	"hillside/testdata/quality-summit-yard-preview.yaml":   "hillside/testdata/quality-summit-yard.yaml",
}

// nativeRefused are the sets that say renderer: gpu-native and that
// hillside does not draw yet: it refuses them by name before staging,
// whatever the renderer. Each is listed with the refusal hillside says.
var nativeRefused = map[string]string{
	"doc/corona.yaml":                     "../doc/corona.yaml:23:9: thing.kind: star is read and not yet drawn",
	"set/testdata/keyhole-to-ground.yaml": "../set/testdata/keyhole-to-ground.yaml:28:3: scene.date: 2026-06-21 is read and not yet drawn",
	"doc/orbital-and-lantern.yaml":        "../doc/orbital-and-lantern.yaml:24:9: thing.kind: lantern is read and not yet drawn",
	"set/testdata/webb-looking-out.yaml":  "../set/testdata/webb-looking-out.yaml:21:3: scene.frame is read and not yet drawn",
}

// TestEverySetSayingNativeIsDrawn requires that each file in sets and doc
// that says renderer: gpu-native is in nativeSets, or in nativeTwins as
// the same file as one that is, or in nativeRefused with its reason, so a
// set given the line later is not left undrawn.
func TestEverySetSayingNativeIsDrawn(t *testing.T) {
	listed := map[string]bool{}
	for _, name := range nativeSets {
		listed[name] = true
	}
	for name := range nativeRefused {
		listed[name] = true
	}
	for name, drawn := range nativeTwins {
		listed[name] = true
		if !listed[drawn] || !sameButFilm(t, name, drawn) {
			t.Errorf("%s is not %s but for its film's size and rays", name, drawn)
		}
	}
	for _, folder := range []string{"sets", "doc"} {
		files, err := filepath.Glob(filepath.Join("..", folder, "*.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range files {
			authored, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			name := folder + "/" + filepath.Base(path)
			if strings.Contains(string(authored), nativeLine) && !listed[name] {
				t.Errorf("%s says renderer: gpu-native and is in neither "+
					"nativeSets, nativeTwins nor nativeRefused, so nothing draws it", name)
			}
		}
	}
}

// sameButFilm is whether the sets called first and second, read from the
// repository root, have the same lines but for the film's width, height
// and rays.
func sameButFilm(t *testing.T, first, second string) bool {
	t.Helper()
	return withoutFilmSize(t, first) == withoutFilmSize(t, second)
}

// withoutFilmSize is the set called name with the lines that say the film's
// width, height and rays taken out.
func withoutFilmSize(t *testing.T, name string) string {
	t.Helper()
	authored, err := os.ReadFile(filepath.Join("..", name))
	if err != nil {
		t.Fatal(err)
	}
	var kept []string
	for _, line := range strings.Split(string(authored), "\n") {
		said := strings.TrimSpace(line)
		if strings.HasPrefix(said, "width:") || strings.HasPrefix(said, "height:") ||
			strings.HasPrefix(said, "rays:") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}
