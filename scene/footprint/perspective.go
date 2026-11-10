// Package footprint assesses bounded geometry through an explicit perspective
// view. It does not infer terrain extrema, lens optics or a world's height
// datum.
package footprint

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// Perspective retains a checked pinhole pose, field and raster geometry.
// Camera local axes are right, up and back, as in film.Pinhole.
type Perspective struct {
	eye                        maths.Subspace
	horizontal, vertical, near float64
	focalAcross, focalDown     float64
}

// New uses the actual horizontal field and pixel width/height ratio.
// Quantities must be explicit: no default FOV, pixel aspect or near distance.
// Pose scale must be zero (identity) or one; a camera is not a scaled world.
func New(eye maths.Subspace, field measure.Angle, width, height int,
	pixelAspect float64, near measure.Length,
) (Perspective, error) {
	angle, distance := field.InRadians(), near.InMetres()
	if !finitePoint(eye.Origin) || !validOrientation(eye.Orientation) ||
		(eye.Scale != 0 && eye.Scale != 1) || width < 1 || height < 1 ||
		!finite(angle) || angle <= 0 || angle >= math.Pi ||
		!finite(pixelAspect) || pixelAspect <= 0 || !finite(distance) || distance <= 0 {
		return Perspective{}, fmt.Errorf("footprint: invalid perspective pose, field or raster")
	}
	horizontal := math.Tan(angle / 2)
	aspect := float64(width) / float64(height) * pixelAspect
	view := Perspective{eye: eye, horizontal: horizontal,
		vertical: horizontal / aspect, near: distance,
		focalAcross: float64(width) / (2 * horizontal),
		focalDown:   float64(width) * pixelAspect / (2 * horizontal)}
	for _, value := range []float64{aspect, view.horizontal, view.vertical, view.focalAcross, view.focalDown} {
		if !finite(value) || value <= 0 {
			return Perspective{}, fmt.Errorf("footprint: perspective exceeds the numeric range")
		}
	}
	return view, nil
}

// validOrientation accepts the documented zero orientation as identity.
// The tolerance is numerical: sqrt(binary64 epsilon), not a scene angle.
func validOrientation(orientation maths.Orientation) bool {
	if orientation == (maths.Orientation{}) {
		return true
	}
	allowance := math.Sqrt(math.Nextafter(1, 2) - 1)
	axes := [3]maths.Vec{orientation.X, orientation.Y, orientation.Z}
	for index, axis := range axes {
		if !finitePoint(axis) || math.Abs(axis.Dot(axis)-1) > allowance {
			return false
		}
		for _, other := range axes[index+1:] {
			if math.Abs(axis.Dot(other)) > allowance {
				return false
			}
		}
	}
	return orientation.X.Cross(orientation.Y).Dot(orientation.Z) > 0
}

// finite rejects invalid arithmetic before culling or budget selection.
func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

// finitePoint does not square large world coordinates to validate them.
func finitePoint(point maths.Vec) bool { return finite(point.X) && finite(point.Y) && finite(point.Z) }
