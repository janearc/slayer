package trace

import (
	_ "embed"
	"image"
	"testing"
	"time"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
)

//go:embed testdata/sections.yaml
var sectionsRaw []byte

// sectionBox is a section of a frame as sections.yaml writes it.
type sectionBox struct {
	Name                     string
	Left, Top, Right, Bottom int
}

// box is the section as a rectangle.
func (given sectionBox) box() image.Rectangle {
	return image.Rect(given.Left, given.Top, given.Right, given.Bottom)
}

// sectionData is sections.yaml: the frame, the eye, the rays a pixel, the
// sections that tile the frame and those that are refused.
type sectionData struct {
	Frame       struct{ Width, Height int }
	Eye         struct{ Across, Height, Depth, Wide float64 }
	Rays        []int
	RefusedRays int `yaml:"refused-rays"`
	Tiles       []sectionBox
	Refused     []sectionBox
}

// sectionFixture reads sections.yaml.
func sectionFixture(test *testing.T) sectionData {
	test.Helper()
	var data sectionData
	if err := yamlfile.Read("sections.yaml", sectionsRaw, &data); err != nil {
		test.Fatal(err)
	}
	return data
}

// atStart is the moment the sections' pictures are taken at. It is
// defined as the start of the film, zero.
const atStart time.Duration = 0

// none is how many pixels, or how much light, there is none of. It is
// defined as zero.
const none = 0

// quarter is the share of a frame's pixels that must be lit, as the
// number of equal parts it is one of. It is defined as four parts.
const quarter = 4

// sectionEye is where the sections' pictures are taken from.
func sectionEye(data sectionData) (maths.Subspace, film.Sensor) {
	origin := maths.V(data.Eye.Across, data.Eye.Height, data.Eye.Depth)
	where := maths.NewSubspace(origin, maths.Identity, none)
	return where, film.Pinhole{Wide: data.Eye.Wide}
}

// a section that is empty or reaches outside the frame is refused, and
// says which.
func TestASectionOutsideTheFrameIsRefused(t *testing.T) {
	data := sectionFixture(t)
	eye, lens := sectionEye(data)
	for _, refused := range data.Refused {
		box := refused.box()
		if _, err := sampledIn(smoking(), eye, lens, atStart,
			data.Frame.Width, data.Frame.Height, data.RefusedRays,
			box); err == nil {
			t.Errorf("a section in %v of a %d by %d frame was drawn: %s",
				box, data.Frame.Width, data.Frame.Height, refused.Name)
		}
	}
}
