package colour_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/colour"
	"github.com/janearc/slayer/ninjafilm/data"
)

// dimPicture is a square of the dim grey with a star at its middle.
func dimPicture(star maths.Light) []maths.Light {
	lights := make([]maths.Light, rowsInATest*rowsInATest)
	for place := range lights {
		lights[place] = maths.Grey(baseLight)
	}
	lights[rowsInATest/pair*rowsInATest+rowsInATest/pair] = star
	return lights
}

// noNumber is whether any part of a light is no number.
func noNumber(light maths.Light) bool {
	return math.IsNaN(light.R) || math.IsNaN(light.G) || math.IsNaN(light.B)
}

// An infinite star bloomed at a radius of one spills to every pixel
// within its reach. Its spill is infinite. Nothing is no number. What is
// past the reach is as it was.
func TestABloomOfOneInfinitePixelHasNoNaN(t *testing.T) {
	size, middle := rowsInATest, rowsInATest/pair
	reach := int(data.Picked("bloom-reach-sigmas"))
	endless := maths.Grey(math.Inf(whole))
	lights := dimPicture(endless)
	got := colour.Bloom(lights, size, size, whole, whole)
	for place, light := range got {
		if noNumber(light) {
			t.Fatalf("pixel %d of the bloom is %v", place, light)
		}
		across, down := place%size-middle, place/size-middle
		if max(across, -across) <= reach && max(down, -down) <= reach {
			if !math.IsInf(light.G, whole) {
				t.Errorf("pixel %d within reach is %v, want infinite", place, light)
			}
		} else if light != lights[place] {
			t.Errorf("pixel %d past reach is %v, want %v", place, light, lights[place])
		}
	}
}

// An infinite part of one colour spills only that part.
func TestABloomOfAnInfiniteColourKeepsItsHue(t *testing.T) {
	size, middle := rowsInATest, rowsInATest/pair
	endlessRed := maths.RGB(math.Inf(whole), none, none)
	got := colour.Bloom(dimPicture(endlessRed),
		size, size, whole, whole)
	for place, light := range got {
		if noNumber(light) {
			t.Fatalf("pixel %d of the bloom is %v", place, light)
		}
	}
	beside := got[middle*size+middle+whole]
	if !math.IsInf(beside.R, whole) || beside.G != baseLight || beside.B != baseLight {
		t.Errorf("beside the star: %v, want infinite red and the rest as it was", beside)
	}
}

// A radius whose weights are zero, and an exposure whose reciprocal is
// infinite, each leave an infinite star bloomed without a NaN.
func TestABloomOfAnInfiniteStarAtTheEdgesOfTheFloatsHasNoNaN(t *testing.T) {
	size := rowsInATest
	star := maths.Grey(math.Inf(whole))
	for name, got := range map[string][]maths.Light{
		"a radius of the smallest float": colour.Bloom(dimPicture(star), size, size,
			whole, math.SmallestNonzeroFloat64),
		"an exposure of the smallest float": colour.Bloom(dimPicture(star), size,
			size, math.SmallestNonzeroFloat64, whole),
	} {
		for place, light := range got {
			if noNumber(light) {
				t.Fatalf("%s: pixel %d is %v", name, place, light)
			}
		}
	}
}

// A pixel that is no number spills nothing. Its neighbours are as they
// were. It stays no number, which Encode draws as black.
func TestANaNPixelDoesNotSpreadInABloom(t *testing.T) {
	size, middle := rowsInATest, rowsInATest/pair
	notANumber := maths.Grey(math.NaN())
	lights := dimPicture(notANumber)
	lights[none] = maths.Grey(starLight)
	got := colour.Bloom(lights, size, size, whole, whole)
	for place, light := range got {
		if place != middle*size+middle && noNumber(light) {
			t.Errorf("pixel %d is %v: the NaN spread", place, light)
		}
	}
	if beside := got[middle*size+middle+whole]; beside != lights[middle*size+middle+whole] {
		t.Errorf("beside the NaN pixel: %v, want it as it was", beside)
	}
	if colour.Encode(got[middle*size+middle].R) != none {
		t.Errorf("the NaN pixel is not drawn as black")
	}
}
