//go:build draw && darwin

package workspace

import (
	"image"
	"image/color"
	"image/draw"
	"os"
	"testing"

	"github.com/gogpu/gg"
	// Registers gg's GPU accelerator, as the ferruccio command does.
	_ "github.com/gogpu/gg/gpu"
	"github.com/gogpu/gpucontext"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
	"github.com/janearc/slayer/ferruccio/native/internal/headless"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
)

// world is the colour of the preview pane's image: a green no theme uses,
// so an overlay drawn beneath the image shows as green at its probe.
var world = color.RGBA{G: 200, A: 255}

// The preview pane fills the window in these tests.
const paneWidth, paneHeight = 800, 600

// overlayTolerance is how far, summed over the channels, a probe may sit
// from the overlay's colour; antialiasing and blending stay well inside it.
const overlayTolerance = 30

// metal is gg's GPU accelerator, which the blank import registers for the
// whole test binary. Left registered with no device, it makes the CPU tests
// drop their spheres and puts itself in software mode for good, so TestMain
// takes it out of use and only the Metal test puts it back.
var metal gg.GPUAccelerator

// TestMain runs every other test with no accelerator, as without the tag.
// The accelerator has drawn nothing yet, so closing it releases nothing.
func TestMain(m *testing.M) {
	metal = gg.Accelerator()
	gg.CloseAccelerator()
	os.Exit(m.Run())
}

// exportOnMetal sets up the play workspace with only its preview pane, the
// pane showing a uniform world image, puts gg's accelerator back and opens
// a real Metal device for the export to hand it. The accelerator is closed
// when the test ends, so the tests after it draw on the CPU.
func exportOnMetal(t *testing.T) (*Workspace, *headless.Device) {
	t.Helper()
	if err := gg.RegisterAccelerator(metal); err != nil {
		t.Fatalf("register gg's accelerator: %v", err)
	}
	device, err := headless.Open()
	if err != nil {
		t.Fatalf("open Metal: %v", err)
	}
	t.Cleanup(func() {
		gg.CloseAccelerator()
		device.Close()
	})
	work := playWorkspace(t)
	read, err := set.Read(work.state.path, []byte(work.state.document.Text()))
	if err != nil {
		t.Fatal(err)
	}
	work.working.scene, err = placement.NewDisplayCache().Scene(read)
	if err != nil {
		t.Fatal(err)
	}
	picture := image.NewRGBA(image.Rect(0, 0, paneWidth, paneHeight))
	draw.Draw(picture, picture.Bounds(), image.NewUniform(world), image.Point{}, draw.Src)
	work.viewer.set(picture)
	work.ui.SetRoot(work.viewer)
	work.ui.Window().HandleResize(paneWidth, paneHeight)
	work.ui.Window().Frame()
	return work, device
}

// TestOverlaysDrawAboveTheWorldImageOnMetal exports the preview pane on the
// real Metal device and checks each overlay is on top of the world image:
// the hover label, the orbit ball's card and the drag label. gg's GPU
// renderer groups draws by clip and, within a group, draws shapes before
// images; an overlay sharing the image's clip is drawn under it. The same
// probes are then drawn on the CPU, which keeps order, to prove each lands
// on its overlay. The CPU goes second, with gg's accelerator closed: a gg
// draw before the device is shared leaves the accelerator in software mode.
func TestOverlaysDrawAboveTheWorldImageOnMetal(t *testing.T) {
	work, device := exportOnMetal(t)
	ball := ballProbe(work.viewer)
	hover := map[string]image.Point{
		"hover label": {X: 420 + 18 + 6, Y: 80 + 22 + 18}, "orbit ball card": ball,
	}
	drag := map[string]image.Point{
		"drag label": {X: 100, Y: paneHeight - 10}, "orbit ball card": ball,
	}
	metal := mergeProbes(probeHovering(t, work, device, "metal", hover),
		probeDragging(t, work, device, "metal", drag))
	gg.CloseAccelerator()
	cpu := mergeProbes(probeHovering(t, work, nil, "cpu", hover),
		probeDragging(t, work, nil, "cpu", drag))
	ground := rgba(appearance.Ground)
	for name := range cpu {
		t.Logf("%s: cpu %v, metal %v", name, cpu[name], metal[name])
		if pixelColourDistance(cpu[name], ground) > overlayTolerance {
			t.Fatalf("%s probe misses its overlay on the CPU: %v", name, cpu[name])
		}
		if pixelColourDistance(metal[name], ground) > overlayTolerance {
			t.Errorf("%s is under the world image on Metal: %v, want %v",
				name, metal[name], ground)
		}
	}
}

// probeHovering exports the pane with the pointer over an object, showing
// the hover label, and samples the probes.
func probeHovering(t *testing.T, work *Workspace, provider gpucontext.DeviceProvider,
	backend string, probes map[string]image.Point) map[string]color.RGBA {
	t.Helper()
	work.viewer.hoverName, work.viewer.hoverPoint = "the ship", geometry.Pt(420, 80)
	defer func() { work.viewer.hoverName = "" }()
	return exportProbes(t, work, provider, backend, probes)
}

// probeDragging exports the pane mid-drag, showing the drag label, and
// samples the probes.
func probeDragging(t *testing.T, work *Workspace, provider gpucontext.DeviceProvider,
	backend string, probes map[string]image.Point) map[string]color.RGBA {
	t.Helper()
	view := work.viewer
	view.dragging = true
	view.dragStart, view.dragCurrent = geometry.Pt(300, 300), geometry.Pt(340, 310)
	defer func() { view.dragging = false }()
	return exportProbes(t, work, provider, backend, probes)
}

// exportProbes exports with the provider, insists the named backend drew
// it, and returns the colour at each probe.
func exportProbes(t *testing.T, work *Workspace, provider gpucontext.DeviceProvider,
	backend string, probes map[string]image.Point) map[string]color.RGBA {
	t.Helper()
	picture, err := work.exportWith(provider)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if drawn := work.ExportDrawn(); drawn.Backend != backend {
		t.Fatalf("export drawn by %+v; want %s", drawn, backend)
	}
	samples := map[string]color.RGBA{}
	for name, at := range probes {
		samples[name] = color.RGBAModel.Convert(picture.At(at.X, at.Y)).(color.RGBA)
	}
	return samples
}

// ballProbe is a point on the orbit ball's card, left of its sphere and
// level with the sphere's centre, where nothing else on the card is drawn.
func ballProbe(view *imageView) image.Point {
	center, _ := view.ball.disc(view.Bounds())
	card := view.ball.bounds(view.Bounds())
	// Six points in from the card's edge, clear of its outline.
	const inset = 6
	return image.Pt(int(card.Min.X)+inset, int(center.Y))
}

// mergeProbes joins two exports' samples, naming each by its export.
func mergeProbes(hovering, dragging map[string]color.RGBA) map[string]color.RGBA {
	merged := map[string]color.RGBA{}
	for name, sample := range hovering {
		merged["hovering: "+name] = sample
	}
	for name, sample := range dragging {
		merged["dragging: "+name] = sample
	}
	return merged
}

// rgba converts a theme colour to eight-bit channels.
func rgba(colour widget.Color) color.RGBA {
	r, g, b, a := colour.RGBA8()
	return color.RGBA{R: r, G: g, B: b, A: a}
}
