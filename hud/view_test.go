package hud_test

import (
	"errors"
	"image"
	"math"
	"testing"

	"github.com/janearc/libtheme-css/css"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/hud"
	"github.com/janearc/slayer/hud/mocks"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"go.uber.org/mock/gomock"
)

// eye is a camera off to one side, above, turned toward the middle of
// the world and seeing in a subspace twice the world's scale.
func eye() maths.Subspace {
	from := maths.V(3, 2, 8)
	return maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, 0.5, 0)), Scale: 2}
}

// for a pinhole, a pinhole with terminal-shaped pixels and a lens, a
// point film.Look's ray through a pixel's middle passes through falls
// back on that pixel's middle: the projection is film's own, undone.
func TestAPointProjectsWhereTheCameraSeesIt(t *testing.T) {
	const w, h = 64, 40
	for name, sensor := range map[string]film.Sensor{
		"pinhole":  film.Pinhole{},
		"sextants": film.Pinhole{Wide: 1, Pixel: 0.75},
		"lens":     film.Lens{Focal: 35}, "none": nil} {
		view := hud.View{Eye: eye(), Sensor: sensor, Width: w,
			Height: h}
		pixels := [][2]int{{0, 0}, {13, 7}, {32, 20}, {63, 39}}
		for _, px := range pixels {
			ray := film.Look(view.Eye, sensor, px[0], px[1], w, h,
				maths.Band{})
			across, down, ahead, err := view.Project(ray.Point(7))
			wantAcross := (float64(px[0]) + 0.5) / w
			wantDown := (float64(px[1]) + 0.5) / h
			off := math.Max(math.Abs(across-wantAcross),
				math.Abs(down-wantDown))
			if err != nil || !ahead || off > 1e-9 {
				t.Errorf("%s pixel %v: %v, %v, %v, %v",
					name, px, across, down, ahead, err)
			}
		}
	}
}

// a point behind the camera, or level with it, is not ahead; a sensor
// that is neither a pinhole nor a lens, and a view of no size, cannot
// be projected through and say so.
func TestWhatCannotBeProjectedSaysSo(t *testing.T) {
	view := hud.View{Eye: eye(), Width: 10, Height: 10}
	behind := eye().Origin.Add(eye().Origin.Sub(maths.V(0, 0.5, 0)))
	if _, _, ahead, err := view.Project(behind); ahead || err != nil {
		t.Errorf("a point behind the eye: ahead %v, %v", ahead, err)
	}
	view.Sensor = fisheye{}
	if _, _, _, err := view.Project(maths.Vec{}); !errors.Is(err,
		hud.ErrHud) {
		t.Errorf("a fisheye for a sensor: %v", err)
	}
	view = hud.View{Eye: eye()}
	if _, _, _, err := view.Project(maths.Vec{}); !errors.Is(err,
		hud.ErrHud) {
		t.Errorf("a view of no size: %v", err)
	}
}

// theme is ink on a ground.
func theme() hud.Theme {
	sheet := css.New()
	sheet.Set("ink", srgb.MustHex("#f0e0c0").Swatch())
	sheet.Set("ground", srgb.MustHex("#202020").Swatch())
	sheet.Set("bright-black", srgb.MustHex("#7a7468").Swatch())
	sheet.Set("green", srgb.MustHex("#8fbf6a").Swatch())
	sheet.Set("yellow", srgb.MustHex("#e0b050").Swatch())
	return hud.NewTheme(sheet)
}

// pinned is a label on the world's middle, "here", in a view of a
// picture 80 by 48 pixels, on a grid of 40 by 16 cells.
func pinned(t *testing.T, view hud.View, occluded bool) hud.Cover {
	t.Helper()
	label := hud.Label{Point: maths.V(0, 0.5, 0), Words: hud.Words("here"),
		Ink: "ink", Occluded: occluded}
	cover, err := hud.Hud{Elements: []hud.Element{label}}.Cover(
		image.Rect(0, 0, 40, 16), 0, theme(), view)
	if err != nil {
		t.Fatal(err)
	}
	return cover
}

// a label lands where the camera puts its point: its first letter in
// the cell its point projects into, the rest after it.
func TestALabelLandsWhereItsPointIs(t *testing.T) {
	view := hud.View{Eye: eye(), Width: 80, Height: 48}
	across, down, _, _ := view.Project(maths.V(0, 0.5, 0))
	x, y := int(across*40), int(down*16)
	mask := pinned(t, view, false).Mask()
	if len(mask) != 4 || mask[0] != image.Pt(x, y) ||
		mask[3] != image.Pt(x+3, y) {
		t.Errorf("the label covers %v, its point is in %d, %d",
			mask, x, y)
	}
}

// a label whose point is behind the camera is not drawn.
func TestALabelBehindTheCameraIsHidden(t *testing.T) {
	from := maths.V(0, 0.5, 5)
	away := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, 0.5, 10))}
	view := hud.View{Eye: away, Width: 80, Height: 48}
	if mask := pinned(t, view, false).Mask(); len(mask) != 0 {
		t.Errorf("a label behind the camera covers %v", mask)
	}
}

// a label that asks to be hidden asks the scene once, along the ray
// from the eye to its point, short of the point: hidden when the scene
// says something is met, drawn when it says nothing is. one that does
// not ask never asks the scene.
func TestALabelIsHiddenByANearerHitWhenItAsks(t *testing.T) {
	from, point := eye().Origin, maths.V(0, 0.5, 0)
	far := point.Sub(from).Len()
	ray := maths.Ray{At: from, Dir: point.Sub(from).Unit()}
	for met, want := range map[bool]int{true: 0, false: 4} {
		scene := mocks.NewMockBlocker(gomock.NewController(t))
		scene.EXPECT().Hit(ray, 0.0, far*(1-1e-6)).Return(far/2, met)
		view := hud.View{Eye: eye(), Width: 80, Height: 48,
			Blocker: scene}
		if got := len(pinned(t, view, true).Mask()); got != want {
			t.Errorf("the scene meets something %v: %d cells",
				met, got)
		}
	}
	quiet := mocks.NewMockBlocker(gomock.NewController(t))
	view := hud.View{Eye: eye(), Width: 80, Height: 48, Blocker: quiet}
	if got := len(pinned(t, view, false).Mask()); got != 4 {
		t.Errorf("a label that does not ask covers %d cells", got)
	}
}

// in a real world, a ball between the eye and the point hides a label
// that asks, a ball beyond the point does not, nor does a ball hidden
// from the picture; a ball drawn at half scale is met where it is.
func TestTheWorldHidesALabelBehindIt(t *testing.T) {
	from, point := eye().Origin, maths.V(0, 0.5, 0)
	between := from.Add(point.Sub(from).Scale(0.5))
	beyond := point.Sub(from.Sub(point).Scale(0.5))
	for name, each := range map[string]struct {
		at     maths.Vec
		scale  float64
		hidden bool
		want   int
	}{"between": {between, 1, false, 0}, "beyond": {beyond, 1, false, 4},
		"between, hidden": {between, 1, true, 4},
		"between, half":   {between, 0.5, false, 0}} {
		world := thing.NewWorld()
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.5},
			Motion: thing.Still(maths.Subspace{Origin: each.at,
				Scale: each.scale}),
			Attrs: thing.Attrs{Hidden: each.hidden}})
		view := hud.View{Eye: eye(), Width: 80, Height: 48,
			Blocker: hud.Seen(world, 0)}
		if got := len(pinned(t, view, true).Mask()); got != each.want {
			t.Errorf("a ball %s: %d cells, want %d",
				name, got, each.want)
		}
	}
}

// fisheye is a sensor that is not a pinhole: every pixel looks ahead.
type fisheye struct{}

// Ray is straight ahead, whatever the pixel.
func (fisheye) Ray(int, int, int, int, maths.Band) maths.Ray {
	return maths.Ray{Dir: maths.V(0, 0, -1)}
}

// a label's roles are asked of the theme even when it is not seen: one
// behind the camera in a role the theme has not is an error the first
// frame, not when it comes into view.
func TestAHiddenLabelsRolesAreStillChecked(t *testing.T) {
	label := hud.Label{Point: eye().Origin.Add(maths.V(0, 0, 50)),
		Words: hud.Words("x"), Ink: "mauve"}
	view := hud.View{Eye: eye(), Width: 80, Height: 48}
	_, err := hud.Hud{Elements: []hud.Element{label}}.Cover(
		image.Rect(0, 0, 40, 16), 0, theme(), view)
	if !errors.Is(err, hud.ErrHud) {
		t.Errorf("a hidden label in an unknown role: %v", err)
	}
}

// a label that asks to be hidden behind the scene, in a view with no
// scene to ask, is drawn.
func TestALabelWithNoSceneToAskIsDrawn(t *testing.T) {
	view := hud.View{Eye: eye(), Width: 80, Height: 48}
	if got := len(pinned(t, view, true).Mask()); got != 4 {
		t.Errorf("an occluded label with no scene covers %d cells", got)
	}
}
