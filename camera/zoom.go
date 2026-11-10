package camera

import (
	"fmt"
	"time"

	"github.com/janearc/slayer/maths"
)

// Zoom is a checked change of horizontal field of view, in degrees.
// It changes the lens without moving the camera or changing exposure.
type Zoom struct {
	from, to float64
	duration time.Duration
	ease     string
}

// NewZoom accepts finite pinhole fields of view below a straight angle.
// Empty easing means linear interpolation of the angle, not focal length.
func NewZoom(from, to float64, duration time.Duration,
	ease string,
) (*Zoom, error) {
	if !zoomAngle(from) || !zoomAngle(to) || duration <= nothing ||
		duration > maxDuration {
		return nil, fmt.Errorf("camera: zoom needs finite fields of " +
			"view above zero and below 180 degrees, " +
			"and a positive duration")
	}
	if ease != "" && ease != "linear" && ease != "smooth" {
		return nil, fmt.Errorf("camera: zoom ease must be " +
			"linear or smooth")
	}
	return &Zoom{from: from, to: to, duration: duration, ease: ease}, nil
}

// At samples elapsed scene time and holds the exact endpoints outside the zoom.
func (zoom *Zoom) At(elapsed time.Duration) (float64, error) {
	if zoom == nil || zoom.duration <= nothing {
		return nothing, fmt.Errorf("camera: no checked zoom")
	}
	if elapsed <= nothing {
		return zoom.from, nil
	}
	if elapsed >= zoom.duration {
		return zoom.to, nil
	}
	share := float64(elapsed) / float64(zoom.duration)
	if zoom.ease == "smooth" {
		share = maths.Smooth(nothing, whole, share)
	}
	return zoom.from + (zoom.to-zoom.from)*share, nil
}

// zoomAngle also rejects values whose conversion would collapse to zero.
func zoomAngle(degrees float64) bool {
	const straightAngleDegrees = 180
	return number(degrees) && degrees > nothing &&
		degrees < straightAngleDegrees &&
		degrees*maths.Degree/2 > nothing
}
