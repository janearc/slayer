package gpuocean

import (
	"errors"
	"math"

	"github.com/janearc/slayer/oceantex"
)

// adjacentAngle is the angle in radians between ray and adjacent, or pi over
// three when the grid is one column wide. It refuses a direction of zero or
// non-finite length, and two directions that are the same.
func adjacentAngle(width int, ray, adjacent oceantex.Ray) (float64, error) {
	if width == 1 {
		return math.Pi / 3, nil
	}
	left, right := ray.Dir, adjacent.Dir
	leftLength := math.Sqrt(left.X*left.X + left.Y*left.Y + left.Z*left.Z)
	rightLength := math.Sqrt(right.X*right.X + right.Y*right.Y + right.Z*right.Z)
	if !rayFinite(leftLength) || !rayFinite(rightLength) || leftLength == 0 || rightLength == 0 {
		return 0, errors.New("gpuocean: invalid adjacent ray direction")
	}
	dot := (left.X*right.X + left.Y*right.Y + left.Z*right.Z) / (leftLength * rightLength)
	if !rayFinite(dot) || dot < -1.00001 || dot > 1.00001 {
		return 0, errors.New("gpuocean: invalid adjacent ray direction")
	}
	angle := math.Acos(math.Min(1, math.Max(-1, dot)))
	if angle <= 0 {
		return 0, errors.New("gpuocean: adjacent rays have no representable angular separation")
	}
	return angle, nil
}

// MarchRays pairs each ray of a camera's width by height grid with PixelAngle,
// the angle in radians to the ray beside it in the same row (the one to the
// left for the last column, pi over three for a one column grid). It refuses a
// bad grid, a ray that is not unit length, and a pair with no angle between.
func MarchRays(camera RayCamera, rays []oceantex.Ray) ([]MarchRay, error) {
	if camera.Width <= 0 || camera.Height <= 0 || camera.Width > MaxRayPixels/camera.Height || len(rays) != camera.Width*camera.Height || len(rays) > MaxSamples {
		return nil, errors.New("gpuocean: invalid ray grid")
	}
	result := make([]MarchRay, len(rays))
	for row := 0; row < camera.Height; row++ {
		for column := 0; column < camera.Width; column++ {
			index := row*camera.Width + column
			direction := rays[index].Dir
			length := direction.X*direction.X + direction.Y*direction.Y + direction.Z*direction.Z
			if !rayFinite(length) || math.Abs(length-1) > 1e-5 {
				return nil, errors.New("gpuocean: ray direction must be unit length")
			}
			beside := column + 1
			if beside >= camera.Width {
				beside = column - 1
			}
			angle := math.Pi / 3
			if camera.Width > 1 {
				other := rays[row*camera.Width+beside].Dir
				dot := direction.X*other.X + direction.Y*other.Y + direction.Z*other.Z
				if !rayFinite(dot) || dot < -1.00001 || dot > 1.00001 {
					return nil, errors.New("gpuocean: invalid adjacent ray direction")
				}
				angle = math.Acos(math.Min(1, math.Max(-1, dot)))
				if angle <= 0 {
					return nil, errors.New("gpuocean: adjacent rays have no representable angular separation")
				}
			}
			result[index] = MarchRay{Ray: rays[index], PixelAngle: angle}
		}
	}
	return result, nil
}
