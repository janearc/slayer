package gpu

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies/scrooge"
)

// The cylinder's arithmetic is checked on production code, not on a copy:
// the host's in render/beamstep (cylinder_test.go), and the kernel's on a
// device (beam_apart_draw_test.go).

// Asked for, the header's look says one in its fourth number; the
// numbers before it are what they were.
func TestTheHeaderSaysBeamRegionWhenScroogeIsAsked(t *testing.T) {
	for asked, want := range map[string]int32{"1": 1, "0": 0, "": 0,
		"yes": 0} {
		t.Setenv(scrooge.Variable, asked)
		encoded := newScene()
		encoded.camera(maths.Local(), film.Pinhole{}, 1, 1)
		got := headerInts(encoded, "look")
		if got[3] != want {
			t.Errorf("with %q look.w is %d, want %d", asked, got[3], want)
		}
		if got[0] != 0 || got[1] != 0 || got[2] != 0 {
			t.Errorf("with %q the other numbers of look moved: %v",
				asked, got)
		}
	}
}

// regionMarks are the pieces of the kernel's text that make the beam
// region's two ways: the cylinders where look.w says one, the boxes'
// region as the other, and the last stretch for the union of the rest.
var regionMarks = []string{
	"world.header->look.w == whole_one",
	"return beam_march_boxes(world, at, way, far)",
	"static BeamMarch beam_march_boxes(",
	"result.step[last] = min(result.step[last], step)",
}

// regionLacks is the marks the text does not hold.
func regionLacks(text string) []string {
	var lacking []string
	for _, mark := range regionMarks {
		if !strings.Contains(text, mark) {
			lacking = append(lacking, mark)
		}
	}
	return lacking
}

// The kernel's text takes the cylinders only where look.w says one, keeps
// the boxes' region as the other way, and keeps the last stretch for the
// union of the rest. This reads the text, so a rename shows here; what
// the two ways draw is the device test's.
func TestTheKernelKeepsTheBoxRegionWhenNotAsked(t *testing.T) {
	for _, mark := range regionLacks(kernelText(t)) {
		t.Errorf("the kernel's text lacks %q", mark)
	}
}

// Anti: a kernel text that always takes the cylinders, with no look.w
// to say so, is reported as lacking that mark and only it.
func TestAKernelThatAlwaysTakesCylindersIsReported(t *testing.T) {
	always := doctored(t, kernelText(t), "world.header->look.w == whole_one",
		"true")
	lacking := regionLacks(always)
	if len(lacking) != 1 || lacking[0] != regionMarks[0] {
		t.Errorf("a kernel with no look.w switch gave %v", lacking)
	}
}
