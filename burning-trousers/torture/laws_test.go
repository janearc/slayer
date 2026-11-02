package torture_test

import (
	"fmt"
	"math"
	"runtime"
	"testing"

	"github.com/janearc/slayer/render/frame"
)

// seeds is how many scenes each law is held to, lit and unlit alike.
const seeds = 200

// broken is what a law found wrong, kept so a test can say the first
// few and how many there were in all.
type broken struct{ found []string }

// add is one thing wrong, in one scene's one pixel.
func (law *broken) add(seed uint64, x int, format string, args ...any) {
	where := fmt.Sprintf("scene %d, look %d: ", seed, x)
	law.found = append(law.found, where+fmt.Sprintf(format, args...))
}

// say tells the test the first five things wrong and the count.
func (law *broken) say(t *testing.T, name string) {
	t.Helper()
	for _, found := range law.found[:min(5, len(law.found))] {
		t.Error(found)
	}
	if len(law.found) > 0 {
		t.Errorf("%s: broken %d times in all", name, len(law.found))
	}
}

// channels is a pixel's light, part by part.
func channels(pixel frame.Pixel) [3]float64 {
	return [3]float64{float64(pixel.R), float64(pixel.G), float64(pixel.B)}
}

// Every pixel of every scene is a number: no NaN and no infinity, lit
// or unlit.
func TestNoPixelIsNaN(t *testing.T) {
	var law broken
	for _, lit := range []bool{false, true} {
		for seed := range uint64(seeds) {
			for x, pixel := range drawn(t, arranged(seed, lit), 1) {
				for _, part := range channels(pixel) {
					if math.IsNaN(part) || math.IsInf(part, 0) {
						law.add(seed, x, "light is %v", part)
					}
				}
			}
		}
	}
	law.say(t, "no NaN")
}

// No pixel of any scene has less than no light.
func TestNoLightIsNegative(t *testing.T) {
	var law broken
	for _, lit := range []bool{false, true} {
		for seed := range uint64(seeds) {
			for x, pixel := range drawn(t, arranged(seed, lit), 1) {
				for _, part := range channels(pixel) {
					if part < 0 {
						law.add(seed, x, "light is %g", part)
					}
				}
			}
		}
	}
	law.say(t, "no negative light")
}

// A scene lit by its sky alone, with nothing glowing and every surface
// giving back no more than arrives, is nowhere brighter than the sky.
func TestNoLightIsMade(t *testing.T) {
	var law broken
	for seed := range uint64(seeds) {
		made := arranged(seed, false)
		sky := [3]float64{made.Sky().R, made.Sky().G, made.Sky().B}
		for x, pixel := range drawn(t, made, 1) {
			for part, light := range channels(pixel) {
				if light > sky[part]*(1+1e-6) {
					law.add(seed, x, "light %.6f under a sky of %.6f",
						light, sky[part])
				}
			}
		}
	}
	law.say(t, "no light made")
}

// The same scene draws the same frame bit for bit: drawn twice, and drawn on
// one core against all of them.
func TestTheSameSceneDrawsTheSameFrame(t *testing.T) {
	var law broken
	for seed := range uint64(seeds) {
		made := arranged(seed, true)
		first, again := drawn(t, made, 1), drawn(t, made, 1)
		cores := runtime.GOMAXPROCS(1)
		alone := drawn(t, made, 1)
		runtime.GOMAXPROCS(cores)
		for x := range first {
			if first[x] != again[x] || first[x] != alone[x] {
				law.add(seed, x, "drawn %v, then %v, alone %v", first[x],
					again[x], alone[x])
			}
		}
	}
	law.say(t, "the same frame")
}

// A scene and its units scaled together draw the same picture: every
// length a thousandth or a thousand times, with the lamps and the gas
// made to land and keep the same light.
func TestScaleChangesNothing(t *testing.T) {
	for _, by := range []float64{1e-3, 1e3} {
		var law broken
		for seed := range uint64(seeds) {
			made := arranged(seed, true)
			whole, scaled := drawn(t, made, 1), drawn(t, made, by)
			for x := range whole {
				near, far := channels(whole[x]), channels(scaled[x])
				for part := range near {
					gap := math.Abs(near[part] - far[part])
					if gap > 1e-4*max(near[part], far[part])+1e-6 {
						law.add(seed, x, "%.6f at scale 1, %.6f at %g",
							near[part], far[part], by)
					}
				}
			}
		}
		law.say(t, fmt.Sprintf("scale %g", by))
	}
}
