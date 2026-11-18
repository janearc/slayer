package workspace

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/gogpu/gg"
	"github.com/gogpu/ui/app"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/render"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/native/internal/editor"
)

func TestProofGestureDispatchesThroughRealWindow(t *testing.T) {
	for _, test := range []struct {
		name string
		mode string
		want [3]float64
	}{
		{name: "orbit", mode: "orbit", want: [3]float64{-35, 14, 0}},
		{name: "zoom", mode: "zoom", want: [3]float64{0, 0, math.Exp(-0.3)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			ui := app.New()
			view := configuredNavigableImage()
			view.SetBounds(geometry.Rect{Min: geometry.Pt(300, 180), Max: geometry.Pt(700, 580)})
			var calls [][3]float64
			view.SetNavigateHandler(func(yaw, pitch, zoom float64) { calls = append(calls, [3]float64{yaw, pitch, zoom}) })
			ui.SetRoot(view)
			canvasContext := gg.NewContext(800, 640)
			defer canvasContext.Close()
			canvas := render.NewCanvas(canvasContext, 800, 640)
			widget.StampScreenOrigin(view, canvas)
			view.Draw(ui.Window().Context(), canvas)
			if err := canvasContext.FlushGPU(); err != nil {
				t.Fatal(err)
			}
			if !view.IsScreenOriginValid() {
				t.Fatal("real draw did not stamp screen origin")
			}
			work := &Workspace{ui: ui, viewer: view}
			if err := work.ProofGesture(test.mode); err != nil {
				t.Fatal(err)
			}
			if len(calls) != 1 || calls[0] != test.want {
				t.Fatalf("window-dispatched %s calls = %v, want %v", test.mode, calls, test.want)
			}
		})
	}
}

func TestProofPreviewsEqualComparesRetainedPixels(t *testing.T) {
	directory := t.TempDir()
	first := filepath.Join(directory, "first.png")
	second := filepath.Join(directory, "second.png")
	third := filepath.Join(directory, "third.png")
	writeProofPNG(t, first, color.RGBA{R: 190, G: 80, B: 50, A: 255})
	writeProofPNG(t, second, color.RGBA{R: 190, G: 80, B: 50, A: 255})
	writeProofPNG(t, third, color.RGBA{R: 190, G: 80, B: 51, A: 255})
	if equal, err := ProofPreviewsEqual(first, second); err != nil || !equal {
		t.Fatalf("identical renderer output comparison = %t, %v", equal, err)
	}
	if equal, err := ProofPreviewsEqual(first, third); err != nil || equal {
		t.Fatalf("different renderer output comparison = %t, %v", equal, err)
	}
}

func writeProofPNG(t *testing.T, path string, fill color.RGBA) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	picture := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			picture.SetRGBA(x, y, fill)
		}
	}
	if err := png.Encode(file, picture); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestProofUndoUsesFocusedPreviewCommandZ(t *testing.T) {
	ui := app.New()
	edit := editor.New("original", nil)
	edit.ApplyText("camera changed")
	view := configuredNavigableImage()
	view.SetNavigateHandler(func(float64, float64, float64) {})
	view.SetUndoHandler(func(ctx widget.Context, redo bool) {
		if redo {
			edit.Redo(ctx)
		} else {
			edit.Undo(ctx)
		}
	})
	ui.SetRoot(view)
	center := geometry.Pt(50, 50)
	ui.HandleEvent(event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, center, center, event.ModNone))
	ui.HandleEvent(event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, center, center, event.ModNone))
	work := &Workspace{ui: ui, editor: edit, viewer: view}
	if err := work.ProofUndo("original"); err != nil {
		t.Fatal(err)
	}
	if edit.Text() != "original" || !view.IsFocused() || edit.IsFocused() {
		t.Fatalf("proof undo text/view focus/editor focus = %q/%v/%v", edit.Text(), view.IsFocused(), edit.IsFocused())
	}
}

func TestProofGestureRejectsUnknownOrUnavailableViewer(t *testing.T) {
	ui := app.New()
	view := newImageView()
	view.SetBounds(geometry.Rect{Min: geometry.Pt(0, 0), Max: geometry.Pt(100, 100)})
	ui.SetRoot(view)
	work := &Workspace{ui: ui, viewer: view}
	if err := work.ProofGesture("orbit"); err == nil {
		t.Fatal("proof gesture accepted a non-navigable preview")
	}
	if err := work.ProofGesture("spin"); err == nil {
		t.Fatal("proof gesture accepted an unknown gesture")
	}
}

var _ widget.Widget = (*imageView)(nil)
