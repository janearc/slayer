package workingview

import (
	"image"
	"image/color"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/session/placement"
)

func TestWorkingSkyFillsOnlyClearPixels(t *testing.T) {
	picture := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	foreground := color.NRGBA{R: 12, G: 34, B: 56, A: 255}
	picture.SetNRGBA(1, 0, foreground)
	scene := testSkyScene()
	fillSky(picture, scene)
	if pixel := picture.NRGBAAt(1, 0); pixel != foreground {
		t.Fatalf("sky changed the foreground pixel: %v", pixel)
	}
	top, bottom := picture.NRGBAAt(0, 0), picture.NRGBAAt(0, 3)
	if top.A != 255 || top.B == 0 || bottom.A != 255 || bottom.B != 0 || bottom.R != 255 {
		t.Fatalf("clear pixels did not receive the gradient: %v, %v", top, bottom)
	}
}

func TestWorkingSkyLeavesUnsupportedBackgroundAlone(t *testing.T) {
	flat := color.NRGBA{R: 12, G: 34, B: 56, A: 255}
	picture := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	picture.SetNRGBA(0, 0, flat)
	scene := testSkyScene()
	scene.SkyGradient = nil
	fillSky(picture, scene)
	if pixel := picture.NRGBAAt(0, 0); pixel != flat {
		t.Fatalf("unsupported sky changed the fallback background: %v", pixel)
	}
}

func BenchmarkWorkingSkyClearFrame(b *testing.B) {
	picture := image.NewNRGBA(image.Rect(0, 0, 1600, 1000))
	scene := testSkyScene()
	b.ReportAllocs()
	for b.Loop() {
		clear(picture.Pix)
		fillSky(picture, scene)
	}
}

func testSkyScene() *placement.DisplayScene {
	return &placement.DisplayScene{
		Eye: maths.Local(), Lens: film.Pinhole{Wide: 60 * maths.Degree},
		Width: 4, Height: 4,
		SkyGradient: placement.WorkingSkyGradient(sky.Gradient{
			Horizon: maths.RGB(.5, .5, .5), Zenith: maths.RGB(.3, .5, 1.4), Ground: maths.RGB(1, 0, 0),
		}),
	}
}
