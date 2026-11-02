package terrain

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

// kilauea is the usgs's 2019 model of kīlauea on this machine's shelf,
// unpacked, or the test is skipped: it is 1.9 gb and never in the
// repository.
func kilauea(t *testing.T) *Raster {
	t.Helper()
	shelf := os.Getenv("SLAYER_SHELF")
	path := filepath.Join(shelf, "unpacked", "kil_2019dem", "kil_2019dem.tif")
	if shelf == "" {
		t.Skip("no SLAYER_SHELF")
	}
	if _, err := os.Stat(path); err != nil {
		t.Skipf("the kīlauea model is not unpacked on the shelf: %v", err)
	}
	raster, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { raster.Close() })
	return raster
}

// the model is as its tags and its world file say: 59,405 by 25,350
// samples a metre apart, its first corner at 251,015.7644 e and
// 2,160,186.5221 n (the world file gives the first sample's middle,
// half a metre in from that).
func TestKilaueaIsWhereItsFilesSay(t *testing.T) {
	raster := kilauea(t)
	across, down := raster.Size()
	west, north, spacing := raster.Corner()
	near := func(got, want float64) bool { return math.Abs(got-want) < 1e-6 }
	if across != 59405 || down != 25350 || spacing != 1 ||
		!near(west, 251015.7644) || !near(north, 2160186.5221) {
		t.Errorf("%d by %d, %g apart, corner %f e %f n", across, down,
			spacing, west, north)
	}
}

// kīlauea iki's crater, a window 600 m across about 264.8 km e and
// 2,147.8 km n: every height inside the model's own range (its
// metadata: -1.02 m to 1,305.66 m), and the crater a hundred metres
// and more below its rim.
func TestKilaueaIkiIsACrater(t *testing.T) {
	raster := kilauea(t)
	window, err := raster.Window(264500, 2148100, 61, 61, 10)
	if err != nil {
		t.Fatal(err)
	}
	lowest, highest := 1e9, -1e9
	for down := range 61 {
		for across := range 61 {
			height := window.At(across, down)
			lowest, highest = min(lowest, height), max(highest, height)
		}
	}
	t.Logf("kīlauea iki window: %.1f m to %.1f m", lowest, highest)
	if lowest < -1.03 || highest > 1305.67 || highest-lowest < 100 {
		t.Errorf("heights %.1f to %.1f m", lowest, highest)
	}
}
