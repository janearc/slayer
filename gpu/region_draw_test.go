//go:build draw && darwin

package gpu

import (
	"image"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

func TestSoftwareRegionsMatchTheFullPicture(t *testing.T) {
	for _, tent := range []bool{false, true} {
		for _, side := range []int{1, 2} {
			world, eye := underASky(t)
			if tent {
				world.Filter = thing.TentFilter
			}
			lens := film.Pinhole{Wide: 50 * maths.Degree}
			whole, _, err := SampledMeasured(world, eye, lens, 0, 16, 12, side)
			if err != nil {
				t.Fatal(err)
			}
			for _, rect := range []image.Rectangle{image.Rect(0, 0, 1, 1), image.Rect(3, 4, 11, 9), image.Rect(14, 10, 16, 12)} {
				part, profile, err := SampledRegion(Software, world, eye, lens, 0, 16, 12, side, rect)
				if err != nil {
					t.Fatal(err)
				}
				if part.W != rect.Dx() || part.H != rect.Dy() {
					t.Fatalf("region %v has size %dx%d", rect, part.W, part.H)
				}
				if profile.Device == "" || profile.InputBytes <= 0 || profile.SampleBytes <= 0 {
					t.Fatalf("region profile omits the device or actual work: %+v", profile)
				}
				for y := 0; y < part.H; y++ {
					for x := 0; x < part.W; x++ {
						if got, want := part.At(x, y), whole.At(x+rect.Min.X, y+rect.Min.Y); got != want {
							t.Fatalf("tent %v side %d region %v pixel %d,%d: %v != %v", tent, side, rect, x, y, got, want)
						}
					}
				}
			}
		}
	}
}

func TestSoftwareRegionMatchesWholeAtSixtyFourRays(t *testing.T) {
	world, eye := underASky(t)
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	whole, _, err := SampledMeasured(world, eye, lens, 0, 2, 2, 64)
	if err != nil {
		t.Fatal(err)
	}
	part, _, err := SampledRegion(Software, world, eye, lens, 0, 2, 2, 64, image.Rect(1, 0, 2, 1))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := part.At(0, 0), whole.At(1, 0); got != want {
		t.Fatalf("region pixel %v differs from whole picture %v", got, want)
	}
}
