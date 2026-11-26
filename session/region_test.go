package session

import (
	"context"
	"image"
	"strings"
	"testing"

	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

func TestCPURegionDrawsTheFullFramesPixels(t *testing.T) {
	full := usualAsked()
	full.width, full.height, full.samples = 12, 10, 2
	keeping := trace.NewKeeping(context.Background(), nil)
	full.keeping = &keeping
	whole, _, err := drawMeasured(thing.NewWorld(), full, 0)
	if err != nil {
		t.Fatal(err)
	}
	box := image.Rect(2, 3, 7, 8)
	partAsk := full
	partAsk.region = &box
	part, _, err := drawMeasured(thing.NewWorld(), partAsk, 0)
	if err != nil {
		t.Fatal(err)
	}
	if part.W != box.Dx() || part.H != box.Dy() {
		t.Fatalf("region is %dx%d", part.W, part.H)
	}
	for y := 0; y < part.H; y++ {
		for x := 0; x < part.W; x++ {
			if part.At(x, y) != whole.At(box.Min.X+x, box.Min.Y+y) {
				t.Fatalf("pixel %d,%d differs", x, y)
			}
		}
	}
}

func TestRegionNamesTheRefusedShortcut(t *testing.T) {
	box := image.Rect(1, 1, 3, 3)
	given := usualAsked()
	given.width, given.height, given.region, given.restir = 8, 8, &box, true
	if err := regionRefusal(given); err == nil || !strings.Contains(err.Error(), "restir") {
		t.Fatalf("restir refusal: %v", err)
	}
}

// Anti: unsupported modes and a region outside the resolved size are refused.
func TestRegionRefusesUnsupportedDraws(t *testing.T) {
	box := image.Rect(1, 1, 3, 3)
	for name, mutate := range map[string]func(*asked){
		"native gpu":     func(a *asked) { a.onGPU, a.nativeMeshes = true, true },
		"native shadows": func(a *asked) { a.onGPU, a.nativeShadows = true, true },
		"wire":           func(a *asked) { a.wire = true },
		"aperture":       func(a *asked) { a.optics.aperture = 1 },
		"restir":         func(a *asked) { a.restir = true },
		"hud":            func(a *asked) { a.overlay = &overlay{} },
		"initial media":  func(a *asked) { a.initialMedia = []trace.InitialMedium{{Index: 1}} },
		"outside":        func(a *asked) { a.width = 2 },
	} {
		given := usualAsked()
		given.width, given.height, given.region = 8, 8, &box
		mutate(&given)
		if err := regionRefusal(given); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

func TestRegionAcceptsSoftwareGPU(t *testing.T) {
	box := image.Rect(1, 1, 3, 3)
	given := usualAsked()
	given.width, given.height, given.region, given.onGPU = 8, 8, &box, true
	if err := regionRefusal(given); err != nil {
		t.Fatal(err)
	}
}

// A resized ask cannot make the region lie outside the resolved frame.
func TestAResizedAskRefusesAnOutsideRegion(t *testing.T) {
	box := image.Rect(3, 3, 7, 7)
	session := &Session{given: asked{width: 10, height: 10, region: &box}}
	_, err := session.asking(Ask{width: 6})
	if err == nil {
		t.Fatal("a region outside the resized frame was accepted")
	}
}

// Anti: a region inside a resized frame is not refused by bounds checking.
func TestAResizedAskAcceptsAnInsideRegion(t *testing.T) {
	box := image.Rect(3, 3, 7, 7)
	session := &Session{given: asked{width: 10, height: 10, region: &box}}
	if _, err := session.asking(Ask{width: 7}); err != nil {
		t.Fatal(err)
	}
}

// Anti: region checks leave a whole-picture job usable.
func TestAWholePictureIsNotRefusedAsARegion(t *testing.T) {
	wanted := jobFrom(t, "scene: {smoke: 1}\nfilm: {width: 10, height: 10}\n")
	if _, err := jobAsked(wanted); err != nil {
		t.Fatalf("a whole picture was refused: %v", err)
	}
}
