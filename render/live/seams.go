package live

import (
	"context"
	"fmt"
	"image"
	"time"

	"github.com/janearc/slayer/interfaces/terminal"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
)

// The seams the view draws and shows through. Each is the least the view
// needs of a piece that already exists: the door hands it a
// session and a terminal.Live, and the tests hand it mocks made by
// mockgen (seams_mock_test.go).

// Drawn is one frame's light and its own hud, as session.Drawn gives
// them.
type Drawn interface {
	Light() *frame.Frame
	Overlay(picture *image.RGBA) error
}

// Stage is a job's scene staged once, as a session is: it draws
// any of the job's frames at any size and rays.
type Stage interface {
	// Frame draws frame index at the size asked; a zero size is the
	// job's own.
	Frame(ctx context.Context, index int, size Size) (Drawn, error)
	// Times is each frame's scene time; a still has one.
	Times() []time.Duration
	// Tone develops the light where the job says no print and no stock.
	Tone() colour.Tone
	// Shape is the size a frame takes when asked none: the job's film,
	// or the scene's own.
	Shape() (width, height int)
	// Exposure is the exposure the scene was staged at.
	Exposure() float64
	// Cameras is the scene's cameras by name, in its own order; none
	// for a scene with no names.
	Cameras() []string
	Close() error
}

// Stager stages a job's scene. A job it cannot draw is refused by name.
type Stager interface {
	Stage(ctx context.Context, wanted job.Job) (Stage, error)
}

// Screen is the terminal the view is shown on, as terminal.Live is.
type Screen interface {
	Events() <-chan terminal.Event
	Screen() terminal.Screen
	Present(cells *terminal.Cells) error
	// Foot writes lines on the screen's last rows, the last on the
	// last row.
	Foot(lines []string) error
}

// Size is what a rung asks of a frame: a width and a height in pixels
// and rays along each side of a pixel. A zero is the job's own.
type Size struct {
	width, height, rays int
}

// NewSize is a size, refused when a part is below nought.
func NewSize(width, height, rays int) (Size, error) {
	if width < none || height < none || rays < none {
		return Size{}, fmt.Errorf("live: a size of %d by %d at %d rays; "+
			"each is nought for the job's own, or more", width, height,
			rays)
	}
	return Size{width: width, height: height, rays: rays}, nil
}

// Width is the picture's width in pixels; nought is the job's.
func (size Size) Width() int { return size.width }

// Height is the picture's height in pixels; nought is the job's.
func (size Size) Height() int { return size.height }

// Rays is the rays along each side of a pixel; nought is the job's.
func (size Size) Rays() int { return size.rays }
