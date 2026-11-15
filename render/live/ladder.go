package live

import (
	"fmt"
	"image"
	"math"
	"time"

	"github.com/janearc/slayer/interfaces/terminal"
)

// pane is the part of the screen the picture is shown in: the screen less
// the foot and the status row, in the screen's sextant pixels, and the
// shape of one of those pixels, its width over its height.
type pane struct {
	width, height int
	pixel         float64
}

// paneOf is the pane of a screen. A screen too short for the foot keeps
// one row of picture, and one that says no pixel shape has square ones.
func paneOf(screen terminal.Screen) pane {
	down := max(one, screen.H/max(one, screen.Rows))
	rows := max(one, screen.Rows-FootRows-statusRows)
	pixel := screen.Pixel
	if pixel <= none {
		pixel = one
	}
	return pane{width: max(one, screen.W), height: rows * down,
		pixel: pixel}
}

// placing is where a picture sits in the pane: the rectangle it fills in
// the pane's pixels, and the size it is drawn at in square pixels so
// that, laid on the pane's pixels, it keeps its shape.
type placing struct {
	shown         image.Rectangle
	width, height int
	paneW, paneH  int
}

// placed fits a picture of a shape into the pane, keeping its shape: as
// large as fits, centred, with bars where it does not fill.
func placed(shapeW, shapeH int, at pane) placing {
	shapeW, shapeH = max(one, shapeW), max(one, shapeH)
	scale := math.Min(float64(at.width)*at.pixel/float64(shapeW),
		float64(at.height)/float64(shapeH))
	drawnW := max(one, rounded(float64(shapeW)*scale))
	drawnH := max(one, min(at.height, rounded(float64(shapeH)*scale)))
	shownW := max(one, min(at.width, rounded(float64(drawnW)/at.pixel)))
	left, top := (at.width-shownW)/pair, (at.height-drawnH)/pair
	return placing{shown: image.Rect(left, top, left+shownW, top+drawnH),
		width: drawnW, height: drawnH, paneW: at.width, paneH: at.height}
}

// rounded is a number to the nearest whole.
func rounded(value float64) int { return int(math.Floor(value + half)) }

// rung is one step of the ladder: the size it draws at and its words for
// the status row.
type rung struct {
	size  Size
	words string
}

// ladderOf is the rungs for a picture drawn width by height: smaller
// first, each a divisor of the size at one ray, then the whole size at
// more rays. A rung the same as the one before is left out.
func ladderOf(width, height int) []rung {
	var rungs []rung
	for _, divisor := range divisors {
		size := Size{width: shrunk(width, divisor),
			height: shrunk(height, divisor), rays: one}
		if len(rungs) > none && rungs[len(rungs)-one].size == size {
			continue
		}
		rungs = append(rungs, rung{size: size, words: sizeWords(divisor) +
			", " + raysWords(one)})
	}
	for _, rays := range finerRays {
		rungs = append(rungs, rung{size: Size{width: width, height: height,
			rays: rays}, words: sizeWords(one) + ", " + raysWords(rays)})
	}
	return rungs
}

// shrunk is a side divided, rounded up, and never below one.
func shrunk(side, divisor int) int {
	return max(one, (side+divisor-one)/divisor)
}

// sizeWords says a rung's size as a share of the pane's.
func sizeWords(divisor int) string {
	if divisor == one {
		return "pane size"
	}
	return fmt.Sprintf("1/%d size", divisor)
}

// raysWords says a count of rays.
func raysWords(rays int) string {
	if rays == one {
		return "1 ray"
	}
	return fmt.Sprintf("%d rays", rays)
}

// predicted is how long a rung will take, from how long the one before
// took: scaled by its pixels, and by its rays squared, as rays are
// counted along each side of a pixel.
func predicted(before rung, took time.Duration, next rung) time.Duration {
	pixels := float64(next.size.width*next.size.height) /
		float64(max(one, before.size.width*before.size.height))
	rays := float64(next.size.rays) / float64(max(one, before.size.rays))
	return time.Duration(float64(took) * pixels * rays * rays)
}

// admitted is whether a rung may be started after one that took took:
// its predicted time is within the ceiling.
func admitted(before rung, took time.Duration, next rung) bool {
	return predicted(before, took, next) <= Ceiling
}
