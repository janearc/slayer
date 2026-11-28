package workspace

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
)

// ProofGesture drives the preview through GoGPU's real Window event dispatcher.
func (work *Workspace) ProofGesture(gesture string) error {
	if gesture != "orbit" && gesture != "zoom" && gesture != "move" && gesture != "turn" {
		return fmt.Errorf("unknown proof gesture %q", gesture)
	}
	if work.ui == nil || work.viewer == nil || !work.viewer.navigable || work.viewer.source == nil || work.viewer.navigate == nil {
		return fmt.Errorf("proof gesture requires a rendered navigable preview")
	}
	bounds := work.viewer.ScreenBounds()
	center := geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2)
	if !finitePoint(center) || bounds.Width() < 1 || bounds.Height() < 1 {
		return fmt.Errorf("proof gesture requires a laid out preview")
	}
	if work.composition == "play" {
		if gesture == "orbit" {
			work.setPlayMode("look")
		}
		if gesture == "move" || gesture == "turn" {
			work.setPlayMode(gesture)
		}
	}
	switch gesture {
	case "orbit", "move", "turn":
		work.ui.HandleEvent(event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, center, center, event.ModNone))
		end := geometry.Pt(center.X+100, center.Y+40)
		work.ui.HandleEvent(event.NewMouseEvent(event.MouseMove, event.ButtonNone, event.ButtonStateLeft, end, end, event.ModNone))
		work.ui.HandleEvent(event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, end, end, event.ModNone))
	case "zoom":
		deltaY := float32(-50)
		if work.state.document != nil {
			if raw, err := work.state.document.Value("wide"); err == nil {
				if wide, err := strconv.ParseFloat(raw, 64); err == nil && wide*math.Exp(-0.3) < 5 {
					deltaY = 50
				}
			}
		}
		work.ui.HandleEvent(event.NewWheelEvent(geometry.Pt(0, deltaY), center, center, event.ModNone))
	}
	return nil
}

// ProofSourceCameraEdit makes the source edit used by the Observe undo proof.
func (work *Workspace) ProofSourceCameraEdit() error {
	if work.composition != "observe" || work.ui == nil || work.editor == nil || work.viewer == nil {
		return fmt.Errorf("undo proof needs the Observe source editor")
	}
	candidate := *work.state.document
	if err := candidate.Navigate(10, 0, 0); err != nil {
		return err
	}
	if candidate.Matches(work.state.document.Text()) {
		return fmt.Errorf("the proof source camera edit did not change the YAML")
	}
	work.editor.ApplyText(candidate.Text())
	work.needsCheck = false
	work.pending.remember(work.state, work.specimen, time.Now())
	work.wakeNavigation(navigationDelay)
	work.ui.Window().Context().RequestFocus(work.viewer)
	return nil
}

// ProofUndo sends Command-Z to the already focused preview through GoGPU's dispatcher.
func (work *Workspace) ProofUndo(expected string) error {
	if work.ui == nil || work.editor == nil || work.viewer == nil {
		return fmt.Errorf("proof undo requires the YAML editor")
	}
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyZ, 'z', event.ModSuper))
	if work.editor.Text() != expected {
		return fmt.Errorf("Command-Z did not restore the pre-gesture draft")
	}
	return nil
}

// ProofText returns the currently visible YAML draft for the proof receipt.
func (work *Workspace) ProofText() string {
	if work.editor == nil {
		return ""
	}
	return work.editor.Text()
}

// ProofPreviewsEqual compares decoded renderer pixels for an undo proof.
func ProofPreviewsEqual(before, after string) (bool, error) {
	first, err := decodeProofPreview(before)
	if err != nil {
		return false, err
	}
	second, err := decodeProofPreview(after)
	if err != nil {
		return false, err
	}
	if first.Bounds() != second.Bounds() {
		return false, nil
	}
	for y := first.Bounds().Min.Y; y < first.Bounds().Max.Y; y++ {
		for x := first.Bounds().Min.X; x < first.Bounds().Max.X; x++ {
			if color.RGBAModel.Convert(first.At(x, y)) != color.RGBAModel.Convert(second.At(x, y)) {
				return false, nil
			}
		}
	}
	return true, nil
}

// decodeProofPreview opens one retained renderer image without modifying it.
func decodeProofPreview(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open proof preview %s: %w", path, err)
	}
	defer file.Close()
	picture, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("decode proof preview %s: %w", path, err)
	}
	return picture, nil
}

// proofWorkingPoints uses the actual laid-out widget coordinates from native drawing.
func (work *Workspace) proofWorkingPoints() (geometry.Point, geometry.Point, error) {
	if work.composition != "play" || work.ui == nil || work.viewer == nil || work.viewer.source == nil || !work.viewer.IsScreenOriginValid() {
		return geometry.Point{}, geometry.Point{}, fmt.Errorf("held working proof needs a displayed native Play viewport")
	}
	bounds := work.viewer.ScreenBounds()
	center := geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2)
	if !finitePoint(center) || bounds.Width() < 1 || bounds.Height() < 1 {
		return geometry.Point{}, geometry.Point{}, fmt.Errorf("working proof viewport has no layout")
	}
	return center, geometry.Pt(center.X+100, center.Y+40), nil
}

// ProofHover moves the pointer, with no button held, to the middle of the
// Play viewport, as a hand would, and returns the object it names there.
func (work *Workspace) ProofHover() (string, error) {
	center, _, err := work.proofWorkingPoints()
	if err != nil {
		return "", err
	}
	work.ui.HandleEvent(event.NewMouseEvent(event.MouseMove, event.ButtonNone, 0, center, center, event.ModNone))
	if work.viewer.hoverName == "" {
		return "", fmt.Errorf("hover proof found no object under the pointer")
	}
	return work.viewer.hoverName, nil
}

// ProofWorkingPress begins a native pointer capture without releasing it.
func (work *Workspace) ProofWorkingPress(kind string) error {
	center, _, err := work.proofWorkingPoints()
	if err != nil {
		return err
	}
	mode := "move"
	if kind == "working-look" {
		mode = "look"
	} else if kind != "working-move" {
		return fmt.Errorf("unknown held working gesture %q", kind)
	}
	work.setPlayMode(mode)
	work.ui.HandleEvent(event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, center, center, event.ModNone))
	return nil
}

// ProofWorkingMove dispatches a held native motion on a later UI tick.
func (work *Workspace) ProofWorkingMove(kind string) error {
	_, end, err := work.proofWorkingPoints()
	if err != nil {
		return err
	}
	work.ui.HandleEvent(event.NewMouseEvent(event.MouseMove, event.ButtonNone, event.ButtonStateLeft, end, end, event.ModNone))
	return nil
}

// ProofWorkingRelease commits the held gesture after its GPU frame was captured.
func (work *Workspace) ProofWorkingRelease() error {
	_, end, err := work.proofWorkingPoints()
	if err != nil {
		return err
	}
	work.ui.HandleEvent(event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, end, end, event.ModNone))
	return nil
}
