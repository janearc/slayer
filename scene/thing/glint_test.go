package thing

import (
	"math"
	"testing"
)

// The numbers the glint tests count in, each true by definition.
const (
	glintX          = nought // defined: the place of the x axis in a direction
	glintY          = unity  // defined: the place of the y axis, the way up
	glintZ          = pair   // defined: the place of the z axis in a direction
	glintHalf       = 0.5    // defined: half, the middle of a cell
	glintDimensions = 3      // defined: the axes of space, a direction's length
)

// glintWidths names the GGX widths the bound test tries, the narrowest
// first, and glintViews the cosines of the looks it tries them from.
var (
	glintWidths = []string{"glint-width-polished", "glint-width-fine",
		"glint-width-medium", "glint-width-whole"}
	glintViews = []string{"glint-view-face", "glint-view-slanted",
		"glint-view-grazing"}
)

// glintView is one eye and one surface whose lobe is summed: the way to
// the eye, its cosine, the GGX width, and how many rings and turns the
// hemisphere is cut into.
type glintView struct {
	eye          [glintDimensions]float64
	viewed       float64
	alpha        float64
	rings, turns float64
}

// cell is the lobe of the lamp at tilt from straight up and turn around
// it, times the solid angle of its cell.
func (view glintView) cell(tilt, turn float64) float64 {
	lamp := [glintDimensions]float64{
		glintX: math.Sin(tilt) * math.Cos(turn),
		glintY: math.Cos(tilt),
		glintZ: math.Sin(tilt) * math.Sin(turn),
	}
	half := [glintDimensions]float64{
		glintX: lamp[glintX] + view.eye[glintX],
		glintY: lamp[glintY] + view.eye[glintY],
		glintZ: lamp[glintZ] + view.eye[glintZ],
	}
	length := math.Sqrt(half[glintX]*half[glintX] +
		half[glintY]*half[glintY] + half[glintZ]*half[glintZ])
	facing := half[glintY] / length
	area := math.Sin(tilt) * (math.Pi / pair / view.rings) *
		(pair * math.Pi / view.turns)
	return Microfacet(facing, lamp[glintY], view.viewed, view.alpha) * area
}

// returned is how much of the light landing on a surface from every
// way its highlights send toward an eye looking at it from cosine
// viewed, for a GGX width alpha, before Fresnel's share: the lobe,
// which holds each lamp's cosine already, summed over the hemisphere by
// the midpoint rule.
func returned(t *testing.T, viewed, alpha float64) float64 {
	t.Helper()
	rings := pickedCount(t, "glint-rings")
	turns := pickedCount(t, "glint-turns")
	view := glintView{
		eye: [glintDimensions]float64{
			glintX: math.Sqrt(unity - viewed*viewed),
			glintY: viewed,
			glintZ: nought,
		},
		viewed: viewed,
		alpha:  alpha,
		rings:  float64(rings),
		turns:  float64(turns),
	}
	var sum float64
	for i := range rings {
		tilt := (float64(i) + glintHalf) / view.rings * math.Pi / pair
		for step := range turns {
			turn := (float64(step) + glintHalf) / view.turns * pair * math.Pi
			sum += view.cell(tilt, turn)
		}
	}
	return sum
}

// A rough surface never sends back more than lands on it: its lobe,
// summed over every way a lamp could stand, is at most one. Smith's
// shadowing takes more the rougher it is and the more slant the look,
// so a near-polished surface seen face on keeps nearly all of it.
func TestAHighlightNeverSendsBackMoreThanLands(t *testing.T) {
	bound := picked(t, "glint-return-bound")
	for _, widthName := range glintWidths {
		alpha := picked(t, widthName)
		for _, viewName := range glintViews {
			viewed := picked(t, viewName)
			if got := returned(t, viewed, alpha); !(got <= bound) {
				t.Errorf("width %v seen from cosine %v sends back %v", alpha,
					viewed, got)
			}
		}
	}
	face := picked(t, "glint-view-face")
	polished := picked(t, "glint-width-polished")
	floor := picked(t, "glint-polished-return-floor")
	if got := returned(t, face, polished); !(got >= floor) {
		t.Errorf("a near-polished surface seen face on sends back only %v", got)
	}
}

// The lobe is brightest where the lamp stands at the mirror way, and
// the smoother the surface the narrower and taller it is there.
func TestAHighlightPeaksAtTheMirrorWay(t *testing.T) {
	smooth := picked(t, "glint-smooth-width")
	rough := picked(t, "glint-rough-width")
	slant := picked(t, "glint-slant-cosine")
	offMirror := picked(t, "glint-off-mirror-facing")
	at := Microfacet(unity, slant, slant, smooth)
	off := Microfacet(offMirror, slant, slant, smooth)
	if !(at > off) {
		t.Errorf("at the mirror way %v, just off it %v", at, off)
	}
	if !(Microfacet(unity, slant, slant, smooth) >
		Microfacet(unity, slant, slant, rough)) {
		t.Error("the smooth peak is no taller than the rough")
	}
	if got := Microfacet(nought, slant, slant, rough); got != nought {
		t.Errorf("a half way edge on gives %v", got)
	}
	if got := Microfacet(unity, slant, slant, nought); got != nought {
		t.Errorf("a polished surface gives %v, a point no picture holds", got)
	}
}
