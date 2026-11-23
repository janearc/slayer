//go:build draw

package trace

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

// The water-only control's picture, 161 by 91, so that its middle pixel
// looks straight down the eye's way, and the water's index.
const (
	skyControlWidth, skyControlHeight = 161, 91
	skyControlIndex                   = 1.333
)

// skyControlMiddles are the middle pixels of the water-only control:
// a look straight down from two metres under a level endless sea into
// water with nothing in it, under a sky and two lamps of no light, drawn
// without the sky's light scattered in the water and with it. The two
// pictures come back too. The sky's light is skyScale times the sky
// that skyControlMost is worked for: one for the control itself.
func skyControlMiddles(test *testing.T, skyScale float64) (without,
	with frame.Pixel, pictures map[string]*frame.Frame) {
	test.Helper()
	const (
		width, height = skyControlWidth, skyControlHeight
		index         = skyControlIndex
	)
	core, err := shape.NewHeightField([]float64{0, 0, 0, 0}, 2, 80)
	if err != nil {
		test.Fatal(err)
	}
	continuation, err := arid.NewInfiniteMeanContinuation(core, 0, 20)
	if err != nil {
		test.Fatal(err)
	}
	base, err := thing.NewClearMediumWithScattering("synthetic-water",
		maths.RGB(.01, .015, .025), maths.RGB(.08, .1, .12), .25,
		1, 8, 12)
	if err != nil {
		test.Fatal(err)
	}
	sampled, err := base.WithDiffuseSkyInscatter(16)
	if err != nil {
		test.Fatal(err)
	}
	render := func(medium *thing.ClearMedium) *frame.Frame {
		test.Helper()
		world := thing.NewWorld()
		boundary := &thing.Thing{Shape: continuation,
			Material: &thing.Material{Name: "synthetic water", Through: 1,
				Gloss: 1, Index: index, ClearMedium: medium}}
		world.Things = append(world.Things, boundary)
		sun, err := emit.NewDistant(maths.V(.35, .93675, 0),
			maths.Light{})
		if err != nil {
			test.Fatal(err)
		}
		fill, err := emit.NewDistant(maths.Up, maths.Light{})
		if err != nil {
			test.Fatal(err)
		}
		world.Lamps = []thing.Lamp{sun, fill}
		world.Sky = sky.Gradient{
			Horizon: maths.RGB(.3, .5, .75).Scale(skyScale),
			Zenith:  maths.RGB(.5, .72, 1).Scale(skyScale),
			Ground:  maths.Light{}}
		world.OceanSurface = &thing.OceanSurface{Boundary: boundary,
			Mean: maths.V(0, 0, 0), ExtentMetres: 80,
			MetresPerUnit: 1, DirectSunMeanPlane: true,
			InfiniteMeanPlane: true, SunRoute: thing.OceanSunRouteGround,
			SunLampIndex: 0, FillLampIndex: 1}
		eye := maths.Subspace{Origin: maths.V(0, -2, -2),
			Orientation: maths.LookAt(maths.V(0, -2, -2), maths.V(0, -10, -2))}
		picture, err := RenderWithInitialMedia(world, eye,
			film.Pinhole{Wide: 70 * maths.Degree}, time.Duration(0),
			width, height, []InitialMedium{{Index: index,
				ClearMedium: medium}})
		if err != nil {
			test.Fatal(err)
		}
		return picture
	}
	pictures = map[string]*frame.Frame{
		"water-surface-control": render(base),
		"water-diffuse-sky":     render(sampled),
	}
	return pictures["water-surface-control"].At(width/2, height/2),
		pictures["water-diffuse-sky"].At(width/2, height/2), pictures
}

// skyControlMost is the most light, by channel, the sky can scatter down
// a look in the control's water. Radiance over the square of the index
// is kept along a way, so no way in the water carries more than the
// index squared times the sky's brightest, its zenith's 0.5, 0.72 and
// 1. A phase adds up to one over the sphere, so a point scatters toward
// the eye at most its scattering times that; and a look gathers, of
// what its points scatter, at most the scattering over the extinction,
// 0.08/0.09, 0.1/0.115 and 0.12/0.145.
var skyControlMost = maths.RGB(
	skyControlIndex*skyControlIndex*0.5*0.08/0.09,
	skyControlIndex*skyControlIndex*0.72*0.1/0.115,
	skyControlIndex*skyControlIndex*1*0.12/0.145)

// underSkyControlMost is whether a lit pixel of the control is at most
// skyControlMost in every channel.
func underSkyControlMost(lit maths.Light) bool {
	return lit.R <= skyControlMost.R && lit.G <= skyControlMost.G &&
		lit.B <= skyControlMost.B
}

// TestOceanDiffuseSkyWaterOnlyControlIsBounded draws a synthetic
// water-only control and bounds its middle pixel: the bound is the most
// the sky can send, not the value it does send, so an error of a few
// times either way inside it passes. Without the sky's light scattered
// in the water the look meets nothing, the sky below the horizon is
// black and both lamps are of no light: the pixel is black, and covered
// whole, since an endless way of water keeps nothing of what is behind
// it. With it the pixel is lit in every channel, and by no more than
// skyControlMost.
//
// The pictures are written only where SLAYER_OCEAN_DIFFUSE_SKY_OUT names
// a folder; the pixels are bounded in every run.
func TestOceanDiffuseSkyWaterOnlyControlIsBounded(test *testing.T) {
	without, with, pictures := skyControlMiddles(test, 1)
	if without.Light() != (maths.Light{}) || without.Cover != 1 {
		test.Errorf("with no sky scattered the open water is %+v, want "+
			"black and covered", without)
	}
	lit := with.Light()
	if !(lit.R > 0 && lit.G > 0 && lit.B > 0) ||
		!underSkyControlMost(lit) {
		test.Errorf("with the sky scattered the open water is %v, want "+
			"lit and at most %v", lit, skyControlMost)
	}
	directory := os.Getenv("SLAYER_OCEAN_DIFFUSE_SKY_OUT")
	if directory == "" {
		return
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		test.Fatal(err)
	}
	for name, picture := range pictures {
		stem := filepath.Join(directory, name)
		if err := raw.Write(stem+".pfm", picture); err != nil {
			test.Fatal(err)
		}
		file, err := os.OpenFile(stem+".png",
			os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
		if err != nil {
			test.Fatal(err)
		}
		if err := png.Encode(file, colour.Tone{Exposure: 1, Knee: .8,
			Top: 2.2}.Develop(picture)); err != nil {
			_ = file.Close()
			test.Fatal(err)
		}
		if err := file.Close(); err != nil {
			test.Fatal(err)
		}
	}
	test.Logf("wrote the %dx%d water-only control to %s", skyControlWidth,
		skyControlHeight, directory)
}

// Anti: the scattered sky is the sky's and not a light of the water's
// own. The same control drawn with the sky scattered is brighter in
// blue than in red, as its sky is and its water scatters.
func TestOceanDiffuseSkyControlIsBluerThanRed(test *testing.T) {
	_, with, _ := skyControlMiddles(test, 1)
	lit := with.Light()
	if !(lit.B > lit.R) {
		test.Errorf("the scattered sky is %v, want more blue than red",
			lit)
	}
}

// skyControlWrongScale is how many times too bright a faulty control's
// sky is: a thousand, a sky scale wrong by three orders.
const skyControlWrongScale = 1000

// Anti: the bound turns a faulty control away. The same control drawn
// under a sky skyControlWrongScale times too bright, which the water's
// light is in proportion to, is lit over skyControlMost in some
// channel, where the true control is under it.
func TestOceanDiffuseSkyControlUnderAWrongSkyIsOverItsBound(
	test *testing.T) {
	_, honest, _ := skyControlMiddles(test, 1)
	_, wrong, _ := skyControlMiddles(test, skyControlWrongScale)
	if !underSkyControlMost(honest.Light()) {
		test.Errorf("the true control %v is over %v", honest.Light(),
			skyControlMost)
	}
	if underSkyControlMost(wrong.Light()) {
		test.Errorf("a sky %d times too bright gives %v, under %v",
			skyControlWrongScale, wrong.Light(), skyControlMost)
	}
}
