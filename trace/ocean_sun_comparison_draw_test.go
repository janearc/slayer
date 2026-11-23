//go:build draw

package trace

import (
	"math"
	"os"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// The sun study's picture, 161 by 91: its middle pixel looks straight
// along the eye's way.
const sunStudyWidth, sunStudyHeight = 161, 91

// sunStudyToward is the way to the study's sun.
var sunStudyToward = maths.V(.35, 1, -1)

// sunStudyPicture is three white balls two metres under a level sea of
// water that takes 0.08, 0.04 and 0.02 a metre, under a sun of eight,
// seen from in the water eight metres before the middle ball's nearest
// point; with the sun brought through the sea's mean plane, or by the
// straight way, where the sea's own surface hides it.
func sunStudyPicture(t *testing.T, meanPlane bool) *frame.Frame {
	t.Helper()
	water, err := thing.NewClearMedium("synthetic-study-water",
		maths.RGB(.08, .04, .02), 1)
	if err != nil {
		t.Fatal(err)
	}
	surface, err := shape.NewHeightField([]float64{0, 0, 0, 0}, 2, 100)
	if err != nil {
		t.Fatal(err)
	}
	boundary := &thing.Thing{Shape: surface, Material: &thing.Material{
		Name: "synthetic flat water", Through: 1, Gloss: 1,
		Index: waterIndex, ClearMedium: water}}
	world := thing.NewWorld()
	world.Things = append(world.Things, boundary)
	for _, p := range []maths.Vec{maths.V(-2, -2, 0), maths.V(0, -2, 1),
		maths.V(2, -2, 2)} {
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
			Motion: resting(p), Material: &thing.Material{
				Name: "white reference sphere", Colour: maths.Grey(.8)}})
	}
	sun, err := emit.NewDistant(sunStudyToward, maths.Grey(8))
	if err != nil {
		t.Fatal(err)
	}
	fill, err := emit.NewDistant(maths.Up, maths.Light{})
	if err != nil {
		t.Fatal(err)
	}
	world.Lamps = []thing.Lamp{sun, fill}
	world.OceanSurface = &thing.OceanSurface{Boundary: boundary,
		ExtentMetres: 100, MetresPerUnit: 1,
		SunRoute: thing.OceanSunRouteGround, SunLampIndex: 0,
		FillLampIndex: 1, DirectSunMeanPlane: meanPlane}
	made, err := RenderWithInitialMedia(world,
		mediumEye(maths.V(0, -2, -8), maths.V(0, 0, 1)),
		film.Pinhole{Wide: 55 * maths.Degree}, 0, sunStudyWidth,
		sunStudyHeight,
		[]InitialMedium{{Index: waterIndex, ClearMedium: water}})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// sunStudyWant is the middle pixel with the sun brought through the
// sea, worked by hand. The middle look meets the middle ball at 0, -2,
// 0, two metres down, where the ball faces back along z.
//
// The sun stands at a cosine c = 1 / |0.35, 1, -1| from straight up, and
// by Snell's law at a sine of sqrt(1 - c^2) / n in the water. Its eight
// lands on the level sea by c, passes by what fresnel does not mirror,
// spreads over a beam wider by the water's cosine w, and comes down
// 2 / w metres of water. The ball's matte white of 0.8 sends 0.8 / pi of
// that by the cosine it lands on the ball at: the refracted way keeps
// the sun's heading, so its part along minus z is its sine times
// 1 / |0.35, -1|. Eight metres of water then stand before the eye.
func sunStudyWant() maths.Light {
	cosine := 1 / sunStudyToward.Len()
	sine := math.Sqrt(1-cosine*cosine) / waterIndex
	inWater := math.Sqrt(1 - sine*sine)
	passes := 8 * (1 - mirrorShare(cosine, waterIndex)) * cosine / inWater
	square := sine / math.Hypot(.35, 1)
	channel := func(takes float64) float64 {
		return 0.8 / math.Pi * passes * square *
			math.Exp(-takes*2/inWater) * math.Exp(-takes*8)
	}
	return maths.RGB(channel(.08), channel(.04), channel(.02))
}

// TestOceanSunComparisonDraw is a small study of the sun's way to a
// thing under the tagged sea: the straight way, which the sea's own
// surface hides, against the opted-in way through its mean plane. By the
// straight way the middle ball is black; through the mean plane its
// middle pixel is the worked value, to a ten thousandth of it: the light
// is asked a hair off the ball, and a pixel is single precision.
//
// The pictures are written only where SLAYER_OCEAN_SUN_COMPARISON_OUT
// names a folder; the pixels are held in every run.
func TestOceanSunComparisonDraw(t *testing.T) {
	off, on := sunStudyPicture(t, false), sunStudyPicture(t, true)
	offMiddle := off.At(sunStudyWidth/2, sunStudyHeight/2)
	onMiddle := on.At(sunStudyWidth/2, sunStudyHeight/2)
	if offMiddle.Light() != (maths.Light{}) || offMiddle.Cover != 1 {
		t.Errorf("by the straight way the middle ball is %+v, want it "+
			"met and black", offMiddle)
	}
	want := sunStudyWant()
	if !(want.B > want.R && want.R > 0.01) {
		t.Fatalf("the study's own worked value is %v", want)
	}
	if !pixelNear(onMiddle, want, 1e-4*want.R) {
		t.Errorf("through the mean plane the middle ball is %+v, want %v",
			onMiddle, want)
	}
	dir := os.Getenv("SLAYER_OCEAN_SUN_COMPARISON_OUT")
	writeClearInscatterImage(t, dir, "legacy-shadow.png", off)
	writeClearInscatterImage(t, dir, "mean-plane-sun.png", on)
}

// Anti: the worked value is the refracted sun's and no other's. The
// black of the straight way does not pass for it, and neither does a
// pixel off the balls, where the look runs on through water to nothing.
func TestOceanSunWorkedValueIsNotTheStraightWaysOrOpenWaters(t *testing.T) {
	want := sunStudyWant()
	off := sunStudyPicture(t, false).At(sunStudyWidth/2, sunStudyHeight/2)
	open := sunStudyPicture(t, true).At(0, sunStudyHeight-1)
	for name, pixel := range map[string]frame.Pixel{
		"the straight way": off, "open water": open} {
		if pixelNear(pixel, want, 1e-4*want.R) {
			t.Errorf("%s drew %+v, which passes for the refracted "+
				"sun's %v", name, pixel, want)
		}
	}
}
