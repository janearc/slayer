package workspace

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"github.com/gogpu/gg"
	"github.com/gogpu/ui/render"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
)

// TestCameraBallsPaintVisibleSpheres checks pixels through the real window paint path.
func TestCameraBallsPaintVisibleSpheres(t *testing.T) {
	work := playWorkspace(t)
	read, err := set.Read(work.state.path, []byte(work.state.document.Text()))
	if err != nil {
		t.Fatal(err)
	}
	work.working.scene, err = placement.NewDisplayCache().Scene(read)
	if err != nil {
		t.Fatal(err)
	}
	photo := image.NewRGBA(image.Rect(0, 0, 800, 600))
	draw.Draw(photo, photo.Bounds(), image.NewUniform(color.RGBA{R: 189, G: 157, B: 103, A: 255}), image.Point{}, draw.Src)
	work.viewer.set(photo)
	work.ui.SetRoot(work.viewer)
	work.ui.Window().HandleResize(800, 600)
	work.ui.Window().Frame()
	context := gg.NewContext(800, 600)
	defer context.Close()
	work.ui.Window().DrawTo(render.NewCanvas(context, 800, 600))
	if err := context.FlushGPU(); err != nil {
		t.Fatal(err)
	}
	pixels := context.Image()
	for _, entry := range []struct {
		name string
		ball *cameraBall
	}{{"Orbit", work.viewer.ball}, {"Travel", work.viewer.travel}} {
		t.Run(entry.name, func(t *testing.T) {
			if entry.ball == nil {
				t.Fatal("instrument absent")
			}
			center, radius := entry.ball.disc(work.viewer.Bounds())
			cx, cy, r := int(center.X), int(center.Y), int(radius)
			// Compare the sphere interior to its own card, excluding title/footer labels.
			card := color.RGBAModel.Convert(pixels.At(cx+76, cy)).(color.RGBA)
			changed := 0
			for y := cy - r + 10; y <= cy+r-10; y++ {
				for x := cx - r + 10; x <= cx+r-10; x++ {
					if (x-cx)*(x-cx)+(y-cy)*(y-cy) > (r-10)*(r-10) {
						continue
					}
					sample := color.RGBAModel.Convert(pixels.At(x, y)).(color.RGBA)
					if pixelColourDistance(sample, card) > 18 {
						changed++
					}
				}
			}
			if changed < 200 {
				t.Fatalf("%s sphere invisible: only%d interior pixels differ from card %v; center=%v bounds=%v", entry.name, changed, card, center, entry.ball.bounds(work.viewer.Bounds()))
			}
		})
	}
}

func pixelColourDistance(a, b color.RGBA) int {
	absolute := func(v int) int {
		if v < 0 {
			return -v
		}
		return v
	}
	return absolute(int(a.R)-int(b.R)) + absolute(int(a.G)-int(b.G)) + absolute(int(a.B)-int(b.B))
}
